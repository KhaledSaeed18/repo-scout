package gitrepo

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func makeRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main", ".")
	writeFile(t, root, "a.go", "package a\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "add a")
	writeFile(t, root, "b.go", "package b\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "add b")
	git(t, root, "checkout", "-q", "-b", "feature")
	writeFile(t, root, "c.go", "package c\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "add c")
	git(t, root, "checkout", "-q", "main")
	git(t, root, "merge", "--no-ff", "-qm", "merge feature", "feature")
	git(t, root, "tag", "v1.0")
	return root
}

func TestIsRepoAndMeta(t *testing.T) {
	root := makeRepo(t)
	a := New()
	if !a.IsRepo(root) {
		t.Fatalf("expected git repo")
	}
	if a.IsRepo(t.TempDir()) {
		t.Fatalf("expected non-repo")
	}
	m, err := a.Meta(context.Background(), root)
	if err != nil {
		t.Fatalf("meta: %v", err)
	}
	if m.BranchCount != 2 {
		t.Fatalf("branch count: got %d, want 2", m.BranchCount)
	}
	if m.TagCount != 1 {
		t.Fatalf("tag count: got %d, want 1", m.TagCount)
	}
	if m.DefaultBranch != "main" {
		t.Fatalf("default branch: got %s", m.DefaultBranch)
	}
}

func TestBranchesAndTags(t *testing.T) {
	root := makeRepo(t)
	a := New()
	ctx := context.Background()
	branches, err := a.Branches(ctx, root, "main")
	if err != nil {
		t.Fatalf("branches: %v", err)
	}
	if len(branches) != 2 {
		t.Fatalf("expected 2 branches, got %d", len(branches))
	}
	foundCurrent := false
	for _, b := range branches {
		if b.Name == "main" && b.IsCurrent {
			foundCurrent = true
		}
	}
	if !foundCurrent {
		t.Fatalf("expected main to be current")
	}
	tags, err := a.Tags(ctx, root)
	if err != nil {
		t.Fatalf("tags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v1.0" {
		t.Fatalf("unexpected tags: %+v", tags)
	}
}

func TestStreamLogs(t *testing.T) {
	root := makeRepo(t)
	a := New()
	var commits []models.Commit
	totalFiles := 0
	err := a.StreamLogs(context.Background(), root, func(c models.Commit, files []FileChange) {
		commits = append(commits, c)
		totalFiles += len(files)
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	// a, b, merge (no files), c = 4 commits
	if len(commits) != 4 {
		t.Fatalf("expected 4 commits, got %d", len(commits))
	}
	mergeSeen := false
	for _, c := range commits {
		if c.IsMerge {
			mergeSeen = true
		}
	}
	if !mergeSeen {
		t.Fatalf("expected a merge commit")
	}
	// a.go, b.go, c.go across commits = 3 file changes
	if totalFiles != 3 {
		t.Fatalf("expected 3 file changes, got %d", totalFiles)
	}
}

func TestAnalyzeHistory(t *testing.T) {
	root := makeRepo(t)
	a := New()
	contrib, files, err := a.AnalyzeHistory(context.Background(), root)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if len(contrib) != 1 {
		t.Fatalf("expected 1 contributor, got %d", len(contrib))
	}
	for _, cs := range contrib {
		if cs.Commits != 4 {
			t.Fatalf("expected 4 commits for contributor, got %d", cs.Commits)
		}
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files in history, got %d", len(files))
	}
	fh := files["c.go"]
	if fh == nil || fh.Commits != 1 || fh.Author != "Test" {
		t.Fatalf("unexpected c.go history: %+v", fh)
	}
}
