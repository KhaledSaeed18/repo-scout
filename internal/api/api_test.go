package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

	srv := New(Deps{DB: db, Jobs: mgr, Hub: hub, Settings: repoStore, AllowedHosts: config.LoopbackHosts()})
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
		"/api/repositories/1/hotspots?months=0",
		"/api/repositories/1/knowledge?depth=1",
		"/api/repositories/1/coupling?hidden=true",
		"/api/repositories/1/trends",
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
	if cd := resp.Header.Get("Content-Disposition"); cd != "attachment; filename=demo-files.csv" {
		t.Fatalf("unexpected content disposition %q", cd)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected csv content type %q", ct)
	}
	resp, _ = get(t, ts, "/api/repositories/1/export?kind=commits&format=json")
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected json content type %q", ct)
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

func TestMetricsAggregatesAndCapsLists(t *testing.T) {
	ts, _ := newTestServer(t)

	resp, body := get(t, ts, "/api/repositories/1/metrics?limit=1")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics: %d %v", resp.StatusCode, body)
	}
	totals := body["totals"].(map[string]any)
	if totals["files"].(float64) != 3 {
		t.Fatalf("expected 3 files, got %v", totals["files"])
	}
	if got := len(body["largestFiles"].([]any)); got != 1 {
		t.Fatalf("expected largestFiles capped at 1, got %d", got)
	}
	if got := len(body["mostComplexFiles"].([]any)); got != 1 {
		t.Fatalf("expected mostComplexFiles capped at 1, got %d", got)
	}
	if body["deepestFile"] != "util/util.go" || body["maxDepth"].(float64) != 1 {
		t.Fatalf("unexpected deepest file: %v depth %v", body["deepestFile"], body["maxDepth"])
	}
	langs := body["languages"].(map[string]any)
	if g, ok := langs["Go"].(map[string]any); !ok || g["files"].(float64) != 2 {
		t.Fatalf("expected 2 Go files, got %v", langs)
	}
}

func TestTreeEscapesLikeWildcards(t *testing.T) {
	ts, srv := newTestServer(t)
	for _, f := range []models.File{
		{RepoID: 1, Path: "a_b/x/one.go", Name: "one.go", Folder: "a_b/x"},
		{RepoID: 1, Path: "axb/y/two.go", Name: "two.go", Folder: "axb/y"},
	} {
		if err := srv.db.Create(&f).Error; err != nil {
			t.Fatal(err)
		}
	}
	resp, body := get(t, ts, "/api/repositories/1/tree?folder=a_b")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("tree: %d %v", resp.StatusCode, body)
	}
	folders := body["folders"].([]any)
	if len(folders) != 1 || folders[0] != "x" {
		t.Fatalf("expected only subfolder x, got %v", folders)
	}
}

func TestDuplicatesGroupsBlocks(t *testing.T) {
	ts, srv := newTestServer(t)
	groups := []models.DuplicateGroup{{RepoID: 1, Lines: 12}, {RepoID: 1, Lines: 8}}
	if err := srv.db.Create(&groups).Error; err != nil {
		t.Fatal(err)
	}
	blocks := []models.DuplicateBlock{
		{GroupID: groups[0].ID, RepoID: 1, FilePath: "b.go", StartLine: 1, EndLine: 12},
		{GroupID: groups[0].ID, RepoID: 1, FilePath: "a.go", StartLine: 4, EndLine: 15},
		{GroupID: groups[1].ID, RepoID: 1, FilePath: "c.go", StartLine: 2, EndLine: 9},
	}
	if err := srv.db.Create(&blocks).Error; err != nil {
		t.Fatal(err)
	}
	resp, body := get(t, ts, "/api/repositories/1/duplicates")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("duplicates: %d %v", resp.StatusCode, body)
	}
	out := body["groups"].([]any)
	if len(out) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(out))
	}
	first := out[0].(map[string]any)["blocks"].([]any)
	if len(first) != 2 || first[0].(map[string]any)["filePath"] != "a.go" {
		t.Fatalf("expected largest group with blocks sorted by path, got %v", first)
	}
	if second := out[1].(map[string]any)["blocks"].([]any); len(second) != 1 {
		t.Fatalf("expected 1 block in second group, got %v", second)
	}
}

