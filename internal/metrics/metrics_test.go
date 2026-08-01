package metrics

import (
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func TestAnalyzeGo(t *testing.T) {
	content := `package demo

import (
	"fmt"
	"strings"
)

// ExportedFunc does a thing.
func ExportedFunc(x int) int {
	if x > 0 {
		return x + 1
	}
	return 0
}

func unexported() {
	for i := 0; i < 10; i++ {
		_ = i && true
	}
}

var Version = "1"
`
	f := models.File{Language: "Go"}
	r := Analyze(f, content)
	if r.Imports != 2 {
		t.Errorf("imports: got %d, want 2", r.Imports)
	}
	if r.Exports != 2 {
		t.Errorf("exports: got %d, want 2 (ExportedFunc, Version)", r.Exports)
	}
	if r.Complexity < 1 {
		t.Errorf("complexity should be at least 1, got %d", r.Complexity)
	}
	if r.FuncCount != 2 {
		t.Errorf("func count: got %d, want 2", r.FuncCount)
	}
	if r.MaxFuncLen < 6 {
		t.Errorf("max func len should be >= 6, got %d", r.MaxFuncLen)
	}
	if r.AvgNesting <= 0 {
		t.Errorf("nesting should be > 0, got %f", r.AvgNesting)
	}
}

func TestAnalyzePython(t *testing.T) {
	content := `import os
import sys


def greet(name):
    if not name:
        return "hi"
    return f"hello {name}"


class Person:
    def __init__(self, name):
        self.name = name
`
	f := models.File{Language: "Python"}
	r := Analyze(f, content)
	if r.Imports != 2 {
		t.Errorf("imports: got %d, want 2", r.Imports)
	}
	if r.Exports != 2 {
		t.Errorf("exports: got %d, want 2 (greet, Person)", r.Exports)
	}
	if r.FuncCount != 2 {
		t.Errorf("func count: got %d, want 2", r.FuncCount)
	}
	if r.AvgNesting <= 0 {
		t.Errorf("nesting should be > 0, got %f", r.AvgNesting)
	}
}

func TestAnalyzeTypeScript(t *testing.T) {
	content := `import { readFile } from "fs";
import path from "path";

export function load(): string {
	if (process.env.X) {
		return path.join("a", "b");
	}
	return "";
}

export const VERSION = "1";
`
	f := models.File{Language: "TypeScript"}
	r := Analyze(f, content)
	if r.Imports != 2 {
		t.Errorf("imports: got %d, want 2", r.Imports)
	}
	if r.Exports != 2 {
		t.Errorf("exports: got %d, want 2", r.Exports)
	}
	if r.FuncCount < 1 {
		t.Errorf("func count: got %d, want >= 1", r.FuncCount)
	}
}

func TestUnknownLanguage(t *testing.T) {
	f := models.File{Language: "Plain Text"}
	if r := Analyze(f, "hello\n"); r != (Result{}) {
		t.Errorf("expected empty result, got %+v", r)
	}
}
