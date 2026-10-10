package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

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
	if strings.TrimSpace(buf.String()) != "[]" {
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

func TestExportStreamsEveryRow(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	// More rows than one batch, to cross a batch boundary.
	commits := make([]models.Commit, batchSize+5)
	for i := range commits {
		commits[i] = models.Commit{RepoID: 1, Hash: fmt.Sprintf("h%05d", i), Author: "Ana", Message: "m",
			Date: time.Date(2024, 1, 1, 0, 0, i, 0, time.UTC)}
	}
	commits[0].Message = "=HYPERLINK(\"http://evil\")"
	if err := db.CreateInBatches(commits, 500).Error; err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Contributor{RepoID: 1, Name: "@ana", Email: "ana@x", Commits: 3})

	var js bytes.Buffer
	if err := Export(db, 1, Params{Kind: KindCommits, Format: FormatJSON}, &js); err != nil {
		t.Fatal(err)
	}
	var decoded []models.Commit
	if err := json.Unmarshal(js.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(decoded) != batchSize+5 || decoded[0].Hash != "h01004" {
		t.Fatalf("expected %d commits newest first, got %d starting %q", batchSize+5, len(decoded), decoded[0].Hash)
	}

	var csvOut bytes.Buffer
	if err := Export(db, 1, Params{Kind: KindCommits, Format: FormatCSV}, &csvOut); err != nil {
		t.Fatal(err)
	}
	rows, err := csv.NewReader(&csvOut).ReadAll()
	if err != nil {
		t.Fatalf("invalid csv: %v", err)
	}
	if len(rows) != batchSize+6 {
		t.Fatalf("expected header plus %d rows, got %d", batchSize+5, len(rows))
	}
	if last := rows[len(rows)-1]; last[4] != "'=HYPERLINK(\"http://evil\")" {
		t.Fatalf("expected the formula defused, got %q", last[4])
	}

	var people bytes.Buffer
	if err := Export(db, 1, Params{Kind: KindContributors, Format: FormatCSV}, &people); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(people.String(), "'@ana,ana@x,3") {
		t.Fatalf("unexpected contributors csv: %q", people.String())
	}
}
