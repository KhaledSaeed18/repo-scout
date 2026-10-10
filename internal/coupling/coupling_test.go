package coupling_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/coupling"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

type nullReporter struct{}

func (nullReporter) SetTotal(int)                     {}
func (nullReporter) SetProgress(float64)              {}
func (nullReporter) Inc(int)                          {}
func (nullReporter) SetMessage(string)                {}
func (nullReporter) Checkpoint(context.Context) error { return nil }

func run(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@example.com")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
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

// scanned builds a history where src/api.js and src/model.js (an import
// pair) and config/app.json and src/model.js (no import) change together
// three times each, and two files only ever change inside sweeping commits.
func scanned(t *testing.T) (*gorm.DB, uint) {
	t.Helper()
	root := t.TempDir()
	run(t, root, "init", "-q", "-b", "main", ".")
	for i := range 3 {
		write(t, root, "src/api.js", fmt.Sprintf("import { m } from './model.js'\nexport const a = %d\n", i))
		write(t, root, "src/model.js", fmt.Sprintf("export const m = %d\n", i))
		run(t, root, "add", ".")
		run(t, root, "commit", "-qm", "api and model")

		write(t, root, "config/app.json", fmt.Sprintf("{\"v\": %d}\n", i))
		write(t, root, "src/model.js", fmt.Sprintf("export const m = %d\n", i+10))
		run(t, root, "add", ".")
		run(t, root, "commit", "-qm", "config and model")

		// More files than the sweeping-commit cutoff of 50.
		for f := range 51 {
			write(t, root, fmt.Sprintf("sweep/f%02d.txt", f), fmt.Sprintf("%d\n", i))
		}
		run(t, root, "add", ".")
		run(t, root, "commit", "-qm", "sweep")
	}

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	repo := models.Repository{Name: "coupling", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	if err := analysis.New(db).Run(context.Background(), repo.ID, 1, nullReporter{}, config.Defaults()); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return db, repo.ID
}

func TestCoupling(t *testing.T) {
	db, id := scanned(t)

	pairs, err := coupling.List(db, id, coupling.Query{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs (sweeping commits skipped), got %+v", pairs)
	}
	byA := map[string]coupling.Pair{}
	for _, p := range pairs {
		byA[p.FileA] = p
	}
	imp, ok := byA["src/api.js"]
	if !ok || imp.FileB != "src/model.js" || imp.Shared != 3 || imp.Link != coupling.LinkImport {
		t.Fatalf("expected linked api/model pair, got %+v", pairs)
	}
	// api.js changed 3 times, model.js 6: 3 shared over an average of 4.5.
	if imp.Degree < 0.66 || imp.Degree > 0.67 {
		t.Fatalf("expected degree 2/3, got %v", imp.Degree)
	}
	hidden, ok := byA["config/app.json"]
	if !ok || hidden.Link != "" {
		t.Fatalf("expected an unlinked config/model pair, got %+v", pairs)
	}

	onlyHidden, err := coupling.List(db, id, coupling.Query{Hidden: true, Limit: 10})
	if err != nil {
		t.Fatalf("list hidden: %v", err)
	}
	if len(onlyHidden) != 1 || onlyHidden[0].FileA != "config/app.json" {
		t.Fatalf("expected only the hidden pair, got %+v", onlyHidden)
	}

	forFile, err := coupling.List(db, id, coupling.Query{Path: "src/api.js", Limit: 10})
	if err != nil {
		t.Fatalf("list for file: %v", err)
	}
	if len(forFile) != 1 || forFile[0].FileB != "src/model.js" {
		t.Fatalf("expected one pair for api.js, got %+v", forFile)
	}
}
