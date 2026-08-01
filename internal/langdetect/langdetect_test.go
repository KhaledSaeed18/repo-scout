package langdetect

import "testing"

func TestDetect(t *testing.T) {
	cases := map[string]string{
		"main.go":      "Go",
		"lib.rs":       "Rust",
		"app.py":       "Python",
		"Main.java":    "Java",
		"Main.kt":      "Kotlin",
		"app.ts":       "TypeScript",
		"app.tsx":      "TypeScript",
		"index.js":     "JavaScript",
		"index.jsx":    "JavaScript",
		"index.php":    "PHP",
		"Program.cs":   "C#",
		"foo.cpp":      "C++",
		"foo.hpp":      "C++",
		"foo.c":        "C",
		"foo.h":        "C",
		"App.swift":    "Swift",
		"Dockerfile":   "Docker",
		"Makefile":     "Make",
		"package.json": "JSON",
		"unknown.xyz":  "",
		"README.md":    "Markdown",
	}
	for path, want := range cases {
		got := Detect(path)
		if want == "" {
			if got != nil {
				t.Errorf("%s: expected nil, got %s", path, got.Name)
			}
			continue
		}
		if got == nil || got.Name != want {
			t.Errorf("%s: expected %s, got %v", path, want, got)
		}
	}
}

func TestCountGo(t *testing.T) {
	content := `package main

// a line comment
/* block
   comment */
func main() {
	// inline logic
	println("hello")
}

/*
 * full block
 */
var x = 1 /* trailing */
`
	loc := Detect("main.go").Count(content)
	if loc.Code != 5 {
		t.Errorf("code: got %d, want 5", loc.Code)
	}
	if loc.Comments != 7 {
		t.Errorf("comments: got %d, want 7", loc.Comments)
	}
	if loc.Blank != 2 {
		t.Errorf("blank: got %d, want 2", loc.Blank)
	}
	if loc.Lines != 14 {
		t.Errorf("lines: got %d, want 14", loc.Lines)
	}
}

func TestCountPython(t *testing.T) {
	content := "# header\n\ndef f():\n    return 1\n"
	loc := Detect("f.py").Count(content)
	if loc.Code != 2 || loc.Comments != 1 || loc.Blank != 1 || loc.Lines != 4 {
		t.Errorf("py loc mismatch: %+v", loc)
	}
}

func TestCountBlocksAcrossLines(t *testing.T) {
	content := "/* start\nmid\nend */\nafter"
	loc := Detect("x.java").Count(content)
	if loc.Comments != 3 || loc.Code != 1 {
		t.Errorf("block loc mismatch: %+v", loc)
	}
}