func TestWritesRequireJSONContentType(t *testing.T) {
	ts, _ := newTestServer(t)
	// A cross-site form or text/plain POST is a "simple" request that skips
	// CORS preflight; it must be refused before reaching the handler.
	for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
		req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/repositories", strings.NewReader(`{"path":"/tmp"}`))
		if err != nil {
			t.Fatal(err)
		}
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnsupportedMediaType {
			t.Fatalf("content-type %q: expected 415, got %d", ct, resp.StatusCode)
		}
	}

	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/settings", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("json put: expected 200, got %d", resp.StatusCode)
	}
}

func TestFilesFilterByPath(t *testing.T) {
	ts, _ := newTestServer(t)
	resp, body := get(t, ts, "/api/repositories/1/files?path=util/util.go")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("files: %d %v", resp.StatusCode, body)
	}
	files := body["files"].([]any)
	if len(files) != 1 || files[0].(map[string]any)["path"] != "util/util.go" || body["total"].(float64) != 1 {
		t.Fatalf("expected exactly util/util.go, got %v", body)
	}
}

func TestSearchInvalidRegexIsBadRequest(t *testing.T) {
	ts, _ := newTestServer(t)
	resp, body := get(t, ts, "/api/search?repo=1&mode=regex&query=%28unclosed")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for an invalid pattern, got %d %v", resp.StatusCode, body)
	}
	if msg, _ := body["error"].(string); !strings.Contains(msg, "invalid pattern") {
		t.Fatalf("expected a useful message, got %q", msg)
	}
}

func TestBrowseFlagsGitRepositories(t *testing.T) {
	ts, _ := newTestServer(t)
	root := t.TempDir()
	for _, dir := range []string{"plain", "project/.git", ".hidden"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	resp, body := get(t, ts, "/api/browse?path="+url.QueryEscape(root))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("browse: %d %v", resp.StatusCode, body)
	}
	if body["isRepo"] != false {
		t.Fatalf("root is not a repository: %v", body["isRepo"])
	}
	got := map[string]bool{}
	for _, e := range body["entries"].([]any) {
		m := e.(map[string]any)
		got[m["name"].(string)] = m["isRepo"].(bool)
	}
	if len(got) != 2 || got["plain"] || !got["project"] {
		t.Fatalf("expected plain=false project=true and no hidden folders, got %v", got)
	}

	_, body = get(t, ts, "/api/browse?path="+url.QueryEscape(filepath.Join(root, "project")))
	if body["isRepo"] != true {
		t.Fatalf("expected the project folder to be flagged as a repository")
	}
}

func TestScanRequestReusesActiveJob(t *testing.T) {
	ts, srv := newTestServer(t)
	var repo models.Repository
	srv.db.First(&repo)
	active := models.Job{RepoID: repo.ID, Kind: "scan", Status: models.JobQueued}
	srv.db.Create(&active)

	body := strings.NewReader(`{"path":"` + repo.Path + `"}`)
	resp, err := http.Post(ts.URL+"/api/repositories", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		Job models.Job `json:"job"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Job.ID != active.ID {
		t.Fatalf("expected the queued job %d to be reused, got job %d", active.ID, out.Job.ID)
	}
	var n int64
	srv.db.Model(&models.Job{}).Where("repo_id = ?", repo.ID).Count(&n)
	if n != 1 {
		t.Fatalf("a second scan was queued for the same repository (%d jobs)", n)
	}
}

func TestRejectsUnknownHosts(t *testing.T) {
	ts, _ := newTestServer(t)
	for host, want := range map[string]int{
		"evil.example":      http.StatusForbidden,
		"evil.example:8080": http.StatusForbidden,
		"localhost:5173":    http.StatusOK,
		"127.0.0.1":         http.StatusOK,
		"[::1]:8080":        http.StatusOK,
		"LOCALHOST":         http.StatusOK,
		"127.0.0.1.nip.io":  http.StatusForbidden,
	} {
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/health", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = host
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("Host %q: expected %d, got %d", host, want, resp.StatusCode)
		}
	}
}
