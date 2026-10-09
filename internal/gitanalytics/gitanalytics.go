// Package gitanalytics serves aggregated views over stored git history:
// heatmaps, streaks, ownership, largest commits, and leaderboards.
package gitanalytics

import (
	"sort"
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

// ComputeHeatmap rolls commit counts up by calendar day and by weekday/hour,
// both on each author's local clock.
func ComputeHeatmap(db *gorm.DB, repoID uint) (Heatmap, error) {
	var commits []models.Commit
	err := db.Select("date", "tz_offset").Where("repo_id = ?", repoID).Find(&commits).Error
	if err != nil {
		return Heatmap{}, err
	}
	h := Heatmap{Total: len(commits), Daily: []Day{}, Hourly: make([][]int, 7)}
	for i := range h.Hourly {
		h.Hourly[i] = make([]int, 24)
	}
	if len(commits) == 0 {
		return h, nil
	}
	byDay := map[string]int{}
	for _, c := range commits {
		local := c.LocalTime()
		byDay[local.Format(dayLayout)]++
		h.Hourly[int(local.Weekday())][local.Hour()]++
	}
	days := make([]Day, 0, len(byDay))
	for d, n := range byDay {
		days = append(days, Day{Date: d, Count: n})
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Date < days[j].Date })
	h.Daily = days
	h.Start = days[0].Date
	h.End = days[len(days)-1].Date
	return h, nil
}

// Streaks analyzes consecutive-day activity for one contributor. A streak is
// only reported as current when it reaches today or yesterday relative to now.
func Streaks(db *gorm.DB, repoID uint, email string, now time.Time) (StreaksResult, error) {
	var commits []models.Commit
	err := db.Select("date", "tz_offset").Where("repo_id = ? AND email = ?", repoID, email).
		Order("date ASC").Find(&commits).Error
	if err != nil {
		return StreaksResult{}, err
	}
	dates := make([]time.Time, len(commits))
	for i, c := range commits {
		dates[i] = c.LocalTime()
	}
	return streaksFromDates(email, dates, now), nil
}

// AllStreaks computes streaks for every contributor of a repository in a
// single pass, ordered by longest streak (then commits) descending.
func AllStreaks(db *gorm.DB, repoID uint, now time.Time) ([]StreaksResult, error) {
	var commits []models.Commit
	err := db.Select("email", "date", "tz_offset").Where("repo_id = ?", repoID).
		Order("date ASC").Find(&commits).Error
	if err != nil {
		return nil, err
	}
	byEmail := map[string][]time.Time{}
	for _, c := range commits {
		byEmail[c.Email] = append(byEmail[c.Email], c.LocalTime())
	}
	out := make([]StreaksResult, 0, len(byEmail))
	for email, dates := range byEmail {
		out = append(out, streaksFromDates(email, dates, now))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Longest.Days != out[j].Longest.Days {
			return out[i].Longest.Days > out[j].Longest.Days
		}
		if out[i].TotalCommits != out[j].TotalCommits {
			return out[i].TotalCommits > out[j].TotalCommits
		}
		return out[i].Email < out[j].Email
	})
	return out, nil
}

// streaksFromDates derives streaks from commit timestamps (any order), using
// the calendar day of each timestamp in its own location.
func streaksFromDates(email string, dates []time.Time, now time.Time) StreaksResult {
	res := StreaksResult{Email: email, All: []Streak{}, TotalCommits: len(dates)}
	if len(dates) == 0 {
		return res
	}
	seen := map[string]bool{}
	for _, d := range dates {
		seen[d.Format(dayLayout)] = true
	}
	days := make([]string, 0, len(seen))
	for d := range seen {
		days = append(days, d)
	}
	sort.Strings(days)
	res.ActiveDays = len(days)

	start, prev := days[0], days[0]
	closeStreak := func(end string) {
		st := Streak{Start: start, End: end, Days: daysBetween(start, end)}
		res.All = append(res.All, st)
		if st.Days > res.Longest.Days {
			res.Longest = st
		}
	}
	for _, d := range days[1:] {
		if daysBetween(prev, d) > 2 {
			closeStreak(prev)
			start = d
		}
		prev = d
	}
	closeStreak(prev)

	last := res.All[len(res.All)-1]
	if daysBetween(last.End, now.UTC().Format(dayLayout)) <= 2 {
		res.Current = last
	}
	return res
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
	sort.Slice(o.ByAuthor, func(i, j int) bool {
		if o.ByAuthor[i].Files != o.ByAuthor[j].Files {
			return o.ByAuthor[i].Files > o.ByAuthor[j].Files
		}
		return o.ByAuthor[i].Author < o.ByAuthor[j].Author
	})
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

const dayLayout = "2006-01-02"

// daysBetween counts calendar days from start to end inclusive.
func daysBetween(start, end string) int {
	s, _ := time.Parse(dayLayout, start)
	e, _ := time.Parse(dayLayout, end)
	return int(e.Sub(s)/24/time.Hour) + 1
}
