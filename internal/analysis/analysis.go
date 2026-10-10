// Package analysis orchestrates a repository scan as a sequence of stages
// that reports progress through the job system.
package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/architecture"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/deps"
	"github.com/KhaledSaeed18/repo-scout/internal/duplicates"
	"github.com/KhaledSaeed18/repo-scout/internal/gitrepo"
	"github.com/KhaledSaeed18/repo-scout/internal/jobs"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/scanner"
	"github.com/KhaledSaeed18/repo-scout/internal/search"
)

// stageCount is the number of pipeline stages (used for progress mapping).
const stageCount = 7

// Runner executes repository scans. It implements jobs.Runner.
type Runner struct {
	db *gorm.DB
}

// New builds a Runner.
func New(db *gorm.DB) *Runner { return &Runner{db: db} }

var manifestNames = []string{
	"package.json", "composer.json", "go.mod", "Cargo.toml",
	"pom.xml", "requirements.txt", "requirements-dev.txt",
}

// Run scans the repository identified by repoID. It reports progress through
// rep and honors pause/cancel through rep.Checkpoint and ctx.
//
// Results are written under a staging ID and swapped in only when every stage
// succeeds, so a rescan that fails or is cancelled leaves the previous
// results in place.
func (r *Runner) Run(ctx context.Context, repoID, jobID uint, rep jobs.Reporter, settings config.Settings) error {
	var repo models.Repository
	if err := r.db.First(&repo, repoID).Error; err != nil {
		return fmt.Errorf("load repo: %w", err)
	}
	if err := r.db.Model(&repo).Update("status", models.RepoScanning).Error; err != nil {
		return err
	}
	// work is the repository as the stages see it: same folder, staging ID.
	work := repo
	work.ID = database.StagingID(repo.ID)
	if err := database.ClearRepoData(r.db, work.ID); err != nil {
		return err
	}

	root := repo.Path
	read := func(rel string) (string, error) {
		data, err := os.ReadFile(filepath.Join(root, rel))
		return string(data), err
	}

	stages := []struct {
		name string
		fn   func() error
	}{
		{"git metadata", func() error { return r.gitMeta(ctx, &work) }},
		{"scanning files", func() error {
			return r.fileScan(ctx, &work, settings, rep, 1, stageCount)
		}},
		{"git history", func() error { return r.gitHistory(ctx, &work, rep, read) }},
		{"dependencies", func() error { return r.dependencies(ctx, &work, read) }},
		{"import graph", func() error { return r.importGraph(ctx, &work, read) }},
		{"duplicates", func() error { return r.duplicates(ctx, &work, settings, read) }},
		{"content index", func() error { return r.contentIndex(ctx, &work, read) }},
	}

	for i, st := range stages {
		if err := r.runStage(ctx, rep, i, len(stages), st.name, st.fn); err != nil {
			r.abandon(&repo)
			return err
		}
	}
	if err := r.promote(&repo, &work); err != nil {
		r.abandon(&repo)
		return err
	}
	return nil
}

// promote swaps the staged results in and marks the repository ready, in one
// transaction so readers see either the old scan or the new one.
func (r *Runner) promote(repo, work *models.Repository) error {
	summary, err := r.summary(work.ID)
	if err != nil {
		return err
	}
	now := time.Now()
	summary["git_remote"] = work.GitRemote
	summary["head_commit"] = work.HeadCommit
	summary["default_branch"] = work.DefaultBranch
	summary["status"] = models.RepoReady
	summary["last_scanned_at"] = now
	summary["updated_at"] = now
	return r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.Repository{}).Where("id = ?", repo.ID).Updates(summary)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("repository %d was removed during the scan", repo.ID)
		}
		return database.PromoteRepoData(tx, work.ID, repo.ID)
	})
}

