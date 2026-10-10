// Package report summarizes a scanned repository for the command line and
// CI: a readable text summary, JSON for tooling, SARIF for code-scanning
// dashboards, and quality gates that fail a build.
package report

import (
	"fmt"
	"path"
	"sort"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/architecture"
	"github.com/KhaledSaeed18/repo-scout/internal/coupling"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/risk"
)

// Gates are the conditions that fail a run. Zero values turn a gate off.
type Gates struct {
	// MaxComplexity fails the run when any file's total complexity exceeds it.
	MaxComplexity int `json:"maxComplexity,omitempty"`
	// MaxDuplicates fails the run when there are more duplicate groups.
	MaxDuplicates int `json:"maxDuplicates,omitempty"`
	// NoCycles fails the run on any circular dependency between folders.
	NoCycles bool `json:"noCycles,omitempty"`
	// NoHiddenCoupling fails the run on any coupled pair nothing explains.
	NoHiddenCoupling bool `json:"noHiddenCoupling,omitempty"`
	// MaxComplexityIncrease fails the run when total complexity grew by more
	// than this against the base. Needs a base.
	MaxComplexityIncrease int `json:"maxComplexityIncrease,omitempty"`
	// NoNewCycles fails the run on circular dependencies the base did not
	// have. Needs a base.
	NoNewCycles bool `json:"noNewCycles,omitempty"`
}

// Base is the scanned tree of the ref the code is compared against.
type Base struct {
	RepoID uint
	// Ref is the ref as given, Commit what it resolved to.
	Ref    string
	Commit string
}

// Options shape a report.
type Options struct {
	// Version is the Repo Scout version that produced the report.
	Version string
	// Top caps each list (hotspots, duplicates, hidden coupling).
	Top   int
	Gates Gates
	// Base, when set, adds a comparison against it.
	Base *Base
}

// Summary holds the repository's headline numbers.
type Summary struct {
	Files           int `json:"files"`
	LinesOfCode     int `json:"linesOfCode"`
	Complexity      int `json:"complexity"`
	Functions       int `json:"functions"`
	Commits         int `json:"commits"`
	Contributors    int `json:"contributors"`
	Dependencies    int `json:"dependencies"`
	DuplicateGroups int `json:"duplicateGroups"`
}

// Location is a span of lines in a file. Lines are zero for a whole file.
type Location struct {
	Path      string `json:"path"`
	StartLine int    `json:"startLine,omitempty"`
	EndLine   int    `json:"endLine,omitempty"`
}

// Cycle is a circular dependency between folders, located at the import
// that starts it when one is known. New marks cycles the base did not have.
type Cycle struct {
	Folders []string  `json:"folders"`
	At      *Location `json:"at,omitempty"`
	New     bool      `json:"new,omitempty"`
}

// Duplicate is a block of code repeated across files.
type Duplicate struct {
	Lines      int        `json:"lines"`
	Similarity float64    `json:"similarity"`
	Locations  []Location `json:"locations"`
}

// ComplexFile is a file over the complexity gate.
type ComplexFile struct {
	Path       string `json:"path"`
	Complexity int    `json:"complexity"`
}

// GateResult is the outcome of one enabled gate.
type GateResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// Report is everything one run found.
type Report struct {
	Tool           string          `json:"tool"`
	Version        string          `json:"version"`
	Repository     string          `json:"repository"`
	HeadCommit     string          `json:"headCommit"`
	GeneratedAt    time.Time       `json:"generatedAt"`
	Summary        Summary         `json:"summary"`
	Hotspots       []risk.Hotspot  `json:"hotspots"`
	Cycles         []Cycle         `json:"cycles"`
	HiddenCoupling []coupling.Pair `json:"hiddenCoupling"`
	Duplicates     []Duplicate     `json:"duplicates"`
	ComplexFiles   []ComplexFile   `json:"complexFiles"`
	Comparison     *Comparison     `json:"comparison,omitempty"`
	Gates          []GateResult    `json:"gates"`
}

