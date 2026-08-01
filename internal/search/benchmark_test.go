package search_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/search"
)

func seededStore(tb testing.TB, files int) (*search.Store, uint) {
	tb.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		tb.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		tb.Fatal(err)
	}
	root := tb.TempDir()
	store := search.New(db)
	if err := db.Create(&models.Repository{ID: 1, Name: "bench", Path: root}).Error; err != nil {
		tb.Fatal(err)
	}
	var rows []models.File
	for i := 0; i < files; i++ {
		rel := fmt.Sprintf("src/file%04d.go", i)
		content := fmt.Sprintf("package src\n\n// needle marker %d\nfunc Handler%d() {\n}\n", i, i)
		os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644)
		rows = append(rows, models.File{ID: uint(i + 1), Path: rel, Language: "Go"})
	}
	if err := store.Reindex(context.Background(), 1, root, rows, nil); err != nil {
		tb.Fatal(err)
	}
	return store, 1
}

func BenchmarkContentSearch(b *testing.B) {
	store, repoID := seededStore(b, 2000)
	q := search.Query{RepoID: repoID, Text: "Handler123", Mode: search.ModeContent}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Search(context.Background(), q, config.Defaults()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFilenameSearch(b *testing.B) {
	store, repoID := seededStore(b, 2000)
	q := search.Query{RepoID: repoID, Text: "file0500", Mode: search.ModeFilename}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.Search(context.Background(), q, config.Defaults()); err != nil {
			b.Fatal(err)
		}
	}
}
