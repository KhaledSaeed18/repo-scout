// Package export renders repository data as CSV or JSON downloads.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Kind identifies the dataset to export.
type Kind string

const (
	KindFiles        Kind = "files"
	KindCommits      Kind = "commits"
	KindContributors Kind = "contributors"
)

// Format identifies the serialization format.
type Format string

const (
	FormatCSV  Format = "csv"
	FormatJSON Format = "json"
)

// Params describes an export request.
type Params struct {
	Kind   Kind
	Format Format
}

// Export streams the requested dataset to w.
func Export(db *gorm.DB, repoID uint, p Params, w io.Writer) error {
	switch p.Kind {
	case KindFiles:
		return exportFiles(db, repoID, p.Format, w)
	case KindCommits:
		return exportCommits(db, repoID, p.Format, w)
	case KindContributors:
		return exportContributors(db, repoID, p.Format, w)
	default:
		return fmt.Errorf("unknown export kind %q", p.Kind)
	}
}

func exportFiles(db *gorm.DB, repoID uint, format Format, w io.Writer) error {
	var rows []models.File
	if err := db.Where("repo_id = ?", repoID).Order("path ASC").Find(&rows).Error; err != nil {
		return err
	}
	if format == FormatJSON {
		return json.NewEncoder(w).Encode(rows)
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"path", "language", "extension", "lines_total", "lines_code", "lines_comment", "lines_blank", "complexity", "imports", "exports", "func_count", "max_func_len", "max_nesting", "author", "commits", "size"}); err != nil {
		return err
	}
	for _, f := range rows {
		rec := []string{f.Path, f.Language, f.Extension,
			strconv.Itoa(f.LinesTotal), strconv.Itoa(f.LinesCode), strconv.Itoa(f.LinesComment), strconv.Itoa(f.LinesBlank),
			strconv.Itoa(f.Complexity), strconv.Itoa(f.Imports), strconv.Itoa(f.Exports),
			strconv.Itoa(f.FuncCount), strconv.Itoa(f.MaxFuncLen), strconv.Itoa(f.MaxNesting),
			f.Author, strconv.Itoa(f.Commits), strconv.FormatInt(f.Size, 10)}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func exportCommits(db *gorm.DB, repoID uint, format Format, w io.Writer) error {
	var rows []models.Commit
	if err := db.Where("repo_id = ?", repoID).Order("date DESC").Find(&rows).Error; err != nil {
		return err
	}
	if format == FormatJSON {
		return json.NewEncoder(w).Encode(rows)
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"hash", "author", "email", "date", "message", "files_changed", "insertions", "deletions", "merge"}); err != nil {
		return err
	}
	for _, c := range rows {
		rec := []string{c.Hash, c.Author, c.Email, c.Date.UTC().Format(time.RFC3339), c.Message,
			strconv.Itoa(c.FilesChanged), strconv.Itoa(c.Insertions), strconv.Itoa(c.Deletions), strconv.FormatBool(c.IsMerge)}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	return nil
}

func exportContributors(db *gorm.DB, repoID uint, format Format, w io.Writer) error {
	var rows []models.Contributor
	if err := db.Where("repo_id = ?", repoID).Order("commits DESC").Find(&rows).Error; err != nil {
		return err
	}
	if format == FormatJSON {
		return json.NewEncoder(w).Encode(rows)
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()
	if err := cw.Write([]string{"name", "email", "commits", "insertions", "deletions", "first_commit_at", "last_commit_at"}); err != nil {
		return err
	}
	for _, c := range rows {
		rec := []string{c.Name, c.Email, strconv.Itoa(c.Commits), strconv.Itoa(c.Insertions), strconv.Itoa(c.Deletions),
			c.FirstCommitAt.UTC().Format(time.RFC3339), c.LastCommitAt.UTC().Format(time.RFC3339)}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	return nil
}
