// Package architecture builds the repository's import graph, detects circular
// dependencies, and flags unused modules and dead files.
package architecture

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Edge is a dependency between two nodes (files or folders).
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	Resolved bool   `json:"resolved"`
}

// Report is the full architecture analysis result.
type Report struct {
	// Edges are the resolved file-level import edges.
	Edges []Edge `json:"edges"`
	// Cycles are strongly connected components of the folder graph.
	Cycles [][]string `json:"cycles"`
	// UnusedModules are folders with no incoming dependency.
	UnusedModules []string `json:"unusedModules"`
	// DeadFiles are files not reachable from any entry point.
	DeadFiles []string `json:"deadFiles"`
	// EntryPoints are the detected application entry points.
	EntryPoints []string `json:"entryPoints"`
	// Unresolved are import specs that did not map to a file in the repo.
	Unresolved []Edge `json:"unresolved"`
	// Folders lists all folders that participate in the module graph.
	Folders []string `json:"folders"`
}

type builder struct {
	repoID     uint
	fileSet    map[string]bool
	dirSet     map[string]bool
	module     string // go module name, when present
	read       func(rel string) (string, error)
	root       string
	edges      []Edge
	unresolved []Edge
}

// Build analyzes the import structure of a repository.
func Build(root string, files []models.File, read func(rel string) (string, error)) (Report, error) {
	b := &builder{
		fileSet: map[string]bool{},
		dirSet:  map[string]bool{},
		read:    read,
		root:    root,
	}
	for _, f := range files {
		b.fileSet[f.Path] = true
		b.dirSet[folderOf(f.Path)] = true
	}
	if b.read == nil {
		b.read = func(rel string) (string, error) { return "", nil }
	}
	if b.fileSet["go.mod"] {
		if content, err := b.read("go.mod"); err == nil {
			b.module = moduleFromGoMod(content)
		}
	}

	for _, f := range files {
		if _, ok := extractors[f.Language]; !ok {
			continue
		}
		content, err := b.read(f.Path)
		if err != nil {
			continue
		}
		for _, spec := range extract(f.Language, content) {
			for _, target := range b.resolve(f.Path, f.Language, spec) {
				if target.resolved {
					b.edges = append(b.edges, Edge{From: f.Path, To: target.path, Kind: "import", Resolved: true})
				} else if target.internal {
					b.unresolved = append(b.unresolved, Edge{From: f.Path, To: target.path, Kind: "import", Resolved: false})
				}
			}
		}
	}

	report := Report{
		Edges:      b.edges,
		Unresolved: b.unresolved,
		EntryPoints: []string{},
		DeadFiles:   []string{},
		Folders:     []string{},
		Cycles:      [][]string{},
		UnusedModules: []string{},
	}

	fileToFiles := map[string][]string{}
	for _, e := range report.Edges {
		fileToFiles[e.From] = append(fileToFiles[e.From], e.To)
	}

	report.EntryPoints = entryPoints(b.fileSet)
	report.DeadFiles = deadFiles(b.fileSet, b.dirSet, report.EntryPoints, fileToFiles)
	report.Folders = sortedKeys(b.dirSet)

	folderGraph := buildFolderGraph(report.Edges, b.dirSet)
	report.Cycles = tarjanCycles(folderGraph)
	report.UnusedModules = unusedModules(b.dirSet, folderGraph, report.EntryPoints)
	sort.Strings(report.DeadFiles)
	sort.Strings(report.UnusedModules)
	return report, nil
}

// ReportFromGraph recomputes a Report from stored edges and the repository
// file list, without re-reading file contents. It is used by the API layer to
// serve architecture data on demand.
func ReportFromGraph(edges []Edge, files []models.File) Report {
	fileSet := map[string]bool{}
	dirSet := map[string]bool{}
	for _, f := range files {
		fileSet[f.Path] = true
		dirSet[folderOf(f.Path)] = true
	}
	report := Report{
		Edges:        edges,
		EntryPoints:  []string{},
		DeadFiles:    []string{},
		Folders:      []string{},
		Cycles:       [][]string{},
		UnusedModules: []string{},
		Unresolved:   []Edge{},
	}
	fileToFiles := map[string][]string{}
	for _, e := range edges {
		fileToFiles[e.From] = append(fileToFiles[e.From], e.To)
	}
	report.EntryPoints = entryPoints(fileSet)
	report.DeadFiles = deadFiles(fileSet, dirSet, report.EntryPoints, fileToFiles)
	report.Folders = sortedKeys(dirSet)
	folderGraph := buildFolderGraph(edges, dirSet)
	report.Cycles = tarjanCycles(folderGraph)
	report.UnusedModules = unusedModules(dirSet, folderGraph, report.EntryPoints)
	sort.Strings(report.DeadFiles)
	sort.Strings(report.UnusedModules)
	return report
}

