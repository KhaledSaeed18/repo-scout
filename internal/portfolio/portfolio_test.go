package portfolio_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/portfolio"
)

type nullReporter struct{}

func (nullReporter) SetTotal(int)                     {}
func (nullReporter) SetProgress(float64)              {}
func (nullReporter) Inc(int)                          {}
func (nullReporter) SetMessage(string)                {}
func (nullReporter) Checkpoint(context.Context) error { return nil }

func gitRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main", "."}, {"add", "."}, {"commit", "-qm", "init"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@x")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func TestBuild(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	cyclic := gitRepo(t, map[string]string{
		"go.mod":  "module example.com/c\n",
		"main.go": "package main\n\nimport \"example.com/c/a\"\n\nfunc main() { a.A() }\n",
		"a/a.go":  "package a\n\nimport \"example.com/c/b\"\n\nfunc A() { b.B() }\n",
		"b/b.go":  "package b\n\nimport \"example.com/c/a\"\n\nfunc B() {\n\tif true {\n\t\ta.A()\n\t}\n}\n",
	})
	plain := gitRepo(t, map[string]string{"main.go": "package main\n\nfunc main() {}\n"})

	scan := func(name, path string) uint {
		repo := models.Repository{Name: name, Path: path}
		if err := db.Create(&repo).Error; err != nil {
			t.Fatal(err)
		}
		if err := analysis.New(db).Run(context.Background(), repo.ID, 0, nullReporter{}, config.Defaults()); err != nil {
			t.Fatalf("scan %s: %v", name, err)
		}
		return repo.ID
	}
	scan("cyclic", cyclic)
	plainID := scan("plain", plain)
	// A second scan of plain gives it a direction.
	if err := os.WriteFile(filepath.Join(plain, "more.go"), []byte("package main\n\nfunc more() int {\n\treturn 1\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := analysis.New(db).Run(context.Background(), plainID, 0, nullReporter{}, config.Defaults()); err != nil {
		t.Fatal(err)
	}
	db.Create(&models.Repository{Name: "waiting", Path: "/nowhere"})

	p, err := portfolio.Build(db)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if p.Unscanned != 1 || len(p.Entries) != 2 {
		t.Fatalf("expected 2 entries and 1 unscanned, got %+v", p)
	}
	c, pl := p.Entries[0], p.Entries[1]
	if c.Repository.Name != "cyclic" || c.Cycles != 1 || c.BusFactor != 1 || c.LinesChange != nil {
		t.Fatalf("unexpected cyclic entry %+v", c)
	}
	if pl.Cycles != 0 || pl.LinesChange == nil || *pl.LinesChange <= 0 {
		t.Fatalf("expected plain to have grown, got %+v", pl)
	}
}
