package export

import (
	"bytes"
	"strings"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func TestExportCSV(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Repository{Name: "r", Path: "/tmp/r"})
	db.Create(&models.File{RepoID: 1, Path: "a.go", Language: "Go", LinesCode: 5, Complexity: 2})

	var buf bytes.Buffer
	if err := Export(db, 1, Params{Kind: KindFiles, Format: FormatCSV}, &buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "a.go") || !strings.HasPrefix(buf.String(), "path,") {
		t.Fatalf("unexpected csv: %q", buf.String())
	}
}

func TestExportCommitsJSON(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Export(db, 1, Params{Kind: KindCommits, Format: FormatJSON}, &buf); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "[]" && strings.TrimSpace(buf.String()) != "null" {
		t.Fatalf("unexpected json: %q", buf.String())
	}
}

func TestExportUnknownKind(t *testing.T) {
	db, _ := database.Open(":memory:")
	var buf bytes.Buffer
	if err := Export(db, 1, Params{Kind: "bogus", Format: FormatCSV}, &buf); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}
