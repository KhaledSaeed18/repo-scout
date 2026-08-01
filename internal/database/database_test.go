package database

import (
	"testing"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestSettingsRoundTrip(t *testing.T) {
	db := testDB(t)
	store := NewSettingsStore(db)

	got, err := store.Load()
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}
	if got.WorkerCount != config.Defaults().WorkerCount {
		t.Fatalf("expected default worker count, got %d", got.WorkerCount)
	}

	want := config.Defaults()
	want.WorkerCount = 8
	want.MaxFileSize = 1 << 20
	if err := store.Save(want); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err = store.Load()
	if err != nil {
		t.Fatalf("load saved: %v", err)
	}
	if got.WorkerCount != 8 || got.MaxFileSize != 1<<20 {
		t.Fatalf("settings not round-tripped: %+v", got)
	}
}

func TestClearRepoData(t *testing.T) {
	db := testDB(t)
	repo := models.Repository{Name: "r", Path: "/tmp/r"}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatalf("create repo: %v", err)
	}
	if err := db.Create(&models.File{RepoID: repo.ID, Path: "a.go"}).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := db.Exec("INSERT INTO file_fts (repo_id, path, content) VALUES (?, ?, ?)",
		repo.ID, "a.go", "package a").Error; err != nil {
		t.Fatalf("insert fts: %v", err)
	}
	if err := ClearRepoData(db, repo.ID); err != nil {
		t.Fatalf("clear: %v", err)
	}
	var files int64
	db.Model(&models.File{}).Where("repo_id = ?", repo.ID).Count(&files)
	if files != 0 {
		t.Fatalf("expected files cleared, got %d", files)
	}
	var fts int64
	db.Raw("SELECT count(*) FROM file_fts WHERE repo_id = ?", repo.ID).Scan(&fts)
	if fts != 0 {
		t.Fatalf("expected fts cleared, got %d", fts)
	}
}
