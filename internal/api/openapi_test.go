package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/legacy"
	"github.com/go-chi/chi/v5"
)

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(openAPISpec)
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("invalid spec: %v", err)
	}
	return doc
}

// TestSpecCoversEveryRoute keeps the spec and the router listing the same
// operations.
func TestSpecCoversEveryRoute(t *testing.T) {
	doc := loadSpec(t)
	documented := map[string]bool{}
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			documented[method+" "+path] = true
		}
	}
	routed := map[string]bool{}
	router := New(Deps{}).Router().(chi.Routes)
	err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if route == "/api/ws" || !strings.HasPrefix(route, "/api/") {
			return nil // the WebSocket upgrade and the interface are not JSON operations
		}
		routed[method+" "+strings.TrimSuffix(route, "/")] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var missing, stale []string
	for op := range routed {
		if !documented[op] {
			missing = append(missing, op)
		}
	}
	for op := range documented {
		if !routed[op] {
			stale = append(stale, op)
		}
	}
	sort.Strings(missing)
	sort.Strings(stale)
	if len(missing) > 0 || len(stale) > 0 {
		t.Fatalf("spec out of date\nrouted but undocumented: %v\ndocumented but not routed: %v", missing, stale)
	}
}

// TestResponsesMatchSpec calls each operation on a scanned fixture and
// validates the status, headers and body against the spec.
func TestResponsesMatchSpec(t *testing.T) {
	doc := loadSpec(t)
	doc.Servers = nil // match the test server's random port
	specRouter, err := legacy.NewRouter(doc)
	if err != nil {
		t.Fatal(err)
	}
	ts, _ := newTestServer(t)
	_, repo := get(t, ts, "/api/repositories/1")
	root := repo["path"].(string)

	calls := []struct{ method, path, body string }{
		{"GET", "/api/health", ""},
		{"GET", "/api/browse?path=" + root, ""},
		{"GET", "/api/repositories", ""},
		{"GET", "/api/repositories/1", ""},
		{"GET", "/api/repositories/1/files?sort=loc&limit=2", ""},
		{"GET", "/api/repositories/1/tree", ""},
		{"GET", "/api/repositories/1/tree?folder=util", ""},
		{"GET", "/api/repositories/1/commits", ""},
		{"GET", "/api/repositories/1/contributors", ""},
		{"GET", "/api/repositories/1/largest-commits", ""},
		{"GET", "/api/repositories/1/ownership", ""},
		{"GET", "/api/repositories/1/branches", ""},
		{"GET", "/api/repositories/1/tags", ""},
		{"GET", "/api/repositories/1/heatmap", ""},
		{"GET", "/api/repositories/1/knowledge", ""},
		{"GET", "/api/repositories/1/dependencies", ""},
		{"GET", "/api/repositories/1/duplicates", ""},
		{"GET", "/api/repositories/1/architecture", ""},
		{"GET", "/api/repositories/1/coupling", ""},
		{"GET", "/api/repositories/1/metrics", ""},
		{"GET", "/api/repositories/1/hotspots?months=0", ""},
		{"GET", "/api/repositories/1/trends", ""},
		{"GET", "/api/repositories/1/export?kind=commits&format=json", ""},
		{"GET", "/api/repositories/1/export?kind=files", ""},
		{"GET", "/api/search?repo=1&query=package", ""},
		{"GET", "/api/search?repo=1&query=util&mode=filename", ""},
		{"GET", "/api/search?repo=1&query=nothingmatchesthis&mode=filename", ""},
		{"GET", "/api/search?repo=1&query=nothingmatchesthis", ""},
		{"GET", "/api/search?repo=1&query=(&mode=regex", ""},
		{"GET", "/api/settings", ""},
		{"PUT", "/api/settings", ""}, // body filled from GET below
		{"GET", "/api/repositories/999", ""},
		{"POST", "/api/repositories", `{"path":"/definitely/not/here"}`},
		{"POST", "/api/repositories", `{"path":"` + root + `"}`},
		{"GET", "/api/jobs", ""},
		{"POST", "/api/jobs/1/cancel", `{}`},
		{"POST", "/api/jobs/999/pause", `{}`},
		{"DELETE", "/api/repositories/1", ""},
	}
	var settings []byte
	for _, c := range calls {
		body := c.body
		if c.method == "PUT" {
			body = string(settings)
		}
		req, err := http.NewRequest(c.method, ts.URL+c.path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if c.method == "GET" && c.path == "/api/settings" {
			settings = data
		}

		route, params, err := specRouter.FindRoute(req)
		if err != nil {
			t.Errorf("%s %s: not in the spec: %v", c.method, c.path, err)
			continue
		}
		input := &openapi3filter.RequestValidationInput{Request: req, PathParams: params, Route: route}
		err = openapi3filter.ValidateResponse(context.Background(), &openapi3filter.ResponseValidationInput{
			RequestValidationInput: input,
			Status:                 resp.StatusCode,
			Header:                 resp.Header,
			Body:                   io.NopCloser(bytes.NewReader(data)),
			Options:                &openapi3filter.Options{IncludeResponseStatus: true, MultiError: true},
		})
		if err != nil {
			t.Errorf("%s %s (%d): %v", c.method, c.path, resp.StatusCode, err)
		}
	}
}

func TestServesTheSpec(t *testing.T) {
	ts := httptest.NewServer(New(Deps{AllowedHosts: []string{"127.0.0.1"}}).Router())
	t.Cleanup(ts.Close)
	resp, err := http.Get(ts.URL + "/api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || resp.Header.Get("Content-Type") != "application/yaml" || !bytes.HasPrefix(data, []byte("openapi: 3.0.3")) {
		t.Fatalf("unexpected spec response %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}