// abandon drops a failed scan's staged rows and puts the repository back to
// how it was: ready if an earlier scan's results are still there, failed if
// it has never been scanned.
func (r *Runner) abandon(repo *models.Repository) {
	_ = database.ClearRepoData(r.db, database.StagingID(repo.ID))
	status := models.RepoFailed
	if repo.LastScannedAt != nil {
		status = models.RepoReady
	}
	r.db.Model(&models.Repository{}).Where("id = ?", repo.ID).
		Updates(map[string]any{"status": status, "updated_at": time.Now()})
}

func (r *Runner) runStage(ctx context.Context, rep jobs.Reporter, idx, count int, name string, fn func() error) error {
	rep.SetProgress(float64(idx) / float64(count))
	rep.SetMessage(name)
	if err := fn(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	rep.SetProgress(float64(idx+1) / float64(count))
	return rep.Checkpoint(ctx)
}

func (r *Runner) gitMeta(ctx context.Context, repo *models.Repository) error {
	git := gitrepo.New()
	if !git.Available() || !git.IsRepo(repo.Path) {
		return nil
	}
	m, err := git.Meta(ctx, repo.Path)
	if err != nil {
		return err
	}
	branches, err := git.Branches(ctx, repo.Path, m.DefaultBranch)
	if err != nil {
		return err
	}
	tags, err := git.Tags(ctx, repo.Path)
	if err != nil {
		return err
	}
	repo.GitRemote, repo.HeadCommit, repo.DefaultBranch = m.Remote, m.Head, m.DefaultBranch
	for i := range branches {
		branches[i].RepoID = repo.ID
	}
	for i := range tags {
		tags[i].RepoID = repo.ID
	}
	if len(branches) > 0 {
		if err := r.db.Create(&branches).Error; err != nil {
			return err
		}
	}
	if len(tags) > 0 {
		if err := r.db.Create(&tags).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) fileScan(ctx context.Context, repo *models.Repository, settings config.Settings, rep jobs.Reporter, idx, count int) error {
	sc := scanner.New(r.db)
	rep.SetMessage("scanning files")
	_, err := sc.Scan(ctx, repo.ID, repo.Path, settings, func(done, total int) {
		frac := 0.0
		if total > 0 {
			frac = float64(done) / float64(total)
		}
		rep.SetProgress(float64(idx)/float64(count) + frac/float64(count))
		rep.SetMessage(fmt.Sprintf("scanning files (%d/%d)", done, total))
		_ = rep.Checkpoint(ctx)
	})
	return err
}

func (r *Runner) gitHistory(ctx context.Context, repo *models.Repository, rep jobs.Reporter, read func(string) (string, error)) error {
	git := gitrepo.New()
	if !git.Available() || !git.IsRepo(repo.Path) {
		return nil
	}
	// Commits and the files they changed stream straight into the database
	// in batches, so memory stays bounded by the batch, not by the history.
	const batch = 500
	commits := make([]models.Commit, 0, batch)
	changes := make([][]gitrepo.FileChange, 0, batch)
	flush := func() error {
		if len(commits) == 0 {
			return nil
		}
		if err := r.db.Create(&commits).Error; err != nil {
			return fmt.Errorf("insert commits: %w", err)
		}
		var rows []models.CommitFile
		for i, c := range commits {
			for _, fc := range changes[i] {
				rows = append(rows, models.CommitFile{
					RepoID: repo.ID, CommitID: c.ID, Path: fc.Path, Additions: fc.Add, Deletions: fc.Del,
				})
			}
		}
		if len(rows) > 0 {
			if err := r.db.CreateInBatches(&rows, batch).Error; err != nil {
				return fmt.Errorf("insert commit files: %w", err)
			}
		}
		commits, changes = commits[:0], changes[:0]
		return nil
	}
	seen := 0
	contrib, files, err := git.AnalyzeHistoryWithCommits(ctx, repo.Path, func(c models.Commit, fcs []gitrepo.FileChange) error {
		c.RepoID = repo.ID
		c.FilesChanged = len(fcs)
		for _, fc := range fcs {
			c.Insertions += fc.Add
			c.Deletions += fc.Del
		}
		commits = append(commits, c)
		changes = append(changes, fcs)
		if len(commits) < batch {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		seen += len(commits)
		rep.SetMessage(fmt.Sprintf("git history (%d commits)", seen))
		return flush()
	})
	if err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}

	// Contributors rollup.
	if len(contrib) > 0 {
		rows := make([]models.Contributor, 0, len(contrib))
		for _, cs := range contrib {
			rows = append(rows, models.Contributor{
				RepoID: repo.ID, Name: cs.Name, Email: cs.Email, Commits: cs.Commits,
				Insertions: cs.Insertions, Deletions: cs.Deletions,
				FirstCommitAt: cs.FirstCommitAt, LastCommitAt: cs.LastCommitAt,
			})
		}
		if err := r.db.Create(&rows).Error; err != nil {
			return err
		}
	}

	if err := r.fileOwnership(repo.ID); err != nil {
		return err
	}
	if len(files) > 0 {
		if err := r.applyFileGitInfo(repo.ID, files); err != nil {
			return err
		}
	}
	rep.SetMessage("git history analyzed")
	return rep.Checkpoint(ctx)
}

// fileOwnership stores, for every scanned file with history, the author who
// made the most of its commits and their share. Bots such as dependabot[bot]
// are left out: they touch files but hold no knowledge of them. It runs in
// SQL so it stays flat in memory however long the history is.
func (r *Runner) fileOwnership(repoID uint) error {
	err := r.db.Exec(`INSERT INTO file_ownerships (repo_id, path, author, email, commits, share)
		SELECT repo_id, path, author, email, commits, share FROM (
			SELECT cf.repo_id, cf.path, MAX(c.author) AS author, c.email,
				COUNT(*) AS commits,
				COUNT(*) * 1.0 / SUM(COUNT(*)) OVER (PARTITION BY cf.path) AS share,
				ROW_NUMBER() OVER (PARTITION BY cf.path ORDER BY COUNT(*) DESC, MAX(c.date) DESC) AS rank
			FROM commit_files cf
			JOIN commits c ON c.id = cf.commit_id
			JOIN files f ON f.repo_id = cf.repo_id AND f.path = cf.path
			WHERE cf.repo_id = ? AND c.author NOT LIKE '%[bot]%'
			GROUP BY cf.path, CASE WHEN c.email = '' THEN c.author ELSE c.email END
		) WHERE rank = 1`, repoID).Error
	if err != nil {
		return fmt.Errorf("file ownership: %w", err)
	}
	return nil
}

// applyFileGitInfo copies git attribution onto the scanned file rows.
func (r *Runner) applyFileGitInfo(repoID uint, files map[string]*gitrepo.FileHistory) error {
	var all []models.File
	if err := r.db.Where("repo_id = ?", repoID).Find(&all).Error; err != nil {
		return err
	}
	const batch = 500
	var updates []models.File
	for _, f := range all {
		fh, ok := files[f.Path]
		if !ok {
			continue
		}
		f.Author = fh.Author
		f.Commits = fh.Commits
		if !fh.First.IsZero() {
			f.FirstCommitAt = &fh.First
		}
		if !fh.Last.IsZero() {
			f.LastCommitAt = &fh.Last
		}
		updates = append(updates, f)
		if len(updates) >= batch {
			if err := r.db.Save(&updates).Error; err != nil {
				return err
			}
			updates = updates[:0]
		}
	}
	if len(updates) > 0 {
		return r.db.Save(&updates).Error
	}
	return nil
}

func (r *Runner) dependencies(ctx context.Context, repo *models.Repository, read func(string) (string, error)) error {
	var manifests []models.File
	if err := r.db.Where("repo_id = ? AND name IN ?", repo.ID, manifestNames).Find(&manifests).Error; err != nil {
		return err
	}
	var rows []models.Dependency
	for _, m := range manifests {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		content, err := read(m.Path)
		if err != nil {
			continue
		}
		mgr, entries, err := deps.Parse(m.Path, content)
		if err != nil || mgr == "" {
			continue
		}
		for _, e := range entries {
			rows = append(rows, models.Dependency{
				RepoID: repo.ID, FilePath: m.Path, Manager: mgr,
				Name: e.Name, Version: e.Version, Scope: e.Scope,
			})
		}
	}
	if len(rows) > 0 {
		if err := r.db.Create(&rows).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) importGraph(ctx context.Context, repo *models.Repository, read func(string) (string, error)) error {
	var files []models.File
	if err := r.db.Select("path", "language").Where("repo_id = ?", repo.ID).Find(&files).Error; err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	rep, err := architecture.Build(repo.Path, files, read)
	if err != nil {
		return err
	}
	if len(rep.Edges) > 0 {
		edges := make([]models.ImportEdge, 0, len(rep.Edges))
		for _, e := range rep.Edges {
			edges = append(edges, models.ImportEdge{
				RepoID: repo.ID, FromFile: e.From, ToFile: e.To, ImportType: e.Kind, Resolved: e.Resolved,
			})
		}
		for i := 0; i < len(edges); i += 500 {
			end := i + 500
			if end > len(edges) {
				end = len(edges)
			}
			if err := r.db.Create(edges[i:end]).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) duplicates(ctx context.Context, repo *models.Repository, settings config.Settings, read func(string) (string, error)) error {
	var files []models.File
	if err := r.db.Where("repo_id = ? AND language != '' AND lines_total > 0", repo.ID).Find(&files).Error; err != nil {
		return err
	}
	if len(files) == 0 {
		return nil
	}
	det := duplicates.New(settings)
	res, err := det.Detect(ctx, repo.ID, repo.Path, files, read)
	if err != nil {
		return err
	}
	if len(res.Groups) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Batched to stay under SQLite's bound-parameter limit on big repos.
		if err := tx.CreateInBatches(&res.Groups, 500).Error; err != nil {
			return err
		}
		// The detector numbers groups 1..n by position; point each block at
		// the ID its group was stored under.
		for i := range res.Blocks {
			idx := int(res.Blocks[i].GroupID) - 1
			if idx < 0 || idx >= len(res.Groups) {
				return fmt.Errorf("duplicate block references unknown group %d", res.Blocks[i].GroupID)
			}
			res.Blocks[i].GroupID = res.Groups[idx].ID
		}
		return tx.CreateInBatches(&res.Blocks, 500).Error
	})
}

func (r *Runner) contentIndex(ctx context.Context, repo *models.Repository, read func(string) (string, error)) error {
	var files []models.File
	if err := r.db.Where("repo_id = ?", repo.ID).Find(&files).Error; err != nil {
		return err
	}
	return search.New(r.db).Reindex(ctx, repo.ID, repo.Path, files, read)
}

// summary rolls up the repository-level totals from the rows stored under
// repoID.
func (r *Runner) summary(repoID uint) (map[string]any, error) {
	var (
		fileCount, loc, code, comments, blank int
		size                                  int64
	)
	err := r.db.Raw(`SELECT COUNT(*), COALESCE(SUM(lines_total),0), COALESCE(SUM(lines_code),0),
		COALESCE(SUM(lines_comment),0), COALESCE(SUM(lines_blank),0), COALESCE(SUM(size),0)
		FROM files WHERE repo_id = ?`, repoID).Row().Scan(&fileCount, &loc, &code, &comments, &blank, &size)
	if err != nil {
		return nil, fmt.Errorf("file totals: %w", err)
	}
	counts := map[string]any{}
	for col, model := range map[string]any{
		"commit_count":      &models.Commit{},
		"contributor_count": &models.Contributor{},
		"dependency_count":  &models.Dependency{},
		"dup_group_count":   &models.DuplicateGroup{},
	} {
		var n int64
		if err := r.db.Model(model).Where("repo_id = ?", repoID).Count(&n).Error; err != nil {
			return nil, fmt.Errorf("count %s: %w", col, err)
		}
		counts[col] = n
	}
	counts["file_count"] = fileCount
	counts["total_loc"] = loc
	counts["total_code"] = code
	counts["total_comments"] = comments
	counts["total_blank"] = blank
	counts["total_size"] = size
	return counts, nil
}
