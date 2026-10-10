package analysis

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
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

// reporter captures progress for assertions. Scan stages report from
// multiple worker goroutines, so it must be safe for concurrent use, same
// as the real jobs.Reporter implementation.
type reporter struct {
	mu      sync.Mutex
	lastMsg string
}

func (r *reporter) SetTotal(n int)        {}
func (r *reporter) SetProgress(f float64) {}
func (r *reporter) Inc(n int)             {}
func (r *reporter) SetMessage(msg string) {
	r.mu.Lock()
	r.lastMsg = msg
	r.mu.Unlock()
}
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

	var changed []models.CommitFile
	db.Where("repo_id = ?", repo.ID).Find(&changed)
	if len(changed) != 6 {
		t.Fatalf("expected 6 commit file rows (5 added, 1 edited), got %d", len(changed))
	}
	var owner models.FileOwnership
	if err := db.Where("repo_id = ? AND path = ?", repo.ID, "pkg/util/util.go").First(&owner).Error; err != nil {
		t.Fatalf("ownership: %v", err)
	}
	if owner.Email != "test@example.com" || owner.Commits != 2 || owner.Share != 1 {
		t.Fatalf("unexpected ownership %+v", owner)
	}

	// FTS index populated
	var fts int64
	db.Raw("SELECT count(*) FROM file_fts WHERE repo_id = ?", repo.ID).Scan(&fts)
	if fts == 0 {
		t.Fatalf("expected fts rows")
	}
}

