package risk

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

type nullReporter struct{}

func (nullReporter) SetTotal(int)                     {}
func (nullReporter) SetProgress(float64)              {}
func (nullReporter) Inc(int)                          {}
func (nullReporter) SetMessage(string)                {}
func (nullReporter) Checkpoint(context.Context) error { return nil }

// commit records every change in root as one commit by who at date.
func commit(t *testing.T, root, who, date, msg string) {
	t.Helper()
	for _, args := range [][]string{{"add", "."}, {"commit", "-qm", msg}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME="+who, "GIT_AUTHOR_EMAIL="+who+"@example.com",
			"GIT_COMMITTER_NAME="+who, "GIT_COMMITTER_EMAIL="+who+"@example.com",
			"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
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

const complexGo = `package core

func Decide(a, b int) int {
	if a > b {
		for i := 0; i < a; i++ {
			if i%%2 == 0 && b > 0 {
				b--
			}
		}
	}
	switch {
	case a == 0:
		return 1
	case b == 0:
		return 2
	}
	return %d
}
`

// scanned builds a repository where core/decide.go is complex and edited
// often by bob, and util/simple.go was written by alice long ago and later
// touched by a bot.
func scanned(t *testing.T) (*gorm.DB, uint) {
	t.Helper()
	root := t.TempDir()
	cmd := exec.Command("git", "-C", root, "init", "-q", "-b", "main", ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	write(t, root, "util/simple.go", "package util\n\nfunc One() int { if true { return 1 }; return 0 }\n")
	commit(t, root, "alice", "2022-01-01T10:00:00", "simple")
	// A later bot commit must not take ownership from alice.
	write(t, root, "util/simple.go", "package util\n\nfunc One() int { if true { return 1 }; return 2 }\n")
	commit(t, root, "dependabot[bot]", "2022-02-01T10:00:00", "bump")
	for i, date := range []string{"2024-01-01T10:00:00", "2024-03-01T10:00:00", "2024-05-01T10:00:00", "2024-06-01T10:00:00"} {
		write(t, root, "core/decide.go", fmt.Sprintf(complexGo, i))
		commit(t, root, "bob", date, "decide")
	}

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := models.Repository{Name: "risk", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	if err := analysis.New(db).Run(context.Background(), repo.ID, 1, nullReporter{}, config.Defaults()); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return db, repo.ID
}

func TestHotspots(t *testing.T) {
	db, id := scanned(t)

	all, err := Hotspots(db, id, 0, 10)
	if err != nil {
		t.Fatalf("hotspots: %v", err)
	}
	if all.Since != nil || all.Until == nil {
		t.Fatalf("expected an all-history window ending at the latest commit, got %+v", all)
	}
	if len(all.Hotspots) != 2 || all.Files != 2 {
		t.Fatalf("expected 2 hotspots, got %+v", all.Hotspots)
	}
	top := all.Hotspots[0]
	if top.Path != "core/decide.go" || top.Revisions != 4 || top.Authors != 1 || top.Score != 1 {
		t.Fatalf("unexpected top hotspot %+v", top)
	}
	if all.Hotspots[1].Score >= 1 || all.Hotspots[1].Score <= 0 {
		t.Fatalf("expected a relative score below the top, got %v", all.Hotspots[1].Score)
	}

	recent, err := Hotspots(db, id, 12, 10)
	if err != nil {
		t.Fatalf("recent hotspots: %v", err)
	}
	if len(recent.Hotspots) != 1 || recent.Hotspots[0].Path != "core/decide.go" {
		t.Fatalf("expected only the recently changed file, got %+v", recent.Hotspots)
	}
}

func TestKnowledge(t *testing.T) {
	db, id := scanned(t)
	rep, err := Knowledge(db, id, 1, 6)
	if err != nil {
		t.Fatalf("knowledge: %v", err)
	}
	if rep.Files != 2 || len(rep.Owners) != 2 {
		t.Fatalf("expected 2 files with 2 owners, got %+v", rep)
	}
	if rep.Owners[0].Author != "bob" || !rep.Owners[0].Active || rep.Owners[1].Active {
		t.Fatalf("expected active bob first and inactive alice, got %+v", rep.Owners)
	}
	if len(rep.AtRisk) != 1 || rep.AtRisk[0].Path != "util/simple.go" || rep.AtRisk[0].Owner != "alice" {
		t.Fatalf("expected alice's file at risk, got %+v", rep.AtRisk)
	}
	if rep.Folders[0].Folder != "util" || rep.Folders[0].InactiveShare != 1 {
		t.Fatalf("expected util first and fully inactive, got %+v", rep.Folders)
	}
}

func TestKnowledgeWithoutHistory(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	rep, err := Knowledge(db, 1, 2, 6)
	if err != nil {
		t.Fatalf("knowledge: %v", err)
	}
	if rep.BusFactor != 0 || rep.Owners == nil || rep.Folders == nil || rep.AtRisk == nil {
		t.Fatalf("expected an empty report with empty lists, got %+v", rep)
	}
	hs, err := Hotspots(db, 1, 12, 10)
	if err != nil || hs.Hotspots == nil || hs.Until != nil {
		t.Fatalf("expected an empty hotspot report, got %+v, %v", hs, err)
	}
}

func TestBuildKnowledgeBusFactor(t *testing.T) {
	now := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	seen := map[string]time.Time{"a@x": now, "b@x": now, "c@x": now}
	files := []ownedFile{
		{Path: "a/1.go", Email: "a@x", Author: "A", LinesCode: 40},
		{Path: "b/1.go", Email: "b@x", Author: "B", LinesCode: 35},
		{Path: "c/1.go", Email: "c@x", Author: "C", LinesCode: 25},
	}
	// Losing A leaves 40% orphaned; losing A and B leaves 75%.
	rep := buildKnowledge(files, seen, now.AddDate(0, -6, 0), 2, 10)
	if rep.BusFactor != 2 {
		t.Fatalf("expected bus factor 2, got %d", rep.BusFactor)
	}
	if rep.InactiveLines != 0 || len(rep.AtRisk) != 0 {
		t.Fatalf("expected nothing at risk, got %+v", rep.AtRisk)
	}
}

func TestFolderAt(t *testing.T) {
	cases := []struct {
		path  string
		depth int
		want  string
	}{
		{"main.go", 2, ""},
		{"internal/api/server.go", 1, "internal"},
		{"internal/api/server.go", 2, "internal/api"},
		{"internal/api/server.go", 5, "internal/api"},
	}
	for _, c := range cases {
		if got := folderAt(c.path, c.depth); got != c.want {
			t.Errorf("folderAt(%q, %d) = %q, want %q", c.path, c.depth, got, c.want)
		}
	}
}
