package search

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearch(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	write(t, root, "cmd/main.go", "package main\n\nfunc main() { println(\"Hello World\") }\n")
	write(t, root, "lib/util.go", "package lib\n\nfunc Helper() { println(\"helper\") }\n")
	write(t, root, "docs/readme.md", "# hello\n")

	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	files := []models.File{
		{RepoID: repo.ID, Path: "cmd/main.go", Name: "main.go", Folder: "cmd", Extension: "go", Language: "Go", LinesTotal: 3},
		{RepoID: repo.ID, Path: "lib/util.go", Name: "util.go", Folder: "lib", Extension: "go", Language: "Go", LinesTotal: 3},
		{RepoID: repo.ID, Path: "docs/readme.md", Name: "readme.md", Folder: "docs", Extension: "md", Language: "Markdown", LinesTotal: 1},
	}
	if err := db.Create(&files).Error; err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		content, _ := os.ReadFile(filepath.Join(root, f.Path))
		if err := db.Exec("INSERT INTO file_fts (rowid, repo_id, path, content) VALUES (?, ?, ?, ?)",
			f.ID, repo.ID, f.Path, string(content)).Error; err != nil {
			t.Fatal(err)
		}
	}

	s := New(db)
	settings := config.Defaults()
	ctx := context.Background()

	// filename search
	res, err := s.Search(ctx, Query{RepoID: repo.ID, Text: "util", Mode: ModeFilename}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Path != "lib/util.go" {
		t.Fatalf("filename search: %+v", res.Hits)
	}

	// folder search
	res, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "cmd", Mode: ModeFolder}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Path != "cmd/main.go" {
		t.Fatalf("folder search: %+v", res.Hits)
	}

	// extension search
	res, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "md", Mode: ModeExtension}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Path != "docs/readme.md" {
		t.Fatalf("extension search: %+v", res.Hits)
	}

	// content search (case-insensitive: matches both "Hello" and "hello")
	res, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "Hello", Mode: ModeContent}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("content search expected 2 hits, got %+v", res.Hits)
	}
	foundMain := false
	for _, h := range res.Hits {
		if h.Path == "cmd/main.go" && len(h.Matches) > 0 {
			foundMain = true
		}
	}
	if !foundMain {
		t.Fatalf("content search missing main.go: %+v", res.Hits)
	}

	// case-sensitive content search: readme has "hello", main.go has "Hello"
	res, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "hello", Mode: ModeContent, CaseSensitive: true}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Path != "docs/readme.md" {
		t.Fatalf("case-sensitive search should only match 'hello': %+v", res.Hits)
	}

	// regex search
	res, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "func .+", Mode: ModeRegex}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 2 {
		t.Fatalf("regex search expected 2 hits, got %+v", res.Hits)
	}

	// invalid regex
	_, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "[", Mode: ModeRegex}, settings)
	if err == nil {
		t.Fatalf("expected invalid regex error")
	}

	// whole-word content search: "main" should not match "main.go" tokens
	res, err = s.Search(ctx, Query{RepoID: repo.ID, Text: "main", Mode: ModeContent, WholeWord: true}, settings)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Hits) != 1 || res.Hits[0].Path != "cmd/main.go" {
		t.Fatalf("whole-word content search: %+v", res.Hits)
	}
}
