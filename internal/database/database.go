// Package database owns the SQLite connection, migrations, and persisted
// settings. It is the single place that touches the underlying database.
package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Open connects to the SQLite database at path, enabling WAL mode and foreign
// keys. ":memory:" is supported for tests.
func Open(path string) (*gorm.DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create database dir: %w", err)
		}
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	// A single connection keeps SQLite writes serialized and avoids lock
	// contention while the worker pool batches inserts.
	sqlDB.SetMaxOpenConns(1)
	if path != ":memory:" {
		if err := db.Exec("PRAGMA journal_mode=WAL;").Error; err != nil {
			return nil, fmt.Errorf("enable wal: %w", err)
		}
		if err := db.Exec("PRAGMA synchronous=NORMAL;").Error; err != nil {
			return nil, fmt.Errorf("set synchronous: %w", err)
		}
	}
	if err := db.Exec("PRAGMA foreign_keys=ON;").Error; err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	return db, nil
}

// Migrate creates all tables, indexes, and the FTS5 content-search table.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Repository{},
		&models.File{},
		&models.Commit{},
		&models.Branch{},
		&models.Tag{},
		&models.Contributor{},
		&models.FileOwnership{},
		&models.Dependency{},
		&models.ImportEdge{},
		&models.DuplicateGroup{},
		&models.DuplicateBlock{},
		&models.Job{},
		&models.Setting{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	if err := ensureFileFTS(db); err != nil {
		return err
	}
	return nil
}

// ensureFileFTS creates the FTS5 table that backs content search. rowid maps
// to files.id so the search layer can join for metadata.
func ensureFileFTS(db *gorm.DB) error {
	const ddl = `CREATE VIRTUAL TABLE IF NOT EXISTS file_fts USING fts5(
		repo_id UNINDEXED,
		path UNINDEXED,
		content,
		tokenize = 'unicode61'
	)`
	if err := db.Exec(ddl).Error; err != nil {
		return fmt.Errorf("create file_fts: %w", err)
	}
	return nil
}

// ClearRepoData removes every analysis row and index entry for a repository.
// It is called before a rescan so runs are idempotent.
func ClearRepoData(db *gorm.DB, repoID uint) error {
	tables := []any{
		&models.File{},
		&models.Commit{},
		&models.Branch{},
		&models.Tag{},
		&models.Contributor{},
		&models.FileOwnership{},
		&models.Dependency{},
		&models.ImportEdge{},
		&models.DuplicateGroup{},
		&models.DuplicateBlock{},
	}
	for _, t := range tables {
		if err := db.Where("repo_id = ?", repoID).Delete(t).Error; err != nil {
			return fmt.Errorf("clear %T: %w", t, err)
		}
	}
	if err := db.Exec("DELETE FROM file_fts WHERE repo_id = ?", repoID).Error; err != nil {
		return fmt.Errorf("clear fts: %w", err)
	}
	return nil
}

// SettingsStore persists user settings as a single JSON row.
type SettingsStore struct {
	db *gorm.DB
}

const settingsKey = "app"

// NewSettingsStore builds a store backed by db.
func NewSettingsStore(db *gorm.DB) *SettingsStore {
	return &SettingsStore{db: db}
}

// Load returns the saved settings, falling back to defaults when unset.
func (s *SettingsStore) Load() (config.Settings, error) {
	var row models.Setting
	err := s.db.Where("key = ?", settingsKey).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return config.Defaults(), nil
	}
	if err != nil {
		return config.Settings{}, fmt.Errorf("load settings: %w", err)
	}
	var out config.Settings
	if err := json.Unmarshal([]byte(row.Value), &out); err != nil {
		return config.Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	return out.WithDefaults(), nil
}

// Save stores the settings as JSON.
func (s *SettingsStore) Save(settings config.Settings) error {
	raw, err := json.Marshal(settings)
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	row := models.Setting{Key: settingsKey, Value: string(raw)}
	if err := s.db.Save(&row).Error; err != nil {
		return fmt.Errorf("save settings: %w", err)
	}
	return nil
}
