// Package metrics measures per-file code quality signals: cyclomatic
// complexity, import/export counts, function length, and nesting depth. The
// analyzers are language-aware regular-expression heuristics; they are fast
// enough to run over every file in a large repository.
package metrics

import (
	"regexp"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Result is the per-file metric payload.
type Result struct {
	Complexity int     `json:"complexity"`
	Imports    int     `json:"imports"`
	Exports    int     `json:"exports"`
	FuncCount  int     `json:"funcCount"`
	AvgFuncLen float64 `json:"avgFuncLen"`
	MaxFuncLen int     `json:"maxFuncLen"`
	AvgNesting float64 `json:"avgNesting"`
	MaxNesting int     `json:"maxNesting"`
}

// Apply writes the result onto a file record.
func (r Result) Apply(f *models.File) {
	f.Complexity = r.Complexity
	f.Imports = r.Imports
	f.Exports = r.Exports
	f.FuncCount = r.FuncCount
	f.AvgFuncLen = r.AvgFuncLen
	f.MaxFuncLen = r.MaxFuncLen
	f.AvgNesting = r.AvgNesting
	f.MaxNesting = r.MaxNesting
}

// langConfig holds the regexes used to analyze one language.
type langConfig struct {
	importRe  *regexp.Regexp
	importFn  func(src string) int
	exportRe  *regexp.Regexp
	funcStart []*regexp.Regexp
	funcSkip  []string
	decisions []*regexp.Regexp
	lineRe    *regexp.Regexp // line comments to strip
	blockRe   *regexp.Regexp // block comments to strip
	py        bool           // python-style indentation
}

var registry = map[string]*langConfig{}

func add(lang string, c *langConfig) { registry[lang] = c }

func mustCompile(s string) *regexp.Regexp { return regexp.MustCompile(s) }

// Analyze computes metrics for a file. Unknown or non-source languages yield
// an empty result.
func Analyze(f models.File, content string) Result {
	lc := registry[f.Language]
	if lc == nil {
		return Result{}
	}
	src := sanitize(lc, content)
	r := Result{}
	if lc.importFn != nil {
		r.Imports = lc.importFn(src)
	} else {
		r.Imports = count(lc.importRe, src)
	}
	r.Exports = count(lc.exportRe, src)
	r.Complexity = complexity(lc, src)
	r.FuncCount, r.AvgFuncLen, r.MaxFuncLen = functions(lc, src)
	r.AvgNesting, r.MaxNesting = nesting(lc, src)
	return r
}

func count(re *regexp.Regexp, src string) int {
	if re == nil {
		return 0
	}
	return len(re.FindAllString(src, -1))
}

// countGoImports counts imported packages, including each package inside a
// grouped import block.
func countGoImports(src string) int {
	n := 0
	inBlock := false
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case inBlock:
			if t == ")" {
				inBlock = false
				continue
			}
			if t != "" {
				n++
			}
		case strings.HasPrefix(t, "import ("):
			inBlock = true
		case strings.HasPrefix(t, "import "):
			n++
		}
	}
	return n
}

func complexity(lc *langConfig, src string) int {
	total := 1
	for _, re := range lc.decisions {
		total += count(re, src)
	}
	return total
}

func sanitize(lc *langConfig, content string) string {
	src := content
	if lc.blockRe != nil {
		src = lc.blockRe.ReplaceAllString(src, " ")
	}
	if lc.lineRe != nil {
		src = lc.lineRe.ReplaceAllString(src, "\n")
	}
	return src
}

// nesting computes the average and maximum indentation depth of code lines.
func nesting(lc *langConfig, src string) (avg float64, max int) {
	var total, n float64
	for _, line := range strings.Split(src, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		depth := indentDepth(line)
		total += depth
		n++
		if depth > float64(max) {
			max = int(depth)
		}
	}
	if n == 0 {
		return 0, 0
	}
	return total / n, max
}

func indentDepth(line string) float64 {
	depth := 0.0
	spaces := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\t':
			depth++
			spaces = 0
		case ' ':
			spaces++
			if spaces == 4 {
				depth++
				spaces = 0
			}
		default:
			return depth
		}
	}
	return depth
}

// functions finds function start lines, brace-matches their bodies, and
// reports count plus average and maximum length in lines. Python uses
// indentation-based bodies instead of braces.
func functions(lc *langConfig, src string) (count int, avg float64, maxLen int) {
	lines := strings.Split(src, "\n")
	if lc.py {
		return pythonFunctions(lines)
	}
	var total int
	for _, start := range lc.funcStart {
		for _, m := range start.FindAllStringIndex(src, -1) {
			lineNo := strings.Count(src[:m[0]], "\n")
			if lineNo >= len(lines) {
				continue
			}
			if skippedControl(lc, lines[lineNo]) {
				continue
			}
			end := braceClose(lines, lineNo)
			if end < 0 {
				continue
			}
			length := end - lineNo + 1
			count++
			total += length
			if length > maxLen {
				maxLen = length
			}
		}
	}
	if count > 0 {
		avg = float64(total) / float64(count)
	}
	return count, avg, maxLen
}

func pythonFunctions(lines []string) (count int, avg float64, maxLen int) {
	var total int
	i := 0
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "def ") || strings.HasPrefix(t, "class ") {
			base := indentDepth(lines[i])
			count++
			end := i + 1
			for end < len(lines) {
				line := lines[end]
				if strings.TrimSpace(line) != "" && indentDepth(line) <= base {
					break
				}
				end++
			}
			length := end - i
			total += length
			if length > maxLen {
				maxLen = length
			}
			i = end
			continue
		}
		i++
	}
	if count > 0 {
		avg = float64(total) / float64(count)
	}
	return count, avg, maxLen
}

