package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fixture is a small Git repository with two folders importing each other.
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"go.mod":  "module example.com/demo\n\ngo 1.22\n",
		"main.go": "package main\n\nimport \"example.com/demo/a\"\n\nfunc main() { a.A() }\n",
		"a/a.go":  "package a\n\nimport \"example.com/demo/b\"\n\nfunc A() { b.B() }\n",
		"b/b.go":  "package b\n\nimport \"example.com/demo/a\"\n\nfunc B() {\n\tif true {\n\t\ta.A()\n\t}\n}\n",
	}
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q", "-b", "main", "."}, {"add", "."}, {"commit", "-qm", "init"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@x")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return root
}

func TestScanJSON(t *testing.T) {
	root := fixture(t)
	var out, errOut bytes.Buffer
	if code := run([]string{"scan", "--quiet", "--format", "json", root}, &out, &errOut); code != exitOK {
		t.Fatalf("exit %d: %s", code, errOut.String())
	}
	var rep struct {
		Repository string
		Summary    struct{ Files, Commits int }
		Cycles     []struct{ Folders []string }
	}
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("invalid json: %v\n%s", err, out.String())
	}
	if rep.Repository != root || rep.Summary.Files != 4 || rep.Summary.Commits != 1 || len(rep.Cycles) != 1 {
		t.Fatalf("unexpected report: %+v", rep)
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no progress with --quiet, got %q", errOut.String())
	}
}

func TestScanGateFailsTheRun(t *testing.T) {
	root := fixture(t)
	report := filepath.Join(t.TempDir(), "out.sarif")
	var out, errOut bytes.Buffer
	code := run([]string{"scan", "--format", "sarif", "--output", report, "--fail-on", "cycles", root}, &out, &errOut)
	if code != exitGateFailed {
		t.Fatalf("expected exit %d for a failing gate, got %d: %s", exitGateFailed, code, errOut.String())
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"ruleId": "RS001"`) || !strings.Contains(errOut.String(), "import graph") {
		t.Fatalf("expected a cycle finding and progress lines, got %s / %s", data, errOut.String())
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"scan", "--format", "xml"},
		{"scan", "--fail-on", "everything"},
		{"scan", "/definitely/not/here"},
		{"scan", "a", "b"},
		{"serve", "extra"},
		{"frobnicate"},
	} {
		var out, errOut bytes.Buffer
		if code := run(args, &out, &errOut); code != exitUsage {
			t.Errorf("%v: expected exit %d, got %d", args, exitUsage, code)
		}
	}
}

func TestVersion(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"-v"}} {
		var out bytes.Buffer
		if code := run(args, &out, &out); code != exitOK || out.String() != "repo-scout dev\n" {
			t.Fatalf("%v: unexpected version output %q (exit %d)", args, out.String(), code)
		}
	}
	var out bytes.Buffer
	if code := run([]string{"--help"}, &out, &out); code != exitOK || !strings.Contains(out.String(), "repo-scout scan") {
		t.Fatalf("unexpected help output %q (exit %d)", out.String(), code)
	}
}

func TestScanAgainstABase(t *testing.T) {
	root := fixture(t)
	// The fixture's only commit has a cycle; make a base without one, then
	// bring the cycle back on top of it.
	gitIn := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=T", "GIT_AUTHOR_EMAIL=t@x", "GIT_COMMITTER_NAME=T", "GIT_COMMITTER_EMAIL=t@x")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	acyclic := "package b\n\nfunc B() {}\n"
	if err := os.WriteFile(filepath.Join(root, "b/b.go"), []byte(acyclic), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn("commit", "-qam", "break the cycle")
	gitIn("tag", "base")
	cyclic := "package b\n\nimport \"example.com/demo/a\"\n\nfunc B() {\n\tif true {\n\t\ta.A()\n\t}\n}\n"
	if err := os.WriteFile(filepath.Join(root, "b/b.go"), []byte(cyclic), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn("commit", "-qam", "bring it back")

	var out, errOut bytes.Buffer
	code := run([]string{"scan", "--quiet", "--format", "json", "--base", "base", "--fail-on", "new-cycles", root}, &out, &errOut)
	if code != exitGateFailed {
		t.Fatalf("expected the new cycle to fail the run, got exit %d: %s", code, errOut.String())
	}
	var rep struct {
		Comparison struct {
			Base              string
			NewCycles         []struct{ Folders []string }
			ComplexityChanges []struct {
				Path          string
				Before, After int
			}
		}
		Gates []struct {
			Name   string
			Passed bool
		}
	}
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	c := rep.Comparison
	if c.Base != "base" || len(c.NewCycles) != 1 {
		t.Fatalf("expected one new cycle against base, got %+v", c)
	}
	if len(c.ComplexityChanges) != 1 || c.ComplexityChanges[0].Path != "b/b.go" || c.ComplexityChanges[0].After <= c.ComplexityChanges[0].Before {
		t.Fatalf("expected b/b.go to grow more complex, got %+v", c.ComplexityChanges)
	}
	if status, _ := exec.Command("git", "-C", root, "status", "--porcelain").Output(); len(status) != 0 {
		t.Fatalf("comparing must not touch the repository: %s", status)
	}

	if code := run([]string{"scan", "--quiet", "--fail-on", "new-cycles", root}, &out, &errOut); code != exitUsage {
		t.Fatalf("new-cycles without --base should be a usage error, got %d", code)
	}
	if code := run([]string{"scan", "--quiet", "--base", "no-such-ref", root}, &out, &errOut); code != exitError {
		t.Fatalf("an unknown base should fail the scan, got %d", code)
	}
}
