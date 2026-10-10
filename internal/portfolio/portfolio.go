// Package portfolio rolls every scanned repository up into one view: size,
// direction since the previous scan, and the risks the other views find one
// repository at a time.
package portfolio

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/coupling"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/report"
	"github.com/KhaledSaeed18/repo-scout/internal/risk"
)

// hiddenCap bounds how many hidden dependencies are counted per repository.
const hiddenCap = 500

// Entry is one repository's line in the portfolio.
type Entry struct {
	Repository models.Repository `json:"repository"`
	Complexity int               `json:"complexity"`
	// LinesChange and ComplexityChange compare the last two scans; nil when
	// there is only one.
	LinesChange      *int `json:"linesChange"`
	ComplexityChange *int `json:"complexityChange"`
	BusFactor        int  `json:"busFactor"`
	// InactiveShare is the share of code whose main author went inactive.
	InactiveShare float64 `json:"inactiveShare"`
	Cycles        int     `json:"cycles"`
	// HiddenDependencies counts coupled pairs nothing explains, up to 500.
	HiddenDependencies int    `json:"hiddenDependencies"`
	TopHotspot         string `json:"topHotspot"`
}

// Portfolio is every repository that has finished a scan.
type Portfolio struct {
	Entries []Entry `json:"entries"`
	// Unscanned counts repositories still waiting for a first scan.
	Unscanned int `json:"unscanned"`
}

// Build assembles the portfolio. Each repository costs a handful of
// aggregate queries, so it stays quick for the tens of repositories one
// machine holds.
func Build(db *gorm.DB) (Portfolio, error) {
	var repos []models.Repository
	if err := db.Order("name, id").Find(&repos).Error; err != nil {
		return Portfolio{}, fmt.Errorf("load repositories: %w", err)
	}
	out := Portfolio{Entries: []Entry{}}
	for _, repo := range repos {
		if repo.LastScannedAt == nil {
			out.Unscanned++
			continue
		}
		e, err := entry(db, repo)
		if err != nil {
			return Portfolio{}, fmt.Errorf("%s: %w", repo.Name, err)
		}
		out.Entries = append(out.Entries, e)
	}
	return out, nil
}

func entry(db *gorm.DB, repo models.Repository) (Entry, error) {
	e := Entry{Repository: repo}
	if err := db.Model(&models.File{}).Where("repo_id = ?", repo.ID).
		Select("COALESCE(SUM(complexity), 0)").Scan(&e.Complexity).Error; err != nil {
		return Entry{}, fmt.Errorf("complexity: %w", err)
	}

	var snaps []models.ScanSnapshot
	if err := db.Where("repo_id = ?", repo.ID).Order("scanned_at DESC, id DESC").Limit(2).Find(&snaps).Error; err != nil {
		return Entry{}, fmt.Errorf("snapshots: %w", err)
	}
	if len(snaps) == 2 {
		lines := snaps[0].TotalCode - snaps[1].TotalCode
		cx := snaps[0].Complexity - snaps[1].Complexity
		e.LinesChange, e.ComplexityChange = &lines, &cx
	}

	know, err := risk.Knowledge(db, repo.ID, 2, 6)
	if err != nil {
		return Entry{}, err
	}
	e.BusFactor = know.BusFactor
	if know.Lines > 0 {
		e.InactiveShare = float64(know.InactiveLines) / float64(know.Lines)
	}

	hot, err := risk.Hotspots(db, repo.ID, 12, 1)
	if err != nil {
		return Entry{}, err
	}
	if len(hot.Hotspots) > 0 {
		e.TopHotspot = hot.Hotspots[0].Path
	}

	cycles, err := report.Cycles(db, repo.ID)
	if err != nil {
		return Entry{}, err
	}
	e.Cycles = len(cycles)

	hidden, err := coupling.List(db, repo.ID, coupling.Query{Hidden: true, Limit: hiddenCap})
	if err != nil {
		return Entry{}, err
	}
	e.HiddenDependencies = len(hidden)
	return e, nil
}
