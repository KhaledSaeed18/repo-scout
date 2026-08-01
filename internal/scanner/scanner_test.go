package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func mkFile(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	mkFile(t, root, "main.go", "package main\n\nfunc main() {\n}\n")
	mkFile(t, root, "util/helper.py", "# comment\n\ndef f():\n    return 1\n")
	mkFile(t, root, "node_modules/x.js", "ignore me\n")
	mkFile(t, root, "data.bin", string([]byte{1, 0, 2, 3}))
	mkFile(t, root, "notes/readme.txt", "hello\n")

	db := testDB(t)
	repo := models.Repository{Name: "r", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatalf("create repo: %v", err)
	}

	sc := New(db)
	settings := config.Defaults()
	var done, total int
	stats, err := sc.Scan(context.Background(), repo.ID, root, settings, func(d, tt int) {
		done, total = d, tt
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	if stats.FileCount != 4 {
		t.Fatalf("file count: got %d, want 4", stats.FileCount)
	}
	if done != 4 || total != 4 {
		t.Fatalf("progress: got %d/%d, want 4/4", done, total)
	}
	if stats.BinaryCount != 1 {
		t.Fatalf("binary count: got %d, want 1", stats.BinaryCount)
	}
	if stats.TotalLOC != 9 {
		t.Fatalf("total loc: got %d, want 9", stats.TotalLOC)
	}
	if stats.ByLanguage["Go"] == nil || stats.ByLanguage["Python"] == nil {
		t.Fatalf("missing language stats: %+v", stats.ByLanguage)
	}

	var files []models.File
	if err := db.Where("repo_id = ?", repo.ID).Find(&files).Error; err != nil {
		t.Fatalf("load files: %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("persisted files: got %d, want 4", len(files))
	}
	for _, f := range files {
		if f.Path == "node_modules/x.js" {
			t.Fatalf("ignored folder leaked into scan: %s", f.Path)
		}
	}
}

func TestScanCancel(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 50; i++ {
		mkFile(t, root, filepath.Join("d", string(rune('a'+i%26)), "f.go"), "package x\n")
	}
	db := testDB(t)
	repo := models.Repository{Name: "r", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatalf("create repo: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sc := New(db)
	_, err := sc.Scan(ctx, repo.ID, root, config.Defaults(), nil)
	if err == nil {
		t.Fatalf("expected cancellation error")
	}
}