func moduleFromGoMod(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		if k != "" {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

func folderOf(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return ""
	}
	return path[:i]
}

type target struct {
	path     string
	resolved bool
	internal bool // internal-looking but not found
}

// resolve maps an import spec to repository nodes. The returned targets are
// either existing file paths (resolved) or intended internal paths (unresolved
// but likely internal).
func (b *builder) resolve(fromFile, lang, spec string) []target {
	switch lang {
	case "Go":
		return b.resolveGo(fromFile, spec)
	case "TypeScript", "JavaScript":
		return resolveJS(fromFile, spec, b)
	case "Python":
		return resolvePython(fromFile, spec, b)
	case "Rust":
		return resolveRust(fromFile, spec, b)
	case "Java", "Kotlin":
		return resolveJavaLike(spec, b)
	case "C#":
		return resolveJavaLike(spec, b)
	case "C", "C++":
		return resolveCPP(fromFile, spec, b)
	case "PHP":
		return resolvePHP(fromFile, spec, b)
	case "Swift":
		return []target{{path: spec, internal: false}}
	default:
		return nil
	}
}

func (b *builder) resolveGo(fromFile, spec string) []target {
	if b.module == "" {
		return nil
	}
	if !strings.HasPrefix(spec, b.module) {
		return nil // external dependency
	}
	rel := strings.TrimPrefix(spec, b.module)
	rel = strings.Trim(rel, "/")
	if rel == "" {
		return []target{{path: ".", resolved: true}}
	}
	dir := rel
	return []target{{path: dir, resolved: b.dirSet[dir], internal: true}}
}

func resolveJS(fromFile, spec string, b *builder) []target {
	if !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../") {
		return nil // external or alias
	}
	dir := folderOf(fromFile)
	base := filepath.Clean(filepath.Join(dir, spec))
	candidates := []string{
		base, base + ".ts", base + ".tsx", base + ".js", base + ".jsx",
		base + ".mjs", base + ".cjs",
		base + "/index.ts", base + "/index.tsx", base + "/index.js", base + "/index.jsx",
	}
	for _, c := range candidates {
		if b.fileSet[c] {
			return []target{{path: c, resolved: true}}
		}
	}
	return nil
}

func resolvePython(fromFile, spec string, b *builder) []target {
	spec = strings.TrimPrefix(spec, ".")
	parts := strings.Split(spec, ".")
	if len(parts) == 0 || parts[0] == "" {
		return nil
	}
	// Try a.b.c -> a/b/c.py or a/b/c/__init__.py, also under src/
	relCandidates := [][]string{
		{filepath.Join(parts...) + ".py"},
		{filepath.Join(parts...), "__init__.py"},
		{"src", filepath.Join(parts...) + ".py"},
		{"src", filepath.Join(parts...), "__init__.py"},
	}
	for _, cand := range relCandidates {
		p := filepath.ToSlash(filepath.Join(cand...))
		if b.fileSet[p] {
			return []target{{path: p, resolved: true}}
		}
	}
	return nil
}

func resolveRust(fromFile, spec string, b *builder) []target {
	spec = strings.TrimSpace(spec)
	// remove :: parts after the second for crate detection
	if strings.HasPrefix(spec, "crate::") {
		rel := strings.TrimPrefix(spec, "crate::")
		parts := strings.SplitN(rel, "::", 2)
		name := parts[0]
		candidates := []string{
			"src/" + name + ".rs",
			"src/" + name + "/mod.rs",
			name + ".rs",
			name + "/mod.rs",
		}
		for _, c := range candidates {
			if b.fileSet[c] {
				return []target{{path: c, resolved: true}}
			}
		}
		return nil
	}
	if strings.HasPrefix(spec, "super::") || strings.HasPrefix(spec, "self::") {
		dir := folderOf(fromFile)
		rel := spec[strings.Index(spec, "::")+2:]
		parts := strings.SplitN(rel, "::", 2)
		name := parts[0]
		candidates := []string{
			dir + "/" + name + ".rs",
			dir + "/" + name + "/mod.rs",
		}
		for _, c := range candidates {
			if b.fileSet[c] {
				return []target{{path: c, resolved: true}}
			}
		}
		return nil
	}
	return nil // external crate
}

func resolveJavaLike(spec string, b *builder) []target {
	parts := strings.Split(spec, ".")
	if len(parts) < 2 {
		return nil
	}
	dir := filepath.ToSlash(filepath.Join(parts[:len(parts)-1]...))
	file := parts[len(parts)-1]
	ext := ".java"
	if b.hasExt(file, ".java") || b.hasExt(file, ".kt") || b.hasExt(file, ".cs") {
		ext = guessExt(file, b)
	}
	for _, prefix := range []string{"", "src/main/java/", "src/main/kotlin/", "src/"} {
		p := prefix + dir + "/" + file + ext
		if b.fileSet[p] {
			return []target{{path: p, resolved: true}}
		}
	}
	return nil
}

func (b *builder) hasExt(name, ext string) bool { return strings.HasSuffix(name, ext) }

