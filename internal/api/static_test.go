package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
)

func TestServesEmbeddedInterface(t *testing.T) {
	ui := fstest.MapFS{
		"index.html":         {Data: []byte("<!doctype html><div id=root></div>")},
		"assets/app-1a2b.js": {Data: []byte("console.log(1)")},
		"favicon.svg":        {Data: []byte("<svg/>")},
	}
	srv := New(Deps{AllowedHosts: config.LoopbackHosts(), UI: ui})
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	cases := []struct {
		path, body, cache string
		status            int
	}{
		{"/", "id=root", "no-cache", 200},
		{"/hotspots", "id=root", "no-cache", 200},
		{"/assets/app-1a2b.js", "console.log", "public, max-age=31536000, immutable", 200},
		{"/favicon.svg", "<svg", "no-cache", 200},
		{"/api/nope", `"error"`, "", 404},
	}
	for _, c := range cases {
		resp, err := http.Get(ts.URL + c.path)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != c.status || !strings.Contains(string(b), c.body) {
			t.Errorf("%s: got %d %q", c.path, resp.StatusCode, b)
		}
		if got := resp.Header.Get("Cache-Control"); got != c.cache {
			t.Errorf("%s: Cache-Control %q, want %q", c.path, got, c.cache)
		}
		if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s: missing nosniff", c.path)
		}
		if c.status == 200 && !strings.Contains(resp.Header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Errorf("%s: missing content security policy", c.path)
		}
	}
}

func TestWithoutInterfaceOnlyTheAPIAnswers(t *testing.T) {
	ts := httptest.NewServer(New(Deps{AllowedHosts: config.LoopbackHosts()}).Router())
	t.Cleanup(ts.Close)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 without an embedded interface, got %d", resp.StatusCode)
	}
}
