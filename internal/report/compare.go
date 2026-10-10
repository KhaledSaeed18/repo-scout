package report

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Comparison is what the scanned code changes against a base ref.
type Comparison struct {
	Base        string  `json:"base"`
	BaseCommit  string  `json:"baseCommit"`
	BaseSummary Summary `json:"baseSummary"`
	// AddedFiles and RemovedFiles list paths, capped at the report's top.
	AddedFiles   []string `json:"addedFiles"`
	RemovedFiles []string `json:"removedFiles"`
	FilesAdded   int      `json:"filesAdded"`
	FilesRemoved int      `json:"filesRemoved"`
	// ComplexityChanges are the files whose complexity moved most.
	ComplexityChanges []FileDelta `json:"complexityChanges"`
	NewCycles         []Cycle     `json:"newCycles"`
	ResolvedCycles    [][]string  `json:"resolvedCycles"`
	Dependencies      []DepChange `json:"dependencies"`

	baseCycles map[string]bool
}

// FileDelta is a file's complexity in the base and now. A file new since the
// base has Before 0.
type FileDelta struct {
	Path   string `json:"path"`
	Before int    `json:"before"`
	After  int    `json:"after"`
}

// DepChange is a dependency added, removed or moved to another version.
type DepChange struct {
	Manager string `json:"manager"`
	Name    string `json:"name"`
	Before  string `json:"before"`
	After   string `json:"after"`
	Change  string `json:"change"` // added, removed or changed
}

// compare builds the comparison of the scanned repository against base.
func compare(db *gorm.DB, headID uint, base Base, head Summary, top int) (*Comparison, error) {
	var baseRepo models.Repository
	if err := db.First(&baseRepo, base.RepoID).Error; err != nil {
		return nil, fmt.Errorf("load base: %w", err)
	}
	baseSum, err := summarize(db, &baseRepo)
	if err != nil {
		return nil, err
	}
	// The base is an exported tree with no history of its own.
	baseSum.Commits, baseSum.Contributors = head.Commits, head.Contributors
	c := &Comparison{
		Base: base.Ref, BaseCommit: base.Commit, BaseSummary: baseSum,
		AddedFiles: []string{}, RemovedFiles: []string{}, ComplexityChanges: []FileDelta{},
		NewCycles: []Cycle{}, ResolvedCycles: [][]string{}, Dependencies: []DepChange{},
	}
	if err := c.files(db, headID, base.RepoID, top); err != nil {
		return nil, err
	}
	if err := c.cycles(db, headID, base.RepoID); err != nil {
		return nil, err
	}
	if err := c.dependencies(db, headID, base.RepoID); err != nil {
		return nil, err
	}
	return c, nil
}

// files finds added and removed files and the largest complexity moves, in
// SQL so large trees stay out of memory.
func (c *Comparison) files(db *gorm.DB, headID, baseID uint, top int) error {
	const onlyIn = `SELECT a.path FROM files a WHERE a.repo_id = ? AND NOT EXISTS (
		SELECT 1 FROM files b WHERE b.repo_id = ? AND b.path = a.path) ORDER BY a.path`
	var added, removed []string
	if err := db.Raw(onlyIn, headID, baseID).Scan(&added).Error; err != nil {
		return fmt.Errorf("added files: %w", err)
	}
	if err := db.Raw(onlyIn, baseID, headID).Scan(&removed).Error; err != nil {
		return fmt.Errorf("removed files: %w", err)
	}
	c.FilesAdded, c.FilesRemoved = len(added), len(removed)
	c.AddedFiles = added[:min(len(added), top)]
	c.RemovedFiles = removed[:min(len(removed), top)]

	if err := db.Raw(`SELECT h.path, COALESCE(b.complexity, 0) AS before, h.complexity AS after
		FROM files h LEFT JOIN files b ON b.repo_id = ? AND b.path = h.path
		WHERE h.repo_id = ? AND h.complexity != COALESCE(b.complexity, 0)
		ORDER BY ABS(h.complexity - COALESCE(b.complexity, 0)) DESC, h.path
		LIMIT ?`, baseID, headID, top).Scan(&c.ComplexityChanges).Error; err != nil {
		return fmt.Errorf("complexity changes: %w", err)
	}
	return nil
}

// cycleKey identifies a cycle by its folders, whatever folder it starts at.
func cycleKey(folders []string) string {
	sorted := slices.Clone(folders)
	sort.Strings(sorted)
	return strings.Join(sorted, "\x00")
}

func (c *Comparison) cycles(db *gorm.DB, headID, baseID uint) error {
	head, err := cycles(db, headID)
	if err != nil {
		return err
	}
	base, err := cycles(db, baseID)
	if err != nil {
		return err
	}
	c.baseCycles = map[string]bool{}
	for _, b := range base {
		c.baseCycles[cycleKey(b.Folders)] = true
	}
	headKeys := map[string]bool{}
	for _, h := range head {
		headKeys[cycleKey(h.Folders)] = true
		if !c.baseCycles[cycleKey(h.Folders)] {
			h.New = true
			c.NewCycles = append(c.NewCycles, h)
		}
	}
	for _, b := range base {
		if !headKeys[cycleKey(b.Folders)] {
			c.ResolvedCycles = append(c.ResolvedCycles, b.Folders)
		}
	}
	return nil
}

// markNewCycles flags the report's cycles that the base did not have.
func markNewCycles(cs []Cycle, base map[string]bool) {
	for i := range cs {
		cs[i].New = !base[cycleKey(cs[i].Folders)]
	}
}

func (c *Comparison) dependencies(db *gorm.DB, headID, baseID uint) error {
	load := func(id uint) (map[string]models.Dependency, error) {
		var rows []models.Dependency
		if err := db.Where("repo_id = ?", id).Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("load dependencies: %w", err)
		}
		out := make(map[string]models.Dependency, len(rows))
		for _, d := range rows {
			out[d.Manager+"\x00"+d.Name] = d
		}
		return out, nil
	}
	head, err := load(headID)
	if err != nil {
		return err
	}
	base, err := load(baseID)
	if err != nil {
		return err
	}
	for k, h := range head {
		b, ok := base[k]
		switch {
		case !ok:
			c.Dependencies = append(c.Dependencies, DepChange{Manager: h.Manager, Name: h.Name, After: h.Version, Change: "added"})
		case b.Version != h.Version:
			c.Dependencies = append(c.Dependencies, DepChange{Manager: h.Manager, Name: h.Name, Before: b.Version, After: h.Version, Change: "changed"})
		}
	}
	for k, b := range base {
		if _, ok := head[k]; !ok {
			c.Dependencies = append(c.Dependencies, DepChange{Manager: b.Manager, Name: b.Name, Before: b.Version, Change: "removed"})
		}
	}
	sort.Slice(c.Dependencies, func(i, j int) bool {
		a, b := c.Dependencies[i], c.Dependencies[j]
		if a.Change != b.Change {
			return a.Change < b.Change
		}
		if a.Manager != b.Manager {
			return a.Manager < b.Manager
		}
		return a.Name < b.Name
	})
	return nil
}
