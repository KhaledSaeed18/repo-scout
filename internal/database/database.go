// Package database owns the SQLite connection, migrations, and persisted
// settings. It is the single place that touches the underlying database.
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// silentLogger wraps GORM's logger to suppress expected "record not found"
// errors that flood startup and idle workers.
type silentLogger struct {
	logger.Interface
}

func (l silentLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if err == gorm.ErrRecordNotFound {
		return
	}
	l.Interface.Trace(context.Background(), begin, fc, err)
}

func (l silentLogger) LogMode(level logger.LogLevel) logger.Interface {
	return silentLogger{Interface: l.Interface.LogMode(level)}
}

// Open connects to the SQLite database at path, enabling WAL mode and foreign
// keys. ":memory:" is supported for tests.
func Open(path string) (*gorm.DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("create database dir: %w", err)
		}
	}
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: silentLogger{Interface: logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
			// Scans promote and aggregate whole tables at once; only flag
			// queries slow enough to matter. No color: logs end up in files
			// and CI output.
			SlowThreshold: 2 * time.Second,
			LogLevel:      logger.Warn,
			Colorful:      false,
		})},
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

// Migrate creates all tables, indexes, and the FTS5 content-search table,
// then applies any pending numbered migrations (see migrations.go).
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&models.Repository{},
		&models.File{},
		&models.Commit{},
		&models.CommitFile{},
		&models.Branch{},
		&models.Tag{},
		&models.Contributor{},
		&models.FileOwnership{},
		&models.Dependency{},
		&models.ImportEdge{},
		&models.FileCoupling{},
		&models.DuplicateGroup{},
		&models.DuplicateBlock{},
		&models.Job{},
		&models.ScanSnapshot{},
		&models.Setting{},
	); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	if err := ensureFileFTS(db); err != nil {
		return err
	}
	return applyMigrations(db, migrations())
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

// stagingOffset separates the IDs scans stage their results under from real
// repository IDs. Analysis tables carry a repo_id but no foreign key, so a
// scan can write under a private ID and swap the rows in once it succeeds.
const stagingOffset uint = 1 << 40

// StagingID is the repository ID a scan of repoID writes its results under
// until they are complete. Readers keep seeing the previous results until
// PromoteRepoData moves the staged rows into place.
func StagingID(repoID uint) uint { return repoID + stagingOffset }

// repoTables lists every table holding per-repository analysis rows.
func repoTables() []any {
	return []any{
		&models.File{},
		&models.Commit{},
		&models.CommitFile{},
		&models.Branch{},
		&models.Tag{},
		&models.Contributor{},
		&models.FileOwnership{},
		&models.Dependency{},
		&models.ImportEdge{},
		&models.FileCoupling{},
		&models.DuplicateGroup{},
		&models.DuplicateBlock{},
	}
}

// ClearRepoData removes every analysis row and index entry for a repository.
func ClearRepoData(db *gorm.DB, repoID uint) error {
	for _, t := range repoTables() {
		if err := db.Where("repo_id = ?", repoID).Delete(t).Error; err != nil {
			return fmt.Errorf("clear %T: %w", t, err)
		}
	}
	if err := db.Exec("DELETE FROM file_fts WHERE repo_id = ?", repoID).Error; err != nil {
		return fmt.Errorf("clear fts: %w", err)
	}
	return nil
}

// ClearStagingData removes results left behind by scans that never finished,
// for example because the process stopped mid-scan. Call it before workers
// start.
func ClearStagingData(db *gorm.DB) error {
	for _, t := range repoTables() {
		if err := db.Where("repo_id >= ?", stagingOffset).Delete(t).Error; err != nil {
			return fmt.Errorf("clear staged %T: %w", t, err)
		}
	}
	if err := db.Exec("DELETE FROM file_fts WHERE repo_id >= ?", stagingOffset).Error; err != nil {
		return fmt.Errorf("clear staged fts: %w", err)
	}
	return nil
}

// PromoteRepoData replaces the results of repository to with the rows staged
// under from. Run it inside a transaction so readers never see a mix.
func PromoteRepoData(tx *gorm.DB, from, to uint) error {
	if err := ClearRepoData(tx, to); err != nil {
		return err
	}
	for _, t := range repoTables() {
		if err := tx.Model(t).Where("repo_id = ?", from).Update("repo_id", to).Error; err != nil {
			return fmt.Errorf("promote %T: %w", t, err)
		}
	}
	if err := tx.Exec("UPDATE file_fts SET repo_id = ? WHERE repo_id = ?", to, from).Error; err != nil {
		return fmt.Errorf("promote fts: %w", err)
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
