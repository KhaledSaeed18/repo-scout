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

func TestAnalyzeGoParsesRatherThanMatches(t *testing.T) {
	content := "package demo\n\n" +
		"import \"regexp\"\n\n" +
		"// The keywords in this pattern and comment must not count: if for case && ||\n" +
		"var pattern = regexp.MustCompile(`\\b(if|for|case)\\b && ||`)\n\n" +
		"type Shape interface{ Area() float64 }\n\n" +
		"type square struct{ side float64 }\n\n" +
		"// Area is a method; methods are functions too.\n" +
		"func (s square) Area() float64 { return s.side * s.side }\n\n" +
		"func Classify(n int, ok bool) string {\n" +
		"\tcheck := func(v int) bool { return v > 0 && ok }\n" +
		"\tswitch {\n" +
		"\tcase n < 0:\n" +
		"\t\treturn \"negative\"\n" +
		"\tcase n == 0 || !check(n):\n" +
		"\t\treturn \"zero\"\n" +
		"\tdefault:\n" +
		"\t\tfor i := range n {\n" +
		"\t\t\tif i > 10 {\n" +
		"\t\t\t\treturn \"big\"\n" +
		"\t\t\t}\n" +
		"\t\t}\n" +
		"\t}\n" +
		"\treturn \"small\"\n" +
		"}\n"
	r := Analyze(models.File{Language: "Go"}, content)
	// Area: 1. Classify: 1 + closure && + 2 cases + || + range + if = 7.
	if r.Complexity != 8 {
		t.Errorf("complexity: got %d, want 8", r.Complexity)
	}
	if r.FuncCount != 2 || r.MaxFuncLen != 16 {
		t.Errorf("functions: got %d, longest %d; want 2, longest 16", r.FuncCount, r.MaxFuncLen)
	}
	// Shape, Area and Classify; pattern and square are unexported.
	if r.Imports != 1 || r.Exports != 3 {
		t.Errorf("imports %d exports %d; want 1 and 3", r.Imports, r.Exports)
	}
}

func TestAnalyzeGoFallsBackOnSyntaxErrors(t *testing.T) {
	r := Analyze(models.File{Language: "Go"}, "package demo\n\nfunc broken( {\n\tif x {\n}\n")
	if r.Complexity < 1 {
		t.Fatalf("expected the pattern analyzer to measure a file that does not parse, got %+v", r)
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
