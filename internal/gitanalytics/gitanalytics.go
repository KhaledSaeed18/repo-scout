// Package gitanalytics serves aggregated views over stored git history:
// heatmaps, streaks, ownership, largest commits, and leaderboards.
package gitanalytics

import (
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Day is one calendar day of commit activity.
type Day struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
}

// Heatmap holds commit activity rolled up by day and by weekday/hour.
type Heatmap struct {
	Daily  []Day   `json:"daily"`
	Hourly [][]int `json:"hourly"` // [weekday][hour], weekday 0 = Sunday
	Total  int     `json:"total"`
	Start  string  `json:"start"`
	End    string  `json:"end"`
}

// Streak is a run of consecutive days with at least one commit.
type Streak struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Days  int    `json:"days"`
}

// StreaksResult is the streak analysis for one contributor.
type StreaksResult struct {
	Email        string   `json:"email"`
	Longest      Streak   `json:"longest"`
	Current      Streak   `json:"current"`
	All          []Streak `json:"all"`
	ActiveDays   int      `json:"activeDays"`
	TotalCommits int      `json:"totalCommits"`
}

// OwnerSummary aggregates file ownership per author.
type OwnerSummary struct {
	Author string  `json:"author"`
	Files  int     `json:"files"`
	Share  float64 `json:"share"`
}

// Ownership is the per-repository ownership rollup.
type Ownership struct {
	ByAuthor []OwnerSummary `json:"byAuthor"`
	Total    int            `json:"total"`
}

// LargestCommit is one heavy commit from history.
type LargestCommit struct {
	Hash         string `json:"hash"`
	Author       string `json:"author"`
	Email        string `json:"email"`
	Date         string `json:"date"`
	Message      string `json:"message"`
	FilesChanged int    `json:"filesChanged"`
	Insertions   int    `json:"insertions"`
	Deletions    int    `json:"deletions"`
}

// Heatmap rolls commit counts up by calendar day and by weekday/hour.
func ComputeHeatmap(db *gorm.DB, repoID uint) (Heatmap, error) {
	var commits []models.Commit
	err := db.Select("date").Where("repo_id = ?", repoID).Order("date ASC").Find(&commits).Error
	if err != nil {
		return Heatmap{}, err
	}
	h := Heatmap{Total: len(commits)}
	if len(commits) == 0 {
		return h, nil
	}
	h.Hourly = make([][]int, 7)
	for i := range h.Hourly {
		h.Hourly[i] = make([]int, 24)
	}
	h.Start = commits[0].Date.Format("2006-01-02")
	h.End = commits[len(commits)-1].Date.Format("2006-01-02")
	byDay := map[string]int{}
	for _, c := range commits {
		d := c.Date.Format("2006-01-02")
		byDay[d]++
		h.Hourly[int(c.Date.Weekday())][c.Date.Hour()]++
	}
	days := make([]Day, 0, len(byDay))
	for d, n := range byDay {
		days = append(days, Day{Date: d, Count: n})
	}
	sortDays(days)
	h.Daily = days
	return h, nil
}

// Streaks analyzes consecutive-day activity for one contributor.
func Streaks(db *gorm.DB, repoID uint, email string) (StreaksResult, error) {
	var commits []models.Commit
	err := db.Select("date").Where("repo_id = ? AND email = ?", repoID, email).
		Order("date ASC").Find(&commits).Error
	if err != nil {
		return StreaksResult{}, err
	}
	res := StreaksResult{Email: email}
	if len(commits) == 0 {
		return res, nil
	}
	seen := map[string]bool{}
	for _, c := range commits {
		seen[c.Date.Format("2006-01-02")] = true
		res.TotalCommits++
	}
	activeDays := make([]string, 0, len(seen))
	for d := range seen {
		activeDays = append(activeDays, d)
	}
	sortDaysString(activeDays)
	res.ActiveDays = len(activeDays)

	var current *Streak
	var longest Streak
	start := activeDays[0]
	prev, _ := time.Parse("2006-01-02", activeDays[0])
	for _, d := range activeDays[1:] {
		cur, _ := time.Parse("2006-01-02", d)
		if cur.Sub(prev) > 24*time.Hour {
			s := Streak{Start: start, End: prev.Format("2006-01-02"), Days: daysBetween(start, prev.Format("2006-01-02"))}
			res.All = append(res.All, s)
			if s.Days > longest.Days {
				longest = s
			}
			start = d
		}
		prev = cur
	}
	s := Streak{Start: start, End: prev.Format("2006-01-02"), Days: daysBetween(start, prev.Format("2006-01-02"))}
	res.All = append(res.All, s)
	if s.Days > longest.Days {
		longest = s
	}
	current = &s
	if longest.Days == 0 {
		longest = s
	}
	res.Longest = longest
	res.Current = *current
	return res, nil
}

