package database

import (
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func TestMigrationsRunOnceInOrder(t *testing.T) {
	db := testDB(t)
	var ran []int
	list := []migration{
		{version: 2, name: "two", up: func(*gorm.DB) error { ran = append(ran, 2); return nil }},
		{version: 3, name: "three", up: func(*gorm.DB) error { ran = append(ran, 3); return nil }},
	}
	for range 2 {
		if err := applyMigrations(db, list); err != nil {
			t.Fatalf("apply: %v", err)
		}
	}
	if len(ran) != 2 || ran[0] != 2 || ran[1] != 3 {
		t.Fatalf("expected 2 then 3 once each, got %v", ran)
	}
}

func TestFailedMigrationIsNotRecorded(t *testing.T) {
	db := testDB(t)
	boom := errors.New("boom")
	list := []migration{{version: 2, name: "fails", up: func(tx *gorm.DB) error {
		if err := tx.Create(&models.Repository{Name: "half", Path: "/half"}).Error; err != nil {
			return err
		}
		return boom
	}}}
	if err := applyMigrations(db, list); !errors.Is(err, boom) {
		t.Fatalf("expected the migration error, got %v", err)
	}
	var version int
	db.Model(&schemaMigration{}).Select("COALESCE(MAX(version), 0)").Scan(&version)
	if version != 1 {
		t.Fatalf("expected the failed migration unrecorded, got version %d", version)
	}
	var repos int64
	db.Model(&models.Repository{}).Count(&repos)
	if repos != 0 {
		t.Fatalf("expected the failed migration rolled back, got %d rows", repos)
	}
}

func TestRefusesNewerSchema(t *testing.T) {
	db := testDB(t)
	if err := db.Create(&schemaMigration{Version: 999, Name: "from the future"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); !errors.Is(err, ErrNewerSchema) {
		t.Fatalf("expected ErrNewerSchema, got %v", err)
	}
}

func TestBackfillOwnershipEmails(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Contributor{}, &models.FileOwnership{}); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Contributor{RepoID: 1, Name: "Ana", Email: "ana@x", Commits: 3})
	db.Create(&models.FileOwnership{RepoID: 1, Path: "a.go", Author: "Ana"})
	db.Create(&models.FileOwnership{RepoID: 1, Path: "b.go", Author: "Ghost"})
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	var rows []models.FileOwnership
	db.Order("path").Find(&rows)
	if rows[0].Email != "ana@x" || rows[1].Email != "" {
		t.Fatalf("expected Ana's email filled and Ghost left blank, got %+v", rows)
	}
}
