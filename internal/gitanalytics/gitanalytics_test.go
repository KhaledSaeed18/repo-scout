package gitanalytics

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

type nullReporter struct{}

func (nullReporter) SetTotal(int)                         {}
func (nullReporter) SetProgress(float64)                  {}
func (nullReporter) Inc(int)                              {}
func (nullReporter) SetMessage(string)                    {}
func (nullReporter) Checkpoint(ctx context.Context) error { return nil }

func gitEnv(date string) []string {
	env := append(os.Environ(),
		"GIT_AUTHOR_NAME=Alice", "GIT_AUTHOR_EMAIL=alice@example.com",
		"GIT_COMMITTER_NAME=Alice", "GIT_COMMITTER_EMAIL=alice@example.com")
	if date != "" {
		env = append(env, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	}
	return env
}

func gitAt(t *testing.T, dir string, date string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitEnv(date)
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

func newRepo(t *testing.T) (*gorm.DB, models.Repository) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	gitAt(t, root, "", "init", "-q", "-b", "main", ".")
	writeFile(t, root, "a.txt", "line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\n")
	gitAt(t, root, "", "add", ".")
	gitAt(t, root, "2024-01-01T10:00:00", "commit", "-qm", "day one")
	writeFile(t, root, "b.txt", "b\n")
	gitAt(t, root, "", "add", ".")
	gitAt(t, root, "2024-01-02T11:00:00", "commit", "-qm", "day two")
	writeFile(t, root, "c.txt", "c\n")
	gitAt(t, root, "", "add", ".")
	gitAt(t, root, "2024-01-05T09:00:00", "commit", "-qm", "day five")
	repo := models.Repository{Name: "fixture", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	if err := analysis.New(db).Run(context.Background(), repo.ID, 1, nullReporter{}, config.Defaults()); err != nil {
		t.Fatalf("analysis: %v", err)
	}
	return db, repo
}

func TestHeatmap(t *testing.T) {
	db, repo := newRepo(t)
	h, err := ComputeHeatmap(db, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if h.Total != 3 {
		t.Fatalf("expected 3 commits, got %d", h.Total)
	}
	if len(h.Daily) != 3 {
		t.Fatalf("expected 3 active days, got %d: %+v", len(h.Daily), h.Daily)
	}
	var firstCommit models.Commit
	if err := db.Where("repo_id = ?", repo.ID).Order("date ASC").First(&firstCommit).Error; err != nil {
		t.Fatal(err)
	}
	if h.Hourly[int(firstCommit.Date.Weekday())][firstCommit.Date.Hour()] != 1 {
		t.Fatalf("expected 1 commit in weekday/hour bucket of %v, got %d",
			firstCommit.Date, h.Hourly[int(firstCommit.Date.Weekday())][firstCommit.Date.Hour()])
	}
	total := 0
	for _, row := range h.Hourly {
		for _, n := range row {
			total += n
		}
	}
	if total != 3 {
		t.Fatalf("expected hourly total 3, got %d", total)
	}
	if h.Start != "2024-01-01" || h.End != "2024-01-05" {
		t.Fatalf("unexpected range %s..%s", h.Start, h.End)
	}
}

func TestStreaks(t *testing.T) {
	db, repo := newRepo(t)
	s, err := Streaks(db, repo.ID, "alice@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if s.TotalCommits != 3 {
		t.Fatalf("expected 3 commits, got %d", s.TotalCommits)
	}
	if s.ActiveDays != 3 {
		t.Fatalf("expected 3 active days, got %d", s.ActiveDays)
	}
	if len(s.All) != 2 {
		t.Fatalf("expected 2 streaks, got %+v", s.All)
	}
	if s.Longest.Days != 2 || s.Longest.Start != "2024-01-01" {
		t.Fatalf("unexpected longest streak %+v", s.Longest)
	}
	if s.Current.Days != 1 || s.Current.Start != "2024-01-05" {
		t.Fatalf("unexpected current streak %+v", s.Current)
	}
}

func TestOwnership(t *testing.T) {
	db, repo := newRepo(t)
	o, err := ComputeOwnership(db, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if o.Total != 3 {
		t.Fatalf("expected 3 owned files, got %d", o.Total)
	}
	if len(o.ByAuthor) != 1 || o.ByAuthor[0].Author != "Alice" || o.ByAuthor[0].Files != 3 {
		t.Fatalf("unexpected ownership %+v", o.ByAuthor)
	}
}

func TestLargestCommits(t *testing.T) {
	db, repo := newRepo(t)
	lc, err := LargestCommits(db, repo.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(lc) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(lc))
	}
	if lc[0].Message != "day one" {
		t.Fatalf("expected day one first, got %q", lc[0].Message)
	}
}

func TestCommitFeed(t *testing.T) {
	db, repo := newRepo(t)
	feed, err := CommitFeed(db, repo.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(feed))
	}
	if feed[0].Message != "day five" {
		t.Fatalf("expected newest first, got %q", feed[0].Message)
	}
}

func TestLeaderboard(t *testing.T) {
	db, repo := newRepo(t)
	lb, err := Leaderboard(db, repo.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(lb) != 1 || lb[0].Commits != 3 {
		t.Fatalf("unexpected leaderboard %+v", lb)
	}
}
