package architecture

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/langdetect"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

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

func readAll(root string) func(rel string) (string, error) {
	return func(rel string) (string, error) {
		data, err := os.ReadFile(filepath.Join(root, rel))
		return string(data), err
	}
}

func fileList(root string) []models.File {
	files := []models.File{}
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		f := models.File{Path: rel}
		if lang := langdetect.Detect(rel); lang != nil {
			f.Language = lang.Name
		}
		files = append(files, f)
		return nil
	})
	return files
}

func TestGoImportGraph(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module example.com/demo\n\ngo 1.22\n")
	write(t, root, "main.go", `package main

import (
	"example.com/demo/pkg/util"
)

func main() {}
`)
	write(t, root, "pkg/util/util.go", "package util\n")
	write(t, root, "pkg/api/api.go", "package api\n")

	files := fileList(root)
	rep, err := Build(root, files, readAll(root))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	found := false
	for _, e := range rep.Edges {
		if e.From == "main.go" && e.To == "pkg/util" && e.Resolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected main.go -> pkg/util edge, got %+v", rep.Edges)
	}
	if len(rep.EntryPoints) != 1 || rep.EntryPoints[0] != "main.go" {
		t.Fatalf("expected main.go entry point, got %+v", rep.EntryPoints)
	}
}

func TestCircularDependency(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module example.com/demo\n")
	write(t, root, "a/a.go", "package a\nimport \"example.com/demo/b\"\n")
	write(t, root, "b/b.go", "package b\nimport \"example.com/demo/a\"\n")
	write(t, root, "main.go", "package main\nimport \"example.com/demo/a\"\nfunc main(){}\n")

	files := fileList(root)
	rep, err := Build(root, files, readAll(root))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(rep.Cycles) != 1 {
		t.Fatalf("expected 1 cycle, got %+v", rep.Cycles)
	}
	cycle := rep.Cycles[0]
	if len(cycle) != 2 {
		t.Fatalf("expected 2-node cycle, got %+v", cycle)
	}
}

func TestDeadFiles(t *testing.T) {
	root := t.TempDir()
	write(t, root, "main.go", "package main\n")
	write(t, root, "used.go", "package main\n")
	write(t, root, "orphan.go", "package main\n")

	files := fileList(root)
	rep, err := Build(root, files, readAll(root))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	// main.go and used.go are reachable only via entry points; used.go has no
	// incoming edge, so only main.go is reachable.
	if len(rep.DeadFiles) != 2 {
		t.Fatalf("expected 2 dead files, got %+v", rep.DeadFiles)
	}
}

func TestJSImportGraph(t *testing.T) {
	root := t.TempDir()
	write(t, root, "src/index.ts", `import { helper } from "./lib/helper";`)
	write(t, root, "src/lib/helper.ts", "export const helper = 1;\n")
	write(t, root, "src/util/other.ts", "export const x = 2;\n")

	files := fileList(root)
	rep, err := Build(root, files, readAll(root))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	found := false
	for _, e := range rep.Edges {
		if e.From == "src/index.ts" && e.To == "src/lib/helper.ts" && e.Resolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected src/index.ts -> src/lib/helper.ts, got %+v", rep.Edges)
	}
}

func TestJSImportGraphJsToTs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "src/index.ts", `import { helper } from "./lib/helper.js";`)
	write(t, root, "src/lib/helper.ts", "export const helper = 1;\n")

	files := fileList(root)
	rep, err := Build(root, files, readAll(root))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	found := false
	for _, e := range rep.Edges {
		if e.From == "src/index.ts" && e.To == "src/lib/helper.ts" && e.Resolved {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected src/index.ts -> src/lib/helper.ts via .js->.ts, got %+v", rep.Edges)
	}
}

func TestUnusedModules(t *testing.T) {
	root := t.TempDir()
	write(t, root, "go.mod", "module example.com/demo\n")
	write(t, root, "main.go", "package main\n")
	write(t, root, "pkg/a/a.go", "package a\n")
	write(t, root, "pkg/b/b.go", "package b\nimport \"example.com/demo/pkg/a\"\n")

	files := fileList(root)
	rep, err := Build(root, files, readAll(root))
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	// pkg/b has an incoming edge, pkg/a does not.
	if len(rep.UnusedModules) != 1 {
		t.Fatalf("expected 1 unused module, got %+v", rep.UnusedModules)
	}
}