func TestFailedRescanKeepsPreviousResults(t *testing.T) {
	db := testDB(t)
	root := makeFixture(t)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	r := New(db)
	if err := r.Run(context.Background(), repo.ID, 1, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	var before int64
	db.Model(&models.File{}).Where("repo_id = ?", repo.ID).Count(&before)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Run(ctx, repo.ID, 2, &cancelledReporter{}, config.Defaults()); err == nil {
		t.Fatal("expected the cancelled rescan to fail")
	}

	db.First(&repo, repo.ID)
	if repo.Status != models.RepoReady {
		t.Fatalf("expected repo back to ready, got %s", repo.Status)
	}
	var after int64
	db.Model(&models.File{}).Where("repo_id = ?", repo.ID).Count(&after)
	if after != before || after == 0 {
		t.Fatalf("expected %d files kept, got %d", before, after)
	}
	var staged int64
	db.Model(&models.File{}).Where("repo_id = ?", database.StagingID(repo.ID)).Count(&staged)
	if staged != 0 {
		t.Fatalf("expected staged rows dropped, got %d", staged)
	}
	var snapshots int64
	db.Model(&models.ScanSnapshot{}).Where("repo_id = ?", repo.ID).Count(&snapshots)
	if snapshots != 1 {
		t.Fatalf("expected only the successful scan snapshotted, got %d", snapshots)
	}
}

func TestEveryScanAddsASnapshot(t *testing.T) {
	db := testDB(t)
	root := makeFixture(t)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	r := New(db)
	if err := r.Run(context.Background(), repo.ID, 1, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	write(t, root, "pkg/extra/extra.go", "package extra\n\nfunc E(a int) int {\n\tif a > 0 {\n\t\treturn a\n\t}\n\treturn 0\n}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "extra")
	if err := r.Run(context.Background(), repo.ID, 2, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("second scan: %v", err)
	}

	var snaps []models.ScanSnapshot
	db.Where("repo_id = ?", repo.ID).Order("id").Find(&snaps)
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}
	first, second := snaps[0], snaps[1]
	if first.FileCount != 5 || second.FileCount != 6 {
		t.Fatalf("expected 5 then 6 files, got %d and %d", first.FileCount, second.FileCount)
	}
	if second.CommitCount != first.CommitCount+1 || second.Complexity <= first.Complexity || second.Functions != first.Functions+1 {
		t.Fatalf("expected growth between scans, got %+v then %+v", first, second)
	}
	if second.HeadCommit == "" || second.HeadCommit == first.HeadCommit {
		t.Fatalf("expected a new head commit, got %q then %q", first.HeadCommit, second.HeadCommit)
	}
}

// cancelledReporter fails every checkpoint the way a cancelled job does.
type cancelledReporter struct{ reporter }

func (r *cancelledReporter) Checkpoint(ctx context.Context) error { return ctx.Err() }

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

func TestDuplicateBlocksReferenceStoredGroups(t *testing.T) {
	db := testDB(t)
	// Another repository's group occupies the first IDs, as on any real
	// database after the first scan.
	if err := db.Create(&models.DuplicateGroup{RepoID: 99, Lines: 6}).Error; err != nil {
		t.Fatal(err)
	}
	root := makeFixture(t)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	if err := New(db).Run(context.Background(), repo.ID, 1, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("run pipeline: %v", err)
	}

	var groups []models.DuplicateGroup
	db.Where("repo_id = ?", repo.ID).Find(&groups)
	var blocks []models.DuplicateBlock
	db.Where("repo_id = ?", repo.ID).Find(&blocks)
	if len(groups) == 0 || len(blocks) == 0 {
		t.Fatalf("expected duplicate groups and blocks, got %d and %d", len(groups), len(blocks))
	}
	ids := map[uint]bool{}
	for _, g := range groups {
		ids[g.ID] = true
	}
	for _, b := range blocks {
		if !ids[b.GroupID] {
			t.Fatalf("block %s points at group %d, which is not one of this repo's groups", b.FilePath, b.GroupID)
		}
	}
}

func TestScanOfARemovedFolderFails(t *testing.T) {
	db := testDB(t)
	root := makeFixture(t)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	r := New(db)
	if err := r.Run(context.Background(), repo.ID, 1, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), repo.ID, 2, &reporter{}, config.Defaults()); err == nil {
		t.Fatal("expected the scan of a removed folder to fail")
	}
	db.First(&repo, repo.ID)
	var files int64
	db.Model(&models.File{}).Where("repo_id = ?", repo.ID).Count(&files)
	if repo.Status != models.RepoReady || files != 5 {
		t.Fatalf("expected the previous results kept and the repo ready, got %s with %d files", repo.Status, files)
	}
}

// history captures what a scan stored about the history, keyed by content
// rather than row IDs, so two scans can be compared.
func history(t *testing.T, db *gorm.DB, repoID uint) map[string]string {
	t.Helper()
	out := map[string]string{}
	var changes []struct {
		Hash, Path string
		Additions  int
		Deletions  int
	}
	if err := db.Raw(`SELECT c.hash, cf.path, cf.additions, cf.deletions FROM commit_files cf
		JOIN commits c ON c.id = cf.commit_id WHERE cf.repo_id = ?`, repoID).Scan(&changes).Error; err != nil {
		t.Fatal(err)
	}
	for _, c := range changes {
		out["change "+c.Hash+" "+c.Path] = fmt.Sprintf("+%d -%d", c.Additions, c.Deletions)
	}
	var owners []models.FileOwnership
	db.Where("repo_id = ?", repoID).Find(&owners)
	for _, o := range owners {
		out["owner "+o.Path] = fmt.Sprintf("%s %d %.2f", o.Email, o.Commits, o.Share)
	}
	var people []models.Contributor
	db.Where("repo_id = ?", repoID).Find(&people)
	for _, p := range people {
		out["contributor "+p.Email] = fmt.Sprintf("%d +%d -%d", p.Commits, p.Insertions, p.Deletions)
	}
	var files []models.File
	db.Where("repo_id = ?", repoID).Find(&files)
	for _, f := range files {
		out["file "+f.Path] = fmt.Sprintf("%d %s", f.Commits, f.Author)
	}
	return out
}

func TestIncrementalRescanMatchesAFullScan(t *testing.T) {
	db := testDB(t)
	root := makeFixture(t)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	r := New(db)
	if err := r.Run(context.Background(), repo.ID, 1, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("first scan: %v", err)
	}

	// New history: an edit, a rename and a new file.
	write(t, root, "pkg/util/util.go", "package util\n\nfunc Help() {\n\tprintln(\"hi\")\n}\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "edit helper")
	git(t, root, "mv", "pkg/dup/two.go", "pkg/dup/second.go")
	write(t, root, "pkg/extra.go", "package pkg\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "rename and add")

	db.First(&repo, repo.ID)
	if r.historyCache(&repo) == nil {
		t.Fatal("expected the previous scan to be usable as a cache")
	}
	if err := r.Run(context.Background(), repo.ID, 2, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("incremental rescan: %v", err)
	}

	fresh := testDB(t)
	clean := models.Repository{Name: "demo", Path: root}
	if err := fresh.Create(&clean).Error; err != nil {
		t.Fatal(err)
	}
	if err := New(fresh).Run(context.Background(), clean.ID, 1, &reporter{}, config.Defaults()); err != nil {
		t.Fatalf("full scan: %v", err)
	}

	got, want := history(t, db, repo.ID), history(t, fresh, clean.ID)
	if len(got) != len(want) {
		t.Errorf("incremental stored %d facts, full scan %d", len(got), len(want))
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: incremental %q, full %q", k, got[k], v)
		}
	}
	if _, ok := got["file pkg/dup/second.go"]; !ok {
		t.Error("expected the renamed file in the results")
	}
}