// Passed reports whether every enabled gate passed.
func (r Report) Passed() bool {
	for _, g := range r.Gates {
		if !g.Passed {
			return false
		}
	}
	return true
}

// Build assembles the report for a scanned repository.
func Build(db *gorm.DB, repoID uint, opts Options) (Report, error) {
	top := max(opts.Top, 1)
	var repo models.Repository
	if err := db.First(&repo, repoID).Error; err != nil {
		return Report{}, fmt.Errorf("load repository: %w", err)
	}
	sum, err := summarize(db, &repo)
	if err != nil {
		return Report{}, err
	}
	rep := Report{
		Tool: "repo-scout", Version: opts.Version, Repository: repo.Path, HeadCommit: repo.HeadCommit,
		GeneratedAt: time.Now().UTC(), Summary: sum,
		Cycles: []Cycle{}, Duplicates: []Duplicate{}, ComplexFiles: []ComplexFile{}, Gates: []GateResult{},
	}

	hot, err := risk.Hotspots(db, repoID, 12, top)
	if err != nil {
		return Report{}, err
	}
	rep.Hotspots = hot.Hotspots

	if rep.Cycles, err = cycles(db, repoID); err != nil {
		return Report{}, err
	}
	if rep.HiddenCoupling, err = coupling.List(db, repoID, coupling.Query{Hidden: true, Limit: top}); err != nil {
		return Report{}, err
	}
	if rep.Duplicates, err = duplicates(db, repoID, top); err != nil {
		return Report{}, err
	}
	if opts.Gates.MaxComplexity > 0 {
		if err := db.Model(&models.File{}).Select("path", "complexity").
			Where("repo_id = ? AND complexity > ?", repoID, opts.Gates.MaxComplexity).
			Order("complexity DESC, path").Scan(&rep.ComplexFiles).Error; err != nil {
			return Report{}, fmt.Errorf("complex files: %w", err)
		}
	}
	if opts.Base != nil {
		if rep.Comparison, err = compare(db, repoID, *opts.Base, rep.Summary, top); err != nil {
			return Report{}, err
		}
		markNewCycles(rep.Cycles, rep.Comparison.baseCycles)
	}
	rep.Gates = evaluate(rep, opts.Gates)
	return rep, nil
}

// summarize reads a repository's headline numbers.
func summarize(db *gorm.DB, repo *models.Repository) (Summary, error) {
	sum := Summary{
		Files: repo.FileCount, LinesOfCode: repo.TotalCode, Commits: repo.CommitCount,
		Contributors: repo.ContributorCount, Dependencies: repo.DependencyCount, DuplicateGroups: repo.DupGroupCount,
	}
	var totals struct {
		Complexity int
		Functions  int
	}
	if err := db.Model(&models.File{}).Where("repo_id = ?", repo.ID).
		Select("COALESCE(SUM(complexity), 0) AS complexity, COALESCE(SUM(func_count), 0) AS functions").
		Scan(&totals).Error; err != nil {
		return Summary{}, fmt.Errorf("totals: %w", err)
	}
	sum.Complexity, sum.Functions = totals.Complexity, totals.Functions
	return sum, nil
}

