// Package export renders repository data as CSV or JSON downloads.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
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

// batchSize bounds how many rows an export holds in memory at once.
const batchSize = 1000

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
	header := []string{"path", "language", "extension", "lines_total", "lines_code", "lines_comment", "lines_blank",
		"complexity", "imports", "exports", "func_count", "max_func_len", "max_nesting", "author", "commits", "size"}
	// Paths are unique per repository, so the path alone keys the pages.
	page := func(last *models.File) *gorm.DB {
		q := db.Where("repo_id = ?", repoID).Order("path ASC")
		if last != nil {
			q = q.Where("path > ?", last.Path)
		}
		return q
	}
	return stream(page, format, w, header, func(f models.File) []string {
		return []string{f.Path, f.Language, f.Extension,
			strconv.Itoa(f.LinesTotal), strconv.Itoa(f.LinesCode), strconv.Itoa(f.LinesComment), strconv.Itoa(f.LinesBlank),
			strconv.Itoa(f.Complexity), strconv.Itoa(f.Imports), strconv.Itoa(f.Exports),
			strconv.Itoa(f.FuncCount), strconv.Itoa(f.MaxFuncLen), strconv.Itoa(f.MaxNesting),
			f.Author, strconv.Itoa(f.Commits), strconv.FormatInt(f.Size, 10)}
	})
}

func exportCommits(db *gorm.DB, repoID uint, format Format, w io.Writer) error {
	header := []string{"hash", "author", "email", "date", "message", "files_changed", "insertions", "deletions", "merge"}
	// Newest first; the ID breaks ties between commits made the same second.
	page := func(last *models.Commit) *gorm.DB {
		q := db.Where("repo_id = ?", repoID).Order("date DESC, id ASC")
		if last != nil {
			q = q.Where("date < ? OR (date = ? AND id > ?)", last.Date, last.Date, last.ID)
		}
		return q
	}
	return stream(page, format, w, header, func(c models.Commit) []string {
		return []string{c.Hash, c.Author, c.Email, c.Date.UTC().Format(time.RFC3339), c.Message,
			strconv.Itoa(c.FilesChanged), strconv.Itoa(c.Insertions), strconv.Itoa(c.Deletions), strconv.FormatBool(c.IsMerge)}
	})
}

func exportContributors(db *gorm.DB, repoID uint, format Format, w io.Writer) error {
	header := []string{"name", "email", "commits", "insertions", "deletions", "first_commit_at", "last_commit_at"}
	page := func(last *models.Contributor) *gorm.DB {
		q := db.Where("repo_id = ?", repoID).Order("commits DESC, id ASC")
		if last != nil {
			q = q.Where("commits < ? OR (commits = ? AND id > ?)", last.Commits, last.Commits, last.ID)
		}
		return q
	}
	return stream(page, format, w, header, func(c models.Contributor) []string {
		return []string{c.Name, c.Email, strconv.Itoa(c.Commits), strconv.Itoa(c.Insertions), strconv.Itoa(c.Deletions),
			c.FirstCommitAt.UTC().Format(time.RFC3339), c.LastCommitAt.UTC().Format(time.RFC3339)}
	})
}

// stream writes rows page by page, as a JSON array or as CSV with header.
// page returns the query for the rows after last (nil for the first page),
// keyed on the sort columns, so each page is one short query and memory stays
// bounded by the page, not the table.
func stream[T any](page func(last *T) *gorm.DB, format Format, w io.Writer, header []string, record func(T) []string) error {
	var cw *csv.Writer
	if format == FormatJSON {
		if _, err := io.WriteString(w, "["); err != nil {
			return err
		}
	} else {
		cw = csv.NewWriter(w)
		if err := cw.Write(header); err != nil {
			return err
		}
	}
	var last *T
	written := 0
	for {
		var batch []T
		if err := page(last).Limit(batchSize).Find(&batch).Error; err != nil {
			return fmt.Errorf("export: %w", err)
		}
		for _, row := range batch {
			if err := writeRow(w, cw, row, record, written); err != nil {
				return err
			}
			written++
		}
		if cw != nil {
			cw.Flush()
			if err := cw.Error(); err != nil {
				return err
			}
		}
		if len(batch) < batchSize {
			break
		}
		last = &batch[len(batch)-1]
	}
	if format == FormatJSON {
		_, err := io.WriteString(w, "]\n")
		return err
	}
	return nil
}

// writeRow writes one row as a JSON array element (comma-separated after the
// first) or as a CSV record.
func writeRow[T any](w io.Writer, cw *csv.Writer, row T, record func(T) []string, index int) error {
	if cw != nil {
		rec := record(row)
		for i := range rec {
			rec[i] = defuseFormula(rec[i])
		}
		return cw.Write(rec)
	}
	data, err := json.Marshal(row)
	if err != nil {
		return err
	}
	if index > 0 {
		if _, err := io.WriteString(w, ","); err != nil {
			return err
		}
	}
	_, err = w.Write(data)
	return err
}

// defuseFormula keeps spreadsheet apps from running a cell as a formula.
// Commit messages and author names come from the repository, so a value
// like "=HYPERLINK(...)" must stay text when the CSV is opened (CWE-1236).
func defuseFormula(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
