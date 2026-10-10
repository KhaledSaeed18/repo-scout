package database

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// migration is one numbered schema or data change. AutoMigrate already adds
// new tables, columns and indexes, so migrations cover what it cannot:
// backfills, renames, drops and type changes. Each runs once, in a
// transaction, in version order.
//
// Shipped migrations are never edited or renumbered; a later change gets the
// next number.
type migration struct {
	version int
	name    string
	up      func(tx *gorm.DB) error
}

// schemaMigration records an applied migration.
type schemaMigration struct {
	Version   int `gorm:"primaryKey;autoIncrement:false"`
	Name      string
	AppliedAt time.Time
}

func (schemaMigration) TableName() string { return "schema_migrations" }

// migrations lists every migration, oldest first.
func migrations() []migration {
	return []migration{
		{
			// Ownership rows from scans before ownership recorded email
			// addresses carry only a name; match it to the contributor so the
			// knowledge views group those authors with their newer rows.
			version: 1,
			name:    "backfill ownership emails",
			up: func(tx *gorm.DB) error {
				return tx.Exec(`UPDATE file_ownerships SET email = COALESCE((
						SELECT c.email FROM contributors c
						WHERE c.repo_id = file_ownerships.repo_id AND c.name = file_ownerships.author
						ORDER BY c.commits DESC LIMIT 1
					), '')
					WHERE email = ''`).Error
			},
		},
	}
}

// ErrNewerSchema means the database was last written by a newer Repo Scout.
var ErrNewerSchema = errors.New("database schema is newer than this build of Repo Scout")

// applyMigrations runs every migration newer than the database's version.
func applyMigrations(db *gorm.DB, list []migration) error {
	if err := db.AutoMigrate(&schemaMigration{}); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	var current int
	if err := db.Model(&schemaMigration{}).Select("COALESCE(MAX(version), 0)").Scan(&current).Error; err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	latest := 0
	if len(list) > 0 {
		latest = list[len(list)-1].version
	}
	if current > latest {
		return fmt.Errorf("%w: database is at version %d, this build knows up to %d; upgrade Repo Scout", ErrNewerSchema, current, latest)
	}
	for _, m := range list {
		if m.version <= current {
			continue
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := m.up(tx); err != nil {
				return err
			}
			return tx.Create(&schemaMigration{Version: m.version, Name: m.name, AppliedAt: time.Now().UTC()}).Error
		})
		if err != nil {
			return fmt.Errorf("migration %d (%s): %w", m.version, m.name, err)
		}
	}
	return nil
}
