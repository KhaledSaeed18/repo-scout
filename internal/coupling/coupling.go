// Package coupling finds files that change together. Files that keep landing
// in the same commits depend on each other whether or not the code says so;
// pairs without an import between them are the hidden dependencies.
package coupling

import (
	"fmt"
	"path"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

const (
	// maxCommitFiles skips sweeping commits (formatting, renames, vendoring)
	// whose files change together only by accident.
	maxCommitFiles = 50
	// minShared is how many commits a pair must share to count.
	minShared = 3
	// minDegree is the lowest coupling degree kept (see Pair.Degree).
	minDegree = 0.3
	// maxPairs bounds what one scan stores.
	maxPairs = 5000
)

// Compute stores the coupled file pairs of a repository, from the commit
// files recorded by the history stage. It runs in SQL so memory stays flat
// however long the history is.
func Compute(db *gorm.DB, repoID uint) error {
	err := db.Exec(`WITH changes AS (
			SELECT cf.commit_id, cf.path
			FROM commit_files cf
			JOIN commits c ON c.id = cf.commit_id
			JOIN files f ON f.repo_id = cf.repo_id AND f.path = cf.path
			WHERE cf.repo_id = ? AND c.files_changed BETWEEN 2 AND ?
		),
		revisions AS (
			SELECT path, COUNT(*) AS n FROM changes GROUP BY path
		),
		pairs AS (
			SELECT a.path AS file_a, b.path AS file_b, COUNT(*) AS shared
			FROM changes a
			JOIN changes b ON a.commit_id = b.commit_id AND a.path < b.path
			GROUP BY a.path, b.path
			HAVING COUNT(*) >= ?
		)
		INSERT INTO file_couplings (repo_id, file_a, file_b, shared, revisions_a, revisions_b, degree)
		SELECT ?, p.file_a, p.file_b, p.shared, ra.n, rb.n, p.shared * 2.0 / (ra.n + rb.n) AS degree
		FROM pairs p
		JOIN revisions ra ON ra.path = p.file_a
		JOIN revisions rb ON rb.path = p.file_b
		WHERE p.shared * 2.0 / (ra.n + rb.n) >= ?
		ORDER BY degree DESC, p.shared DESC, p.file_a, p.file_b
		LIMIT ?`, repoID, maxCommitFiles, minShared, repoID, minDegree, maxPairs).Error
	if err != nil {
		return fmt.Errorf("change coupling: %w", err)
	}
	return nil
}

// Pair is two files that change together.
type Pair struct {
	FileA string `json:"fileA"`
	FileB string `json:"fileB"`
	// Shared is how many commits changed both files.
	Shared     int `json:"shared"`
	RevisionsA int `json:"revisionsA"`
	RevisionsB int `json:"revisionsB"`
	// Degree is shared commits over the average of the two files' commits:
	// 1 means they always change together.
	Degree float64 `json:"degree"`
	// Linked reports an import between the two files, directly or through
	// the other file's package folder.
	Linked bool `json:"linked"`
}

// Query selects which pairs List returns.
type Query struct {
	// Path limits the list to pairs involving one file.
	Path string
	// Hidden keeps only pairs without an import between them.
	Hidden bool
	Limit  int
}

// List returns stored pairs, strongest first.
func List(db *gorm.DB, repoID uint, q Query) ([]Pair, error) {
	tx := db.Model(&models.FileCoupling{}).Where("repo_id = ?", repoID)
	if q.Path != "" {
		tx = tx.Where("file_a = ? OR file_b = ?", q.Path, q.Path)
	}
	var rows []models.FileCoupling
	// Hidden pairs are filtered after the import lookup, so read all of them.
	if !q.Hidden {
		tx = tx.Limit(q.Limit)
	}
	if err := tx.Order("degree DESC, shared DESC, file_a, file_b").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list coupling: %w", err)
	}
	linked, err := importLinks(db, repoID, rows)
	if err != nil {
		return nil, err
	}
	out := make([]Pair, 0, min(len(rows), q.Limit))
	for _, r := range rows {
		p := Pair{
			FileA: r.FileA, FileB: r.FileB, Shared: r.Shared,
			RevisionsA: r.RevisionsA, RevisionsB: r.RevisionsB, Degree: r.Degree,
			Linked: linked(r.FileA, r.FileB) || linked(r.FileB, r.FileA),
		}
		if q.Hidden && p.Linked {
			continue
		}
		out = append(out, p)
		if len(out) == q.Limit {
			break
		}
	}
	return out, nil
}

// importLinks loads the resolved imports leaving the files in rows and
// returns a check for whether from imports to, either the file itself or the
// folder holding it (Go and similar languages import packages, not files).
func importLinks(db *gorm.DB, repoID uint, rows []models.FileCoupling) (func(from, to string) bool, error) {
	seen := map[string]bool{}
	var files []string
	for _, r := range rows {
		for _, f := range []string{r.FileA, r.FileB} {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	edges := map[[2]string]bool{}
	// Chunked to stay under SQLite's bound-parameter limit.
	for i := 0; i < len(files); i += 500 {
		var batch []models.ImportEdge
		if err := db.Select("from_file", "to_file").
			Where("repo_id = ? AND resolved = ? AND from_file IN ?", repoID, true, files[i:min(i+500, len(files))]).
			Find(&batch).Error; err != nil {
			return nil, fmt.Errorf("load imports: %w", err)
		}
		for _, e := range batch {
			edges[[2]string{e.FromFile, e.ToFile}] = true
		}
	}
	return func(from, to string) bool {
		return edges[[2]string{from, to}] || edges[[2]string{from, path.Dir(to)}]
	}, nil
}
