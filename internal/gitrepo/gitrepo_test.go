package gitrepo

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	gitOut(t, dir, args...)
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
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
	err := a.StreamLogs(context.Background(), root, func(c models.Commit, files []FileChange) error {
		commits = append(commits, c)
		totalFiles += len(files)
		return nil
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

func TestStreamLogsIgnoresPrivateRefs(t *testing.T) {
	root := makeRepo(t)
	// A tool-owned ref (e.g. an editor checkpoint) must not leak into history.
	tree := strings.TrimSpace(gitOut(t, root, "rev-parse", "HEAD^{tree}"))
	hidden := strings.TrimSpace(gitOut(t, root, "commit-tree", tree, "-p", "HEAD", "-m", "checkpoint"))
	git(t, root, "update-ref", "refs/tools/checkpoints/one", hidden)
	// A detached HEAD commit is real work and must be included.
	git(t, root, "checkout", "-q", "--detach")
	writeFile(t, root, "d.go", "package main\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "detached work")

	var messages []string
	err := New().StreamLogs(context.Background(), root, func(c models.Commit, _ []FileChange) error {
		messages = append(messages, c.Message)
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for _, m := range messages {
		if m == "checkpoint" {
			t.Fatalf("private ref commit leaked into history: %v", messages)
		}
	}
	if len(messages) != 5 || messages[0] != "detached work" {
		t.Fatalf("expected 5 commits starting with detached work, got %v", messages)
	}
}

func TestStreamLogsStopsOnCallbackError(t *testing.T) {
	root := makeRepo(t)
	stop := errors.New("stop")
	calls := 0
	err := New().StreamLogs(context.Background(), root, func(models.Commit, []FileChange) error {
		calls++
		return stop
	})
	if !errors.Is(err, stop) {
		t.Fatalf("expected callback error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected the stream to stop after the first commit, got %d calls", calls)
	}
}

func TestStreamLogsEmptyRepo(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main", ".")
	called := false
	err := New().StreamLogs(context.Background(), root, func(models.Commit, []FileChange) error { called = true; return nil })
	if err != nil {
		t.Fatalf("stream on empty repo: %v", err)
	}
	if called {
		t.Fatalf("expected no commits")
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

func TestStreamLogsKeepsAuthorTimezone(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main", ".")
	writeFile(t, root, "a.go", "package a\n")
	git(t, root, "add", ".")
	cmd := exec.Command("git", "-C", root, "commit", "-qm", "late night")
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE=2024-01-01T23:30:00+03:00", "GIT_COMMITTER_DATE=2024-01-01T23:30:00+03:00")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}

	var got models.Commit
	if err := New().StreamLogs(context.Background(), root, func(c models.Commit, _ []FileChange) error { got = c; return nil }); err != nil {
		t.Fatal(err)
	}
	if got.TZOffset != 180 {
		t.Fatalf("expected +180 minute offset, got %d", got.TZOffset)
	}
	if want := time.Date(2024, 1, 1, 20, 30, 0, 0, time.UTC); !got.Date.Equal(want) || got.Date.Location() != time.UTC {
		t.Fatalf("expected UTC instant %v, got %v", want, got.Date)
	}
	if local := got.LocalTime(); local.Hour() != 23 || local.Day() != 1 {
		t.Fatalf("expected author-local 23:30 on Jan 1, got %v", local)
	}
}

func TestSplitRename(t *testing.T) {
	cases := []struct{ in, old, new string }{
		{"a.go", "", "a.go"},
		{"old.go => new.go", "old.go", "new.go"},
		{"src/{a => b}/c.go", "src/a/c.go", "src/b/c.go"},
		{"src/{ => sub}/c.go", "src/c.go", "src/sub/c.go"},
		{"{lib => pkg}/x.go", "lib/x.go", "pkg/x.go"},
		{"dir/{x.go => y.go}", "dir/x.go", "dir/y.go"},
	}
	for _, c := range cases {
		old, nw := splitRename(c.in)
		if old != c.old || nw != c.new {
			t.Errorf("splitRename(%q) = %q, %q; want %q, %q", c.in, old, nw, c.old, c.new)
		}
	}
}

func TestAnalyzeHistoryFollowsRenames(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init", "-q", "-b", "main", ".")
	writeFile(t, root, "old/name.go", "package a\n\nfunc A() {}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "add")
	writeFile(t, root, "old/name.go", "package a\n\nfunc A() { println() }\n")
	git(t, root, "commit", "-qam", "edit")
	git(t, root, "mv", "old", "new")
	git(t, root, "commit", "-qm", "move")
	writeFile(t, root, "new/name.go", "package a\n\nfunc A() { println(1) }\n")
	git(t, root, "commit", "-qam", "edit again")

	var paths []string
	_, files, err := New().AnalyzeHistoryWithCommits(context.Background(), root, nil, func(_ models.Commit, fcs []FileChange) error {
		for _, fc := range fcs {
			paths = append(paths, fc.Path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	for _, p := range paths {
		if p != "new/name.go" {
			t.Fatalf("expected every change under the latest name, got %v", paths)
		}
	}
	if fh := files["new/name.go"]; fh == nil || fh.Commits != 4 {
		t.Fatalf("expected 4 commits on new/name.go, got %+v", fh)
	}
	if _, ok := files["old/name.go"]; ok {
		t.Fatal("expected no history under the old name")
	}
}

func TestStreamLogsAppliesMailmap(t *testing.T) {
	root := makeRepo(t)
	writeFile(t, root, ".mailmap", "Real Name <real@example.com> Test <test@example.com>\n")
	var authors []string
	err := New().StreamLogs(context.Background(), root, func(c models.Commit, _ []FileChange) error {
		authors = append(authors, c.Author+" <"+c.Email+">")
		return nil
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	for _, a := range authors {
		if a != "Real Name <real@example.com>" {
			t.Fatalf("expected mailmapped identity, got %v", authors)
		}
	}
}

// mapCache is a ChangeCache over a map, counting what it served.
type mapCache struct {
	changes map[string][]FileChange
	served  int
}

func (m *mapCache) Has(hash string) (bool, error) {
	_, ok := m.changes[hash]
	return ok, nil
}

func (m *mapCache) Changes(hash string) ([]FileChange, error) {
	m.served++
	return m.changes[hash], nil
}

// collect streams history into commit order and changes per commit.
func collect(t *testing.T, stream func(fn func(models.Commit, []FileChange) error) error) ([]string, map[string][]FileChange) {
	t.Helper()
	var order []string
	got := map[string][]FileChange{}
	if err := stream(func(c models.Commit, files []FileChange) error {
		order = append(order, c.Hash)
		got[c.Hash] = files
		return nil
	}); err != nil {
		t.Fatalf("stream: %v", err)
	}
	return order, got
}

func TestStreamCachedMatchesAFullStream(t *testing.T) {
	root := makeRepo(t)
	a := New()
	ctx := context.Background()
	_, before := collect(t, func(fn func(models.Commit, []FileChange) error) error { return a.StreamLogs(ctx, root, fn) })

	// New work after the first scan, including a binary file.
	writeFile(t, root, "d.go", "package d\n\nfunc D() {}\n")
	writeFile(t, root, "logo.bin", "\x00\x01\x02")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "add d")

	cache := &mapCache{changes: before}
	order, cached := collect(t, func(fn func(models.Commit, []FileChange) error) error {
		return a.streamCached(ctx, root, cache, fn)
	})
	fullOrder, full := collect(t, func(fn func(models.Commit, []FileChange) error) error { return a.StreamLogs(ctx, root, fn) })

	if strings.Join(order, ",") != strings.Join(fullOrder, ",") {
		t.Fatalf("history order differs:\ncached %v\nfull   %v", order, fullOrder)
	}
	for hash, files := range full {
		if len(files) != len(cached[hash]) {
			t.Fatalf("commit %s: cached %v, full %v", hash, cached[hash], files)
		}
		for i := range files {
			if files[i].Path != cached[hash][i].Path || files[i].Add != cached[hash][i].Add {
				t.Fatalf("commit %s: cached %v, full %v", hash, cached[hash], files)
			}
		}
	}
	if cache.served != len(before) {
		t.Fatalf("expected the %d known commits served from the cache, got %d", len(before), cache.served)
	}
}

func TestStreamCachedDropsRewrittenHistory(t *testing.T) {
	root := makeRepo(t)
	a := New()
	ctx := context.Background()
	_, before := collect(t, func(fn func(models.Commit, []FileChange) error) error { return a.StreamLogs(ctx, root, fn) })

	// Rewrite the tip: the old tip commit is no longer part of the history.
	git(t, root, "commit", "-q", "--amend", "-m", "merge feature, reworded")
	cache := &mapCache{changes: before}
	order, _ := collect(t, func(fn func(models.Commit, []FileChange) error) error {
		return a.streamCached(ctx, root, cache, fn)
	})
	fullOrder, _ := collect(t, func(fn func(models.Commit, []FileChange) error) error { return a.StreamLogs(ctx, root, fn) })
	if strings.Join(order, ",") != strings.Join(fullOrder, ",") {
		t.Fatalf("expected the rewritten history, got %v want %v", order, fullOrder)
	}
}