func guessExt(file string, b *builder) string {
	for _, e := range []string{".java", ".kt", ".cs"} {
		if b.fileSet[file+e] {
			return e
		}
	}
	return ".java"
}

func resolveCPP(fromFile, spec string, b *builder) []target {
	if strings.HasPrefix(spec, "/") {
		return nil
	}
	dir := folderOf(fromFile)
	candidates := []string{
		dir + "/" + spec,
		filepath.ToSlash(spec),
	}
	for _, c := range candidates {
		if b.fileSet[c] {
			return []target{{path: c, resolved: true}}
		}
	}
	return nil
}

func resolvePHP(fromFile, spec string, b *builder) []target {
	// use Vendor\Package\Class; -> Vendor/Package/Class.php
	rel := strings.ReplaceAll(spec, "\\", "/")
	rel = strings.TrimPrefix(rel, "/")
	candidates := []string{rel + ".php", folderOf(fromFile) + "/" + rel + ".php"}
	for _, c := range candidates {
		if b.fileSet[c] {
			return []target{{path: c, resolved: true}}
		}
	}
	return nil
}

func entryPoints(fileSet map[string]bool) []string {
	var out []string
	for f := range fileSet {
		base := f
		if i := strings.LastIndex(base, "/"); i >= 0 {
			base = base[i+1:]
		}
		switch {
		case base == "main.go", base == "main.rs", base == "main.py", base == "__main__.py",
			base == "index.ts", base == "index.tsx", base == "index.js", base == "index.jsx",
			base == "app.ts", base == "app.tsx", base == "app.js", base == "app.jsx",
			base == "server.ts", base == "server.js":
			out = append(out, f)
		}
	}
	sort.Strings(out)
	return out
}

// deadFiles returns files not reachable from entry points via import edges.
func deadFiles(fileSet, dirSet map[string]bool, entries []string, edges map[string][]string) []string {
	reached := map[string]bool{}
	queue := append([]string(nil), entries...)
	for _, e := range entries {
		reached[e] = true
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		// reaching a folder reaches all its files
		if dirSet[cur] {
			for f := range fileSet {
				if folderOf(f) == cur && !reached[f] {
					reached[f] = true
					queue = append(queue, f)
				}
			}
		}
		for _, next := range edges[cur] {
			if !reached[next] {
				reached[next] = true
				queue = append(queue, next)
			}
		}
	}
	var dead []string
	for f := range fileSet {
		if !reached[f] {
			dead = append(dead, f)
		}
	}
	return dead
}

// buildFolderGraph aggregates file-level edges into a folder-level graph.
// Targets that are directories become their own node.
func buildFolderGraph(edges []Edge, dirSet map[string]bool) map[string][]string {
	graph := map[string][]string{}
	add := func(from, to string) {
		graph[from] = append(graph[from], to)
	}
	for _, e := range edges {
		fromDir := folderOf(e.From)
		to := e.To
		if !dirSet[to] {
			to = folderOf(e.To)
		}
		if fromDir != to {
			add(fromDir, to)
		}
	}
	return graph
}

func unusedModules(dirSet map[string]bool, folderGraph map[string][]string, entries []string) []string {
	entryDirs := map[string]bool{}
	for _, e := range entries {
		entryDirs[folderOf(e)] = true
	}
	incoming := map[string]bool{}
	for _, tos := range folderGraph {
		for _, to := range tos {
			incoming[to] = true
		}
	}
	var out []string
	for dir := range dirSet {
		if dir == "" {
			continue
		}
		if entryDirs[dir] || incoming[dir] {
			continue
		}
		out = append(out, dir)
	}
	sort.Strings(out)
	return out
}

// tarjanCycles returns strongly connected components of size > 1, meaning
// genuine circular dependencies between folders.
func tarjanCycles(graph map[string][]string) [][]string {
	index := 0
	stack := []string{}
	indices := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var cycles [][]string

	var strongconnect func(v string)
	strongconnect = func(v string) {
		indices[v] = index
		low[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range graph[v] {
			if _, ok := indices[w]; !ok {
				strongconnect(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && indices[w] < low[v] {
				low[v] = indices[w]
			}
		}

		if low[v] == indices[v] {
			var comp []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp = append(comp, w)
				if w == v {
					break
				}
			}
			if len(comp) > 1 {
				sort.Strings(comp)
				cycles = append(cycles, comp)
			}
		}
	}

	nodes := map[string]bool{}
	for from := range graph {
		nodes[from] = true
		for _, to := range graph[from] {
			nodes[to] = true
		}
	}
	for n := range nodes {
		if _, ok := indices[n]; !ok {
			strongconnect(n)
		}
	}
	sort.Slice(cycles, func(i, j int) bool { return len(cycles[i]) < len(cycles[j]) })
	return cycles
}

var extractors = map[string]bool{
	"Go": true, "Rust": true, "Python": true, "Java": true, "Kotlin": true,
	"TypeScript": true, "JavaScript": true, "PHP": true, "C#": true,
	"C++": true, "C": true, "Swift": true,
}
