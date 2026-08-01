// Package search provides indexed file and content search. Filename, folder,
// and extension queries hit the files table; content queries use the FTS5
// index with an exact-match verification pass, and regex queries scan file
// content directly under a configurable bound.
package search

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Mode values.
const (
	ModeAll       = "all"
	ModeFilename  = "filename"
	ModeFolder    = "folder"
	ModeExtension = "extension"
	ModeContent   = "content"
	ModeRegex     = "regex"
)

// Query describes a search request.
type Query struct {
	RepoID        uint   `json:"repoId"`
	Text          string `json:"text"`
	Mode          string `json:"mode"`
	CaseSensitive bool   `json:"caseSensitive"`
	WholeWord     bool   `json:"wholeWord"`
	ExtFilter     string `json:"extFilter"`
	Limit         int    `json:"limit"`
	Offset        int    `json:"offset"`
}

// Match is one matching line in a file.
type Match struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

// Hit is one file containing matches.
type Hit struct {
	FileID     uint    `json:"fileId"`
	Path       string  `json:"path"`
	Language   string  `json:"language"`
	Size       int64   `json:"size"`
	LinesTotal int     `json:"linesTotal"`
	Matches    []Match `json:"matches"`
}

// Result is a page of search hits.
type Result struct {
	Hits      []Hit `json:"hits"`
	Total     int   `json:"total"`
	Truncated bool  `json:"truncated"`
}

// Store searches the repository data.
type Store struct {
	db *gorm.DB
}

// New builds a search Store.
func New(db *gorm.DB) *Store { return &Store{db: db} }

// Search runs a query against the given repository.
func (s *Store) Search(ctx context.Context, q Query, settings config.Settings) (Result, error) {
	if q.Limit <= 0 || q.Limit > 200 {
		q.Limit = 50
	}
	var repo models.Repository
	if err := s.db.First(&repo, q.RepoID).Error; err != nil {
		return Result{}, fmt.Errorf("load repo: %w", err)
	}
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" {
		return Result{}, nil
	}

	matcher, err := buildMatcher(q)
	if err != nil {
		return Result{}, err
	}

	switch q.Mode {
	case ModeFilename:
		return s.filenameSearch(ctx, q, &repo)
	case ModeFolder:
		return s.folderSearch(ctx, q, &repo)
	case ModeExtension:
		return s.extensionSearch(ctx, q, &repo)
	case ModeRegex:
		return s.contentSearch(ctx, q, &repo, matcher, settings, true)
	default:
		return s.contentSearch(ctx, q, &repo, matcher, settings, false)
	}
}

func (s *Store) filenameSearch(ctx context.Context, q Query, repo *models.Repository) (Result, error) {
	var rows []models.File
	err := s.db.Where("repo_id = ?", repo.ID).
		Where(nameMatch("name", q.Text, q.CaseSensitive), likeArgs(q.Text, q.CaseSensitive)).
		Order("path ASC").Limit(q.Limit + 1).Offset(q.Offset).
		Find(&rows).Error
	if err != nil {
		return Result{}, err
	}
	return fileHits(rows, q.Limit), nil
}

func (s *Store) folderSearch(ctx context.Context, q Query, repo *models.Repository) (Result, error) {
	var rows []models.File
	err := s.db.Where("repo_id = ?", repo.ID).
		Where(nameMatch("folder", q.Text, q.CaseSensitive), likeArgs(q.Text, q.CaseSensitive)).
		Order("path ASC").Limit(q.Limit + 1).Offset(q.Offset).
		Find(&rows).Error
	if err != nil {
		return Result{}, err
	}
	return fileHits(rows, q.Limit), nil
}

// nameMatch builds a substring clause. instr uses binary (case-sensitive)
// comparison; lower() + LIKE is case-insensitive.
func nameMatch(column, text string, caseSensitive bool) string {
	if caseSensitive {
		return "instr(" + column + ", ?) > 0"
	}
	return "lower(" + column + ") LIKE lower(?) ESCAPE '\\'"
}

