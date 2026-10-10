// Package risk combines code metrics with Git history to point at the code
// most likely to cause trouble: hotspots that are both complex and often
// changed, and knowledge held by people who are no longer around.
package risk

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Hotspot is a file ranked by how complex it is and how often it changes.
type Hotspot struct {
	Path         string     `json:"path"`
	Language     string     `json:"language"`
	Complexity   int        `json:"complexity"`
	LinesCode    int        `json:"linesCode"`
	Revisions    int        `json:"revisions"`
	Churn        int        `json:"churn"`
	Authors      int        `json:"authors"`
	LastCommitAt *time.Time `json:"lastCommitAt"`
	// Score is revisions times complexity, relative to the top hotspot (0..1).
	Score float64 `json:"score"`
}

// HotspotReport is the ranked list plus the window it covers.
type HotspotReport struct {
	Hotspots []Hotspot `json:"hotspots"`
	// Since is the start of the window, nil when it covers all history.
	Since *time.Time `json:"since"`
	// Until is the latest commit; windows count back from it, not from today,
	// so a dormant repository still shows where its work went.
	Until *time.Time `json:"until"`
	// Files is how many analyzed files changed within the window.
	Files int `json:"files"`
}

// Hotspots ranks files changed within the last months of history (0 for all
// history) by revisions times complexity.
func Hotspots(db *gorm.DB, repoID uint, months, limit int) (HotspotReport, error) {
	rep := HotspotReport{Hotspots: []Hotspot{}}
	latest, err := latestCommit(db, repoID)
	if err != nil || latest == nil {
		return rep, err
	}
	rep.Until = latest
	since := time.Time{}
	if months > 0 {
		since = latest.AddDate(0, -months, 0).Truncate(time.Second)
		rep.Since = &since
	}

	const from = `FROM commit_files cf
		JOIN commits c ON c.id = cf.commit_id
		JOIN files f ON f.repo_id = cf.repo_id AND f.path = cf.path
		WHERE cf.repo_id = ? AND c.date >= ? AND f.complexity > 0`
	var count int64
	if err := db.Raw(`SELECT COUNT(DISTINCT cf.path) `+from, repoID, since).Scan(&count).Error; err != nil {
		return rep, fmt.Errorf("count hotspots: %w", err)
	}
	rep.Files = int(count)

	var rows []struct {
		Path       string
		Language   string
		Complexity int
		LinesCode  int
		Revisions  int
		Churn      int
		Authors    int
	}
	err = db.Raw(`SELECT cf.path, f.language, f.complexity, f.lines_code,
			COUNT(*) AS revisions,
			SUM(cf.additions + cf.deletions) AS churn,
			COUNT(DISTINCT CASE WHEN c.email = '' THEN c.author ELSE c.email END) AS authors
		`+from+`
		GROUP BY cf.path
		ORDER BY COUNT(*) * f.complexity DESC, cf.path ASC
		LIMIT ?`, repoID, since, limit).Scan(&rows).Error
	if err != nil {
		return rep, fmt.Errorf("hotspots: %w", err)
	}
	if len(rows) == 0 {
		return rep, nil
	}

	paths := make([]string, len(rows))
	for i, r := range rows {
		paths[i] = r.Path
	}
	var files []models.File
	if err := db.Select("path", "last_commit_at").Where("repo_id = ? AND path IN ?", repoID, paths).
		Find(&files).Error; err != nil {
		return rep, fmt.Errorf("hotspot dates: %w", err)
	}
	last := make(map[string]*time.Time, len(files))
	for _, f := range files {
		last[f.Path] = f.LastCommitAt
	}

	top := float64(rows[0].Revisions * rows[0].Complexity)
	for _, r := range rows {
		rep.Hotspots = append(rep.Hotspots, Hotspot{
			Path: r.Path, Language: r.Language, Complexity: r.Complexity, LinesCode: r.LinesCode,
			Revisions: r.Revisions, Churn: r.Churn, Authors: r.Authors, LastCommitAt: last[r.Path],
			Score: float64(r.Revisions*r.Complexity) / top,
		})
	}
	return rep, nil
}

// latestCommit returns the date of the newest commit, or nil without history.
func latestCommit(db *gorm.DB, repoID uint) (*time.Time, error) {
	var c models.Commit
	err := db.Select("date").Where("repo_id = ?", repoID).Order("date DESC").First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest commit: %w", err)
	}
	d := c.Date.UTC()
	return &d, nil
}