// Ownership aggregates primary-authorship across files.
func ComputeOwnership(db *gorm.DB, repoID uint) (Ownership, error) {
	var files []models.File
	err := db.Select("author", "path").Where("repo_id = ? AND author != ''", repoID).Find(&files).Error
	if err != nil {
		return Ownership{}, err
	}
	o := Ownership{Total: len(files)}
	counts := map[string]int{}
	for _, f := range files {
		counts[f.Author]++
	}
	for a, n := range counts {
		o.ByAuthor = append(o.ByAuthor, OwnerSummary{Author: a, Files: n, Share: float64(n) / float64(len(files))})
	}
	sortOwners(o.ByAuthor)
	return o, nil
}

// LargestCommits lists the heaviest commits by total lines changed.
func LargestCommits(db *gorm.DB, repoID uint, limit int) ([]LargestCommit, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	var commits []models.Commit
	err := db.Where("repo_id = ?", repoID).
		Order("insertions + deletions DESC").Limit(limit).Find(&commits).Error
	if err != nil {
		return nil, err
	}
	out := make([]LargestCommit, 0, len(commits))
	for _, c := range commits {
		out = append(out, LargestCommit{
			Hash: c.Hash, Author: c.Author, Email: c.Email,
			Date: c.Date.Format(time.RFC3339), Message: c.Message,
			FilesChanged: c.FilesChanged, Insertions: c.Insertions, Deletions: c.Deletions,
		})
	}
	return out, nil
}

// CommitFeed returns recent commits, newest first.
func CommitFeed(db *gorm.DB, repoID uint, limit, offset int) ([]models.Commit, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	var commits []models.Commit
	err := db.Where("repo_id = ?", repoID).Order("date DESC").
		Limit(limit).Offset(offset).Find(&commits).Error
	return commits, err
}

// Leaderboard ranks contributors by commit count.
func Leaderboard(db *gorm.DB, repoID uint) ([]models.Contributor, error) {
	var rows []models.Contributor
	err := db.Where("repo_id = ?", repoID).Order("commits DESC").Find(&rows).Error
	return rows, err
}

func sortDays(days []Day) {
	for i := 1; i < len(days); i++ {
		for j := i; j > 0 && days[j].Date < days[j-1].Date; j-- {
			days[j], days[j-1] = days[j-1], days[j]
		}
	}
}

func sortDaysString(days []string) {
	for i := 1; i < len(days); i++ {
		for j := i; j > 0 && days[j] < days[j-1]; j-- {
			days[j], days[j-1] = days[j-1], days[j]
		}
	}
}

func sortOwners(owners []OwnerSummary) {
	for i := 1; i < len(owners); i++ {
		for j := i; j > 0 && owners[j].Files > owners[j-1].Files; j-- {
			owners[j], owners[j-1] = owners[j-1], owners[j]
		}
	}
}

func daysBetween(start, end string) int {
	s, _ := time.Parse("2006-01-02", start)
	e, _ := time.Parse("2006-01-02", end)
	return int(e.Sub(s)/24/time.Hour) + 1
}