// evaluate checks each enabled gate against the report.
func evaluate(r Report, g Gates) []GateResult {
	out := []GateResult{}
	if g.MaxComplexity > 0 {
		out = append(out, GateResult{
			Name:   "max-complexity",
			Passed: len(r.ComplexFiles) == 0,
			Detail: fmt.Sprintf("%d files above complexity %d", len(r.ComplexFiles), g.MaxComplexity),
		})
	}
	if g.MaxDuplicates > 0 {
		out = append(out, GateResult{
			Name:   "max-duplicates",
			Passed: r.Summary.DuplicateGroups <= g.MaxDuplicates,
			Detail: fmt.Sprintf("%d duplicate groups, limit %d", r.Summary.DuplicateGroups, g.MaxDuplicates),
		})
	}
	if g.NoCycles {
		out = append(out, GateResult{
			Name:   "no-cycles",
			Passed: len(r.Cycles) == 0,
			Detail: fmt.Sprintf("%d circular dependencies", len(r.Cycles)),
		})
	}
	if c := r.Comparison; c != nil && g.MaxComplexityIncrease > 0 {
		grew := r.Summary.Complexity - c.BaseSummary.Complexity
		out = append(out, GateResult{
			Name:   "max-complexity-increase",
			Passed: grew <= g.MaxComplexityIncrease,
			Detail: fmt.Sprintf("complexity %+d against %s, limit %+d", grew, c.Base, g.MaxComplexityIncrease),
		})
	}
	if c := r.Comparison; c != nil && g.NoNewCycles {
		out = append(out, GateResult{
			Name:   "no-new-cycles",
			Passed: len(c.NewCycles) == 0,
			Detail: fmt.Sprintf("%d circular dependencies not in %s", len(c.NewCycles), c.Base),
		})
	}
	if g.NoHiddenCoupling {
		out = append(out, GateResult{
			Name:   "no-hidden-coupling",
			Passed: len(r.HiddenCoupling) == 0,
			Detail: fmt.Sprintf("%d hidden dependencies", len(r.HiddenCoupling)),
		})
	}
	return out
}

// cycles finds circular folder dependencies and the import that opens each.
func cycles(db *gorm.DB, repoID uint) ([]Cycle, error) {
	var rows []models.ImportEdge
	if err := db.Where("repo_id = ?", repoID).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("load imports: %w", err)
	}
	edges := make([]architecture.Edge, 0, len(rows))
	for _, e := range rows {
		edges = append(edges, architecture.Edge{From: e.FromFile, To: e.ToFile, Kind: e.ImportType, Resolved: e.Resolved})
	}
	var files []models.File
	if err := db.Select("path", "language").Where("repo_id = ?", repoID).Find(&files).Error; err != nil {
		return nil, fmt.Errorf("load files: %w", err)
	}
	out := []Cycle{}
	for _, folders := range architecture.ReportFromGraph(edges, files).Cycles {
		c := Cycle{Folders: folders}
		if len(folders) > 1 {
			c.At = openingImport(edges, folders[0], folders[1])
		}
		out = append(out, c)
	}
	return out, nil
}

// openingImport finds a file in folder from that imports folder to, or a
// file inside it.
func openingImport(edges []architecture.Edge, from, to string) *Location {
	var hits []string
	for _, e := range edges {
		if !e.Resolved || path.Dir(e.From) != from {
			continue
		}
		if e.To == to || path.Dir(e.To) == to {
			hits = append(hits, e.From)
		}
	}
	if len(hits) == 0 {
		return nil
	}
	sort.Strings(hits)
	return &Location{Path: hits[0]}
}

// duplicates returns the largest duplicate groups with their locations.
func duplicates(db *gorm.DB, repoID uint, top int) ([]Duplicate, error) {
	var groups []models.DuplicateGroup
	if err := db.Where("repo_id = ?", repoID).Order("lines DESC, similarity DESC, id").Limit(top).
		Find(&groups).Error; err != nil {
		return nil, fmt.Errorf("load duplicates: %w", err)
	}
	if len(groups) == 0 {
		return []Duplicate{}, nil
	}
	ids := make([]uint, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}
	var blocks []models.DuplicateBlock
	if err := db.Where("group_id IN ?", ids).Order("file_path, start_line").Find(&blocks).Error; err != nil {
		return nil, fmt.Errorf("load duplicate blocks: %w", err)
	}
	byGroup := map[uint][]Location{}
	for _, b := range blocks {
		byGroup[b.GroupID] = append(byGroup[b.GroupID], Location{Path: b.FilePath, StartLine: b.StartLine, EndLine: b.EndLine})
	}
	out := make([]Duplicate, 0, len(groups))
	for _, g := range groups {
		out = append(out, Duplicate{Lines: g.Lines, Similarity: g.Similarity, Locations: byGroup[g.ID]})
	}
	return out, nil
}
