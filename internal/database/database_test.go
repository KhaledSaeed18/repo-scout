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

func TestPromoteRepoData(t *testing.T) {
	db := testDB(t)
	repo := models.Repository{Name: "r", Path: "/tmp/r"}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatalf("create repo: %v", err)
	}
	staging := StagingID(repo.ID)
	if err := db.Create(&models.File{RepoID: repo.ID, Path: "old.go"}).Error; err != nil {
		t.Fatalf("create old file: %v", err)
	}
	if err := db.Create(&models.File{RepoID: staging, Path: "new.go"}).Error; err != nil {
		t.Fatalf("create staged file: %v", err)
	}
	if err := db.Exec("INSERT INTO file_fts (repo_id, path, content) VALUES (?, ?, ?)",
		staging, "new.go", "package a").Error; err != nil {
		t.Fatalf("insert fts: %v", err)
	}

	if err := db.Transaction(func(tx *gorm.DB) error { return PromoteRepoData(tx, staging, repo.ID) }); err != nil {
		t.Fatalf("promote: %v", err)
	}
	var paths []string
	db.Model(&models.File{}).Where("repo_id = ?", repo.ID).Pluck("path", &paths)
	if len(paths) != 1 || paths[0] != "new.go" {
		t.Fatalf("expected only the staged file, got %v", paths)
	}
	var fts int64
	db.Raw("SELECT count(*) FROM file_fts WHERE repo_id = ?", repo.ID).Scan(&fts)
	if fts != 1 {
		t.Fatalf("expected staged fts row promoted, got %d", fts)
	}
	var left int64
	db.Model(&models.File{}).Where("repo_id = ?", staging).Count(&left)
	if left != 0 {
		t.Fatalf("expected no staged rows left, got %d", left)
	}
}

func TestClearStagingData(t *testing.T) {
	db := testDB(t)
	if err := db.Create(&models.File{RepoID: 1, Path: "kept.go"}).Error; err != nil {
		t.Fatalf("create file: %v", err)
	}
	if err := db.Create(&models.File{RepoID: StagingID(1), Path: "orphan.go"}).Error; err != nil {
		t.Fatalf("create staged file: %v", err)
	}
	if err := ClearStagingData(db); err != nil {
		t.Fatalf("clear staging: %v", err)
	}
	var paths []string
	db.Model(&models.File{}).Pluck("path", &paths)
	if len(paths) != 1 || paths[0] != "kept.go" {
		t.Fatalf("expected only real rows kept, got %v", paths)
	}
}
