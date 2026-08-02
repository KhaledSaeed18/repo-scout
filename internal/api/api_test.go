package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/jobs"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/ws"
)

func gitCommit(t *testing.T, dir, date, msg string) {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "commit", "-qm", msg)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func newTestServer(t *testing.T) (*httptest.Server, *Server) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	write(t, root, "main.go", "package main\n\nimport \"example.com/demo/util\"\n\nfunc main() {}\n")
	write(t, root, "util/util.go", "package util\n\nfunc Help() {}\n")
	write(t, root, "go.mod", "module example.com/demo\n\ngo 1.22\n")
	if err := exec.Command("git", "-C", root, "init", "-q", "-b", "main", ".").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", root, "add", ".").Run(); err != nil {
		t.Fatal(err)
	}
	gitCommit(t, root, "2024-02-01T10:00:00", "initial commit")

	repoStore := database.NewSettingsStore(db)
	hub := ws.New()
	load := func() config.Settings { return config.Defaults() }
	mgr := jobs.New(db, analysis.New(db), load, hub)
	repo := models.Repository{Name: "demo", Path: root}
	if err := db.Create(&repo).Error; err != nil {
		t.Fatal(err)
	}
	if err := analysis.New(db).Run(context.Background(), repo.ID, 1, &nullRep{}, load()); err != nil {
		t.Fatalf("seed analysis: %v", err)
	}

	srv := New(db, mgr, hub, repoStore)
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts, srv
}

type nullRep struct{}

func (nullRep) SetTotal(int)                     {}
func (nullRep) SetProgress(float64)              {}
func (nullRep) Inc(int)                          {}
func (nullRep) SetMessage(string)                {}
func (nullRep) Checkpoint(context.Context) error { return nil }

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

func get(t *testing.T, ts *httptest.Server, path string) (*http.Response, map[string]any) {
	t.Helper()
	resp, err := http.Get(ts.URL + path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp, body
}

func del(t *testing.T, ts *httptest.Server, path string) (*http.Response, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, ts.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var body map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return resp, body
}

func TestHealth(t *testing.T) {
	ts, _ := newTestServer(t)
	resp, _ := get(t, ts, "/api/health")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRepositoryEndpoints(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, _ := get(t, ts, "/api/repositories")
	if resp.StatusCode != 200 {
		t.Fatalf("list repos: %d", resp.StatusCode)
	}

	resp, body := get(t, ts, "/api/repositories/1")
	if resp.StatusCode != 200 {
		t.Fatalf("get repo: %d", resp.StatusCode)
	}
	if body["fileCount"].(float64) != 3 {
		t.Fatalf("expected 3 files, got %v", body["fileCount"])
	}

	for _, p := range []string{
		"/api/repositories/1/files",
		"/api/repositories/1/tree",
		"/api/repositories/1/commits",
		"/api/repositories/1/contributors",
		"/api/repositories/1/heatmap",
		"/api/repositories/1/dependencies",
		"/api/repositories/1/duplicates",
		"/api/repositories/1/architecture",
		"/api/repositories/1/metrics",
	} {
		resp, _ := get(t, ts, p)
		if resp.StatusCode != 200 {
			t.Fatalf("%s: expected 200, got %d", p, resp.StatusCode)
		}
	}

	resp, body = get(t, ts, "/api/search?repo=1&query=package&mode=content")
	if resp.StatusCode != 200 {
		t.Fatalf("search: %d", resp.StatusCode)
	}
	if body["total"].(float64) < 1 {
		t.Fatalf("expected search hits, got %v", body["total"])
	}

	resp, _ = get(t, ts, "/api/repositories/1/svg")
	if resp.StatusCode != 200 {
		t.Fatalf("svg: %d", resp.StatusCode)
	}

	resp, body = get(t, ts, "/api/settings")
	if resp.StatusCode != 200 || body["theme"] == nil {
		t.Fatalf("settings: %d %v", resp.StatusCode, body)
	}

	resp, body = get(t, ts, "/api/jobs")
	if resp.StatusCode != 200 {
		t.Fatalf("jobs: %d", resp.StatusCode)
	}
	if body["jobs"] == nil {
		t.Fatalf("expected jobs list")
	}

	resp, _ = get(t, ts, "/api/repositories/1/export?kind=files&format=csv")
	if resp.StatusCode != 200 {
		t.Fatalf("export: %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Disposition"); !strings.Contains(ct, "files.csv") {
		t.Fatalf("unexpected content disposition %q", ct)
	}

	resp, _ = get(t, ts, "/api/repositories/999")
	if resp.StatusCode != 404 {
		t.Fatalf("missing repo: expected 404, got %d", resp.StatusCode)
	}
}

func TestDeleteRepoCascades(t *testing.T) {
	ts, srv := newTestServer(t)

	if err := srv.db.Create(&models.Job{RepoID: 1, Kind: "scan", Status: "completed"}).Error; err != nil {
		t.Fatal(err)
	}

	var fileCount int64
	srv.db.Model(&models.File{}).Where("repo_id = ?", 1).Count(&fileCount)
	if fileCount == 0 {
		t.Fatal("expected seeded files for repo 1")
	}

	resp, body := del(t, ts, "/api/repositories/1")
	if resp.StatusCode != 200 || body["deleted"] != true {
		t.Fatalf("delete repo: %d %v", resp.StatusCode, body)
	}

	resp, _ = get(t, ts, "/api/repositories/1")
	if resp.StatusCode != 404 {
		t.Fatalf("expected repo gone, got %d", resp.StatusCode)
	}

	for _, tbl := range []any{
		&models.File{}, &models.Commit{}, &models.Branch{}, &models.Tag{},
		&models.Contributor{}, &models.Job{},
	} {
		var n int64
		if err := srv.db.Model(tbl).Where("repo_id = ?", 1).Count(&n).Error; err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Fatalf("expected no %T rows for deleted repo, got %d", tbl, n)
		}
	}
}
