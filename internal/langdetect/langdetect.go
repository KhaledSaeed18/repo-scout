// Package langdetect identifies a file's programming language and measures its
// physical lines of code using a comment-aware line counter.
package langdetect

import (
	"path/filepath"
	"strings"
)

// LOC is a line-of-code breakdown for one file.
type LOC struct {
	Lines    int
	Code     int
	Comments int
	Blank    int
}

// Lang describes a language's file signatures and comment syntax.
type Lang struct {
	Name          string
	Extensions    []string
	Filenames     []string
	LineComments  []string
	BlockComments [][2]string
}

// Detect resolves the language for a file path, returning nil when unknown.
func Detect(path string) *Lang {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		ext = ext[1:]
	}
	if l, ok := byExtension[ext]; ok {
		return l
	}
	base := filepath.Base(path)
	if l, ok := byFilename[base]; ok {
		return l
	}
	return nil
}

func (l *Lang) matches(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != "" {
		ext = ext[1:]
	}
	for _, e := range l.Extensions {
		if e == ext {
			return true
		}
	}
	base := filepath.Base(path)
	for _, f := range l.Filenames {
		if f == base {
			return true
		}
	}
	return false
}

// Count measures lines of code, comments, and blanks for the language. It
// tracks multi-line block comments and counts a line with code before a block
// comment opener as code.
func (l *Lang) Count(content string) LOC {
	loc := LOC{}
	var blockClose string
	inBlock := false

	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		loc.Lines++
		trimmed := strings.TrimSpace(line)
		switch {
		case inBlock:
			if idx := strings.Index(line, blockClose); idx >= 0 {
				inBlock = false
				if strings.TrimSpace(line[idx+len(blockClose):]) == "" {
					loc.Comments++
				} else {
					loc.Code++
				}
			} else {
				loc.Comments++
			}
		case trimmed == "":
			loc.Blank++
		default:
			if open := firstBlockStart(line, l.BlockComments); open != "" {
				if strings.TrimSpace(beforeToken(line, open)) != "" {
					loc.Code++
				} else {
					loc.Comments++
				}
				rest := line[strings.Index(line, open)+len(open):]
				close := afterBlockStart(open, l.BlockComments)
				if !strings.Contains(rest, close) {
					inBlock = true
					blockClose = close
				}
			} else if hasLineComment(line, l.LineComments) {
				loc.Comments++
			} else {
				loc.Code++
			}
		}
	}
	return loc
}

func beforeToken(line, token string) string {
	idx := strings.Index(line, token)
	if idx < 0 {
		return line
	}
	return line[:idx]
}

// firstBlockStart returns the earliest block-comment opener present, or "".
func firstBlockStart(line string, blocks [][2]string) string {
	best := -1
	bestOpen := ""
	for _, b := range blocks {
		if idx := strings.Index(line, b[0]); idx >= 0 && (best < 0 || idx < best) {
			best = idx
			bestOpen = b[0]
		}
	}
	return bestOpen
}

func afterBlockStart(open string, blocks [][2]string) string {
	for _, b := range blocks {
		if b[0] == open {
			return b[1]
		}
	}
	return ""
}

func hasLineComment(line string, markers []string) bool {
	for _, m := range markers {
		if strings.Contains(line, m) {
			return true
		}
	}
	return false
}

// Detect names the extension of a source file when it is one of the tracked
// languages (used by metrics to know which files to analyze).
func (l *Lang) CanonicalName() string { return l.Name }
