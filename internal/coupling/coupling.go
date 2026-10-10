// Package coupling finds files that change together. Files that keep landing
// in the same commits depend on each other whether or not the code says so;
// pairs without an import between them are the hidden dependencies.
package coupling

import (
	"fmt"
	"path"
	"slices"
	"strings"

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
	// Link explains why the pair is expected to change together, or is
	// empty when nothing does: a hidden dependency.
	Link Link `json:"link"`
}

// Link is a reason two files are expected to change together.
type Link string

// Link values.
const (
	// LinkImport: one file imports the other or the folder holding it.
	LinkImport Link = "import"
	// LinkPackage: Go files in one folder share a package without imports.
	LinkPackage Link = "package"
	// LinkTest: a test and the file it is named after.
	LinkTest Link = "test"
	// LinkLockfile: a manifest and the lockfile generated from it.
	LinkLockfile Link = "lockfile"
)

// Query selects which pairs List returns.
type Query struct {
	// Path limits the list to pairs involving one file.
	Path string
	// Hidden keeps only pairs nothing explains (see Link).
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
	imports, err := importLinks(db, repoID, rows)
	if err != nil {
		return nil, err
	}
	out := make([]Pair, 0, min(len(rows), q.Limit))
	for _, r := range rows {
		p := Pair{
			FileA: r.FileA, FileB: r.FileB, Shared: r.Shared,
			RevisionsA: r.RevisionsA, RevisionsB: r.RevisionsB, Degree: r.Degree,
		}
		if imports(r.FileA, r.FileB) || imports(r.FileB, r.FileA) {
			p.Link = LinkImport
		} else {
			p.Link = conventionLink(r.FileA, r.FileB)
		}
		if q.Hidden && p.Link != "" {
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

// lockfilesOf lists the lockfiles generated from a manifest.
func lockfilesOf(manifest string) []string {
	switch manifest {
	case "package.json":
		return []string{"package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb", "npm-shrinkwrap.json"}
	case "go.mod":
		return []string{"go.sum"}
	case "Cargo.toml":
		return []string{"Cargo.lock"}
	case "composer.json":
		return []string{"composer.lock"}
	case "pyproject.toml":
		return []string{"poetry.lock", "uv.lock", "pdm.lock"}
	case "Pipfile":
		return []string{"Pipfile.lock"}
	case "Gemfile":
		return []string{"Gemfile.lock"}
	}
	return nil
}

// conventionLink explains a pair by naming and layout conventions: Go files
// sharing a package, a test and its subject, a manifest and its lockfile.
func conventionLink(a, b string) Link {
	dirA, nameA := path.Split(a)
	dirB, nameB := path.Split(b)
	if dirA == dirB && strings.HasSuffix(a, ".go") && strings.HasSuffix(b, ".go") {
		return LinkPackage
	}
	if subjectA, ok := testSubject(nameA); ok && subjectA == nameB {
		return LinkTest
	}
	if subjectB, ok := testSubject(nameB); ok && subjectB == nameA {
		return LinkTest
	}
	if dirA == dirB && (slices.Contains(lockfilesOf(nameA), nameB) || slices.Contains(lockfilesOf(nameB), nameA)) {
		return LinkLockfile
	}
	return ""
}

// testSubject returns the file a test file is named after, such as
// scanner.go for scanner_test.go, api.ts for api.test.ts or api.spec.ts,
// util.py for test_util.py, and Parser.java for ParserTest.java.
func testSubject(name string) (string, bool) {
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for _, suffix := range []string{"_test", ".test", ".spec", "Tests", "Test", "_spec"} {
		if base, ok := strings.CutSuffix(stem, suffix); ok && base != "" {
			return base + ext, true
		}
	}
	if base, ok := strings.CutPrefix(stem, "test_"); ok && base != "" {
		return base + ext, true
	}
	return "", false
}