// likeArgs returns the pattern bound to nameMatch's placeholder.
func likeArgs(text string, caseSensitive bool) string {
	if caseSensitive {
		return text
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(text)
	return "%" + escaped + "%"
}

func (s *Store) extensionSearch(ctx context.Context, q Query, repo *models.Repository) (Result, error) {
	var rows []models.File
	err := s.db.Where("repo_id = ?", repo.ID).
		Where("extension = ?", strings.TrimPrefix(strings.ToLower(q.Text), ".")).
		Order("path ASC").Limit(q.Limit + 1).Offset(q.Offset).
		Find(&rows).Error
	if err != nil {
		return Result{}, err
	}
	return fileHits(rows, q.Limit), nil
}

// contentSearch finds files whose content matches. regexMode skips the FTS5
// prefilter and scans files directly, bounded by the search limit.
func (s *Store) contentSearch(ctx context.Context, q Query, repo *models.Repository, matcher *regexp.Regexp, settings config.Settings, regexMode bool) (Result, error) {
	var rows []models.File
	base := s.db.Where("repo_id = ?", repo.ID)
	if q.ExtFilter != "" {
		base = base.Where("extension = ?", strings.TrimPrefix(strings.ToLower(q.ExtFilter), "."))
	}

	if !regexMode {
		// FTS5 prefilter narrows candidates to files containing the tokens.
		var ids []uint
		fts := ftsQuery(q.Text)
		err := s.db.Raw(
			"SELECT rowid FROM file_fts WHERE file_fts MATCH ? AND repo_id = ? ORDER BY rowid LIMIT ?",
			fts, repo.ID, 5000,
		).Scan(&ids).Error
		if err != nil {
			return Result{}, fmt.Errorf("fts search: %w", err)
		}
		if len(ids) == 0 {
			return Result{}, nil
		}
		base = base.Where("id IN ?", ids)
	}

	if err := base.Order("id ASC").Limit(q.Limit + 1).Offset(q.Offset).Find(&rows).Error; err != nil {
		return Result{}, err
	}

	res := Result{}
	scanned := 0
	for _, f := range rows {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		scanned++
		if scanned > settings.MaxSearchFiles {
			res.Truncated = true
			break
		}
		data, err := os.ReadFile(filepath.Join(repo.Path, f.Path))
		if err != nil {
			continue
		}
		hit := Hit{
			FileID:     f.ID,
			Path:       f.Path,
			Language:   f.Language,
			Size:       f.Size,
			LinesTotal: f.LinesTotal,
		}
		for i, line := range strings.Split(string(data), "\n") {
			if matcher.MatchString(line) {
				hit.Matches = append(hit.Matches, Match{Line: i + 1, Text: snippet(line)})
				if len(hit.Matches) >= 50 {
					break
				}
			}
		}
		if len(hit.Matches) > 0 {
			res.Hits = append(res.Hits, hit)
		}
	}
	res.Total = len(res.Hits)
	if len(res.Hits) > q.Limit {
		res.Hits = res.Hits[:q.Limit]
	}
	return res, nil
}

func fileHits(rows []models.File, limit int) Result {
	res := Result{}
	for _, f := range rows {
		res.Hits = append(res.Hits, Hit{
			FileID:     f.ID,
			Path:       f.Path,
			Language:   f.Language,
			Size:       f.Size,
			LinesTotal: f.LinesTotal,
		})
	}
	res.Total = len(res.Hits)
	if len(res.Hits) > limit {
		res.Hits = res.Hits[:limit]
		res.Truncated = true
	}
	return res
}

// ftsQuery builds an FTS5 MATCH expression ANDing quoted tokens.
func ftsQuery(text string) string {
	var parts []string
	for _, tok := range strings.Fields(text) {
		parts = append(parts, `"`+strings.ReplaceAll(tok, `"`, `""`)+`"`)
	}
	return strings.Join(parts, " AND ")
}

// buildMatcher compiles the exact-match regex used to verify content hits.
func buildMatcher(q Query) (*regexp.Regexp, error) {
	var pattern string
	if q.Mode == ModeRegex {
		pattern = q.Text
		if q.WholeWord {
			pattern = `\b` + pattern + `\b`
		}
	} else {
		pattern = regexp.QuoteMeta(q.Text)
		if q.WholeWord {
			pattern = `\b` + pattern + `\b`
		}
	}
	if !q.CaseSensitive {
		pattern = `(?i)` + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}
	return re, nil
}

func snippet(line string) string {
	if len(line) > 240 {
		return line[:240]
	}
	return line
}
