package coupling

import (
	"fmt"
	"testing"
	"time"

	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// TestComputeScales guards against a quadratic plan: a self-join without an
// index took minutes on a history this size. Done right it takes well under a
// second, so the bound is generous enough never to flake.
func TestComputeScales(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test")
	}
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	const repoID, fileCount, commitCount, perCommit = 1, 400, 6000, 5
	files := make([]models.File, fileCount)
	for i := range files {
		files[i] = models.File{RepoID: repoID, Path: fmt.Sprintf("src/f%03d.ts", i)}
	}
	if err := db.CreateInBatches(files, 500).Error; err != nil {
		t.Fatal(err)
	}
	commits := make([]models.Commit, commitCount)
	for i := range commits {
		commits[i] = models.Commit{RepoID: repoID, Hash: fmt.Sprintf("c%05d", i), FilesChanged: perCommit,
			Date: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(i) * time.Hour)}
	}
	if err := db.CreateInBatches(commits, 500).Error; err != nil {
		t.Fatal(err)
	}
	var changes []models.CommitFile
	for i, c := range commits {
		// Neighbouring files change together, so many pairs pass the thresholds.
		for j := range perCommit {
			changes = append(changes, models.CommitFile{RepoID: repoID, CommitID: c.ID, Path: files[(i+j)%fileCount].Path})
		}
	}
	if err := db.CreateInBatches(changes, 500).Error; err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	if err := Compute(db, repoID); err != nil {
		t.Fatalf("compute: %v", err)
	}
	if took := time.Since(start); took > 15*time.Second {
		t.Fatalf("coupling took %s for %d commit files; the join is not using its index", took, len(changes))
	}
	var pairs int64
	db.Model(&models.FileCoupling{}).Where("repo_id = ?", repoID).Count(&pairs)
	if pairs == 0 {
		t.Fatal("expected coupled pairs")
	}
	// Computing again must not trip over a leftover temporary table.
	if err := Compute(db, repoID); err != nil {
		t.Fatalf("second compute: %v", err)
	}
}
