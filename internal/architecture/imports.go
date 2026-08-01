package architecture

import (
	"regexp"
	"strings"
)

var (
	goImportRe   = regexp.MustCompile(`(?m)^\s*import\s+`)
	jsImportRe   = regexp.MustCompile(`(?:import\s+(?:[^"']*?\s+from\s+)?["']([^"']+)["']|import\(\s*["']([^"']+)["']\s*\)|require\(\s*["']([^"']+)["']\s*\))`)
	pyImportRe   = regexp.MustCompile(`(?m)^\s*(?:import|from)\s+([\w.]+)`)
	rustUseRe    = regexp.MustCompile(`\buse\s+([^;{]+)`)
	javaImportRe = regexp.MustCompile(`(?m)^\s*import\s+(?:static\s+)?([\w.]+)`)
	csUsingRe    = regexp.MustCompile(`(?m)^\s*using\s+([\w.]+)`)
	cppIncludeRe = regexp.MustCompile(`#include\s+([<"])([^>"]+)[>"]`)
	phpUseRe     = regexp.MustCompile(`(?m)^\s*use\s+([\w\\]+)`)
	swiftUseRe   = regexp.MustCompile(`(?m)^\s*import\s+(\w+)`)
)

// extract returns the raw import specs referenced by a file.
func extract(lang, content string) []string {
	switch lang {
	case "Go":
		return extractGo(content)
	case "TypeScript", "JavaScript":
		var out []string
		for _, m := range jsImportRe.FindAllStringSubmatch(content, -1) {
			for _, g := range m[1:] {
				if g != "" {
					out = append(out, g)
				}
			}
		}
		return out
	case "Python":
		var out []string
		for _, m := range pyImportRe.FindAllStringSubmatch(content, -1) {
			out = append(out, m[1])
		}
		return out
	case "Rust":
		var out []string
		for _, m := range rustUseRe.FindAllStringSubmatch(content, -1) {
			for _, part := range strings.Split(m[1], ",") {
				spec := strings.TrimSpace(part)
				if spec != "" {
					out = append(out, spec)
				}
			}
		}
		return out
	case "Java", "Kotlin":
		var out []string
		for _, m := range javaImportRe.FindAllStringSubmatch(content, -1) {
			out = append(out, m[1])
		}
		return out
	case "C#":
		var out []string
		for _, m := range csUsingRe.FindAllStringSubmatch(content, -1) {
			out = append(out, m[1])
		}
		return out
	case "C", "C++":
		var out []string
		for _, m := range cppIncludeRe.FindAllStringSubmatch(content, -1) {
			if m[1] == "\"" {
				out = append(out, m[2])
			}
		}
		return out
	case "PHP":
		var out []string
		for _, m := range phpUseRe.FindAllStringSubmatch(content, -1) {
			out = append(out, strings.TrimPrefix(m[1], "\\"))
		}
		return out
	case "Swift":
		var out []string
		for _, m := range swiftUseRe.FindAllStringSubmatch(content, -1) {
			out = append(out, m[1])
		}
		return out
	}
	return nil
}

func extractGo(content string) []string {
	var specs []string
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case inBlock:
			if t == ")" {
				inBlock = false
				continue
			}
			if i := strings.Index(t, `"`); i >= 0 {
				if j := strings.LastIndex(t[i+1:], `"`); j >= 0 {
					specs = append(specs, t[i+1:i+1+j])
				}
			}
		case strings.HasPrefix(t, "import ("):
			inBlock = true
		case strings.HasPrefix(t, "import "):
			rest := strings.TrimPrefix(t, "import ")
			if i := strings.Index(rest, `"`); i >= 0 {
				if j := strings.LastIndex(rest[i+1:], `"`); j >= 0 {
					specs = append(specs, rest[i+1:i+1+j])
				}
			}
		}
	}
	return specs
}
