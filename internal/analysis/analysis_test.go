package analysis

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/jobs"
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

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
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

// reporter captures progress for assertions.
type reporter struct {
	lastMsg string
}

func (r *reporter) SetTotal(n int)                       {}
func (r *reporter) SetProgress(f float64)                {}
func (r *reporter) Inc(n int)                            {}
func (r *reporter) SetMessage(msg string)                { r.lastMsg = msg }
func (r *reporter) Checkpoint(ctx context.Context) error { return nil }

func makeFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "go.mod", "module example.com/demo\n\ngo 1.22\n")
	write(t, root, "main.go", `package main

import (
	"example.com/demo/pkg/util"
)

func main() {
	util.Help()
}
`)
	write(t, root, "pkg/util/util.go", `package util

// Help prints a message.
func Help() {
	println("hello")
}
`)
	write(t, root, "pkg/dup/one.go", `package dup

func build() string {
	if true {
		return "x"
	}
	return "y"
}
`)
	write(t, root, "pkg/dup/two.go", `package dup

func build() string {
	if true {
		return "x"
	}
	return "y"
}
`)
	git(t, root, "init", "-q", "-b", "main", ".")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "initial")
	write(t, root, "pkg/util/util.go", "package util\n\nfunc Help() {\n\tprintln(\"hello world\")\n}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "update helper")
	git(t, root, "tag", "v1.0")
	return root
}

func TestRunPipeline(t *testing.T) {
	db := testDB(t)
	root := makeFixture(t)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}

	r := New(db)
	rep := &reporter{}
	err := r.Run(context.Background(), repo.ID, 1, rep, config.Defaults())
	if err != nil {
		t.Fatalf("run pipeline: %v", err)
	}

	db.First(&repo)
	if repo.Status != models.RepoReady {
		t.Fatalf("expected repo ready, got %s", repo.Status)
	}
	if repo.FileCount != 5 {
		t.Fatalf("expected 5 files, got %d", repo.FileCount)
	}
	if repo.CommitCount != 2 {
		t.Fatalf("expected 2 commits, got %d", repo.CommitCount)
	}
	if repo.ContributorCount != 1 {
		t.Fatalf("expected 1 contributor, got %d", repo.ContributorCount)
	}
	if repo.DupGroupCount < 1 {
		t.Fatalf("expected >= 1 duplicate group, got %d", repo.DupGroupCount)
	}

	var edges []models.ImportEdge
	if err := db.Where("repo_id = ?", repo.ID).Find(&edges).Error; err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range edges {
		if e.FromFile == "main.go" && e.ToFile == "pkg/util" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected main.go -> pkg/util import edge, got %+v", edges)
	}

	var commits []models.Commit
	db.Where("repo_id = ?", repo.ID).Order("date ASC").Find(&commits)
	if len(commits) != 2 || commits[0].Hash == "" {
		t.Fatalf("unexpected commits: %+v", commits)
	}

	// FTS index populated
	var fts int64
	db.Raw("SELECT count(*) FROM file_fts WHERE repo_id = ?", repo.ID).Scan(&fts)
	if fts == 0 {
		t.Fatalf("expected fts rows")
	}
}

func TestRunPipelineNonGit(t *testing.T) {
	db := testDB(t)
	root := t.TempDir()
	write(t, root, "file.go", "package x\n")
	repo := models.Repository{Name: "plain", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	r := New(db)
	err := r.Run(context.Background(), repo.ID, 1, &reporter{}, config.Defaults())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	db.First(&repo)
	if repo.Status != models.RepoReady {
		t.Fatalf("expected ready, got %s", repo.Status)
	}
	if repo.FileCount != 1 {
		t.Fatalf("expected 1 file, got %d", repo.FileCount)
	}
}

var _ jobs.Reporter = (*reporter)(nil)