func skippedControl(lc *langConfig, line string) bool {
	t := strings.TrimSpace(line)
	for _, kw := range lc.funcSkip {
		if strings.HasPrefix(t, kw) {
			rest := strings.TrimSpace(strings.TrimPrefix(t, kw))
			if rest == "" || strings.HasPrefix(rest, "(") || strings.HasPrefix(rest, " ") {
				return true
			}
		}
	}
	return false
}

// braceClose returns the index of the line whose brace closes the block opened
// on line start, or -1 when unbalanced.
func braceClose(lines []string, start int) int {
	depth := 0
	for i := start; i < len(lines); i++ {
		for _, ch := range lines[i] {
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					return i
				}
			}
		}
	}
	return -1
}

func init() {
	add("Go", &langConfig{
		importFn:  countGoImports,
		exportRe:  mustCompile(`(?m)^\s*(?:func|var|const|type)\s+[A-Z]`),
		funcStart: []*regexp.Regexp{mustCompile(`(?m)^\s*func\s`)},
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
	add("Rust", &langConfig{
		importRe:  mustCompile(`\buse\b`),
		exportRe:  mustCompile(`(?m)^\s*pub\s`),
		funcStart: []*regexp.Regexp{mustCompile(`(?m)^\s*(?:(?:pub|async|unsafe|extern)\s+)*fn\s`)},
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
	add("Python", &langConfig{
		importRe:  mustCompile(`(?m)^\s*(?:import|from\s+\S+\s+import)\s`),
		exportRe:  mustCompile(`(?m)^(?:def|class)\s`),
		funcStart: []*regexp.Regexp{mustCompile(`(?m)^\s*(?:async\s+)?(?:def|class)\s`)},
		decisions: []*regexp.Regexp{
			mustCompile(`\b(?:if|elif|for|while|except|with|match|case)\b`),
			mustCompile(`\b(?:and|or)\b`),
		},
		lineRe: mustCompile(`#.*`),
		py:     true,
	})
	add("TypeScript", jsConfig())
	add("JavaScript", jsConfig())
	add("Java", &langConfig{
		importRe:  mustCompile(`(?m)^\s*import\s`),
		exportRe:  mustCompile(`(?m)^\s*public\s`),
		funcStart: []*regexp.Regexp{cFamilyFunc()},
		funcSkip:  cFamilySkip,
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
	add("Kotlin", &langConfig{
		importRe:  mustCompile(`(?m)^\s*import\s`),
		exportRe:  mustCompile(`(?m)^\s*(?:public|open)\s|^\s*fun\s|^\s*(?:class|object|interface)\s`),
		funcStart: []*regexp.Regexp{mustCompile(`(?m)^\s*(?:public|private|protected|internal|override|suspend|operator|infix|tailrec|external|inline)\s+fun\s|^\s*fun\s`)},
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
	add("PHP", &langConfig{
		importRe:  mustCompile(`\buse\s+[A-Z\\]`),
		exportRe:  mustCompile(`(?m)^\s*(?:public|protected)\s+function\s|^\s*(?:abstract\s+)?class\s`),
		funcStart: []*regexp.Regexp{mustCompile(`(?m)^\s*(?:public|private|protected|static|abstract|final|async)?\s*(?:function\s+)?(?:function\s)`)},
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`(?://.*|#.*)`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
	add("C#", &langConfig{
		importRe:  mustCompile(`(?m)^\s*using\s`),
		exportRe:  mustCompile(`(?m)^\s*public\s`),
		funcStart: []*regexp.Regexp{cFamilyFunc()},
		funcSkip:  cFamilySkip,
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
	add("C++", cLikeConfig())
	add("C", cLikeConfig())
	add("Swift", &langConfig{
		importRe:  mustCompile(`(?m)^\s*import\s`),
		exportRe:  mustCompile(`\bpublic\b`),
		funcStart: []*regexp.Regexp{mustCompile(`(?m)^\s*(?:public|private|internal|fileprivate|open|static|class|struct|enum)\s+(?:func\s+)?func\s|^\s*func\s`)},
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	})
}

func cFamilyDecisions() []*regexp.Regexp {
	return []*regexp.Regexp{
		mustCompile(`\b(?:if|for|while|switch|catch|case|do|else\s+if)\b`),
		mustCompile(`\&\&|\|\||\?`),
	}
}

func cLikeConfig() *langConfig {
	return &langConfig{
		importRe:  mustCompile(`#include`),
		exportRe:  nil,
		funcStart: []*regexp.Regexp{cFamilyFunc()},
		funcSkip:  cFamilySkip,
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	}
}

// cFamilyFunc matches a line that looks like a function definition: it ends
// with ")" or ") {" but is not a control statement (filtered via funcSkip).
func cFamilyFunc() *regexp.Regexp {
	return mustCompile(`(?m)^\s*[\w<>&*:\s,\.]+\([^;]*\)\s*\{?\s*$`)
}

var cFamilySkip = []string{"if", "for", "while", "switch", "catch", "return", "throw", "case", "else", "do", "foreach"}

func jsConfig() *langConfig {
	return &langConfig{
		importRe: mustCompile(`\b(?:import|require\()\b`),
		exportRe: mustCompile(`\bexport\b`),
		funcStart: []*regexp.Regexp{
			mustCompile(`(?m)^\s*(?:export\s+)?(?:default\s+)?(?:async\s+)?function\s`),
			mustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?\([^)]*\)\s*=>\s*\{`),
			mustCompile(`(?m)^\s*(?:export\s+)?(?:async\s+)?[\w$]+\s*\([^)]*\)\s*\{`),
		},
		decisions: cFamilyDecisions(),
		lineRe:    mustCompile(`//.*`),
		blockRe:   mustCompile(`/\*.*?\*/`),
	}
}
