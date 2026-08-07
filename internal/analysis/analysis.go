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
func (r *Runner) Run(ctx context.Context, repoID, jobID uint, rep jobs.Reporter, settings config.Settings) error {
	var repo models.Repository
	if err := r.db.First(&repo, repoID).Error; err != nil {
		return fmt.Errorf("load repo: %w", err)
	}
	if err := r.db.Model(&repo).Update("status", models.RepoScanning).Error; err != nil {
		return err
	}
	root := repo.Path
	if err := database.ClearRepoData(r.db, repoID); err != nil {
		return err
	}

	read := func(rel string) (string, error) {
		data, err := os.ReadFile(filepath.Join(root, rel))
		return string(data), err
	}

	stages := []struct {
		name string
		fn   func() error
	}{
		{"git metadata", func() error { return r.gitMeta(ctx, &repo) }},
		{"scanning files", func() error {
			return r.fileScan(ctx, &repo, settings, rep, 1, stageCount)
		}},
		{"git history", func() error { return r.gitHistory(ctx, &repo, rep, read) }},
		{"dependencies", func() error { return r.dependencies(ctx, &repo, read) }},
		{"import graph", func() error { return r.importGraph(ctx, &repo, read) }},
		{"duplicates", func() error { return r.duplicates(ctx, &repo, settings, read) }},
		{"content index", func() error { return r.contentIndex(ctx, &repo, read) }},
	}

	for i, st := range stages {
		if err := r.runStage(ctx, rep, i, len(stages), st.name, st.fn); err != nil {
			r.db.Model(&models.Repository{}).Where("id = ?", repo.ID).
				Updates(map[string]any{"status": models.RepoFailed, "updated_at": time.Now()})
			return err
		}
	}

	if err := r.updateSummary(repo.ID); err != nil {
		return err
	}
	now := time.Now()
	if err := r.db.Model(&models.Repository{}).Where("id = ?", repo.ID).Updates(map[string]any{
		"status":          models.RepoReady,
		"last_scanned_at": now,
		"updated_at":      now,
	}).Error; err != nil {
		return err
	}
	return nil
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
	if err := r.db.Model(&models.Repository{}).Where("id = ?", repo.ID).Updates(map[string]any{
		"git_remote": m.Remote, "head_commit": m.Head, "default_branch": m.DefaultBranch,
	}).Error; err != nil {
		return err
	}
	if err := r.db.Where("repo_id = ?", repo.ID).Delete(&models.Branch{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("repo_id = ?", repo.ID).Delete(&models.Tag{}).Error; err != nil {
		return err
	}
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
	var commits []models.Commit
	contrib, files, err := git.AnalyzeHistoryWithCommits(ctx, repo.Path, func(c models.Commit, changes []gitrepo.FileChange) {
		c.FilesChanged = len(changes)
		for _, fc := range changes {
			c.Insertions += fc.Add
			c.Deletions += fc.Del
		}
		commits = append(commits, c)
	})
	if err != nil {
		return err
	}

	// Persist commits in batches.
	if len(commits) > 0 {
		const batch = 500
		for i := 0; i < len(commits); i += batch {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			end := i + batch
			if end > len(commits) {
				end = len(commits)
			}
			chunk := commits[i:end]
			for j := range chunk {
				chunk[j].RepoID = repo.ID
			}
			if err := r.db.Create(&chunk).Error; err != nil {
				return err
			}
		}
	}

	// Contributors rollup.
	if err := r.db.Where("repo_id = ?", repo.ID).Delete(&models.Contributor{}).Error; err != nil {
		return err
	}
	if err := r.db.Where("repo_id = ?", repo.ID).Delete(&models.FileOwnership{}).Error; err != nil {
		return err
	}
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

	// File ownership + per-file git attribution.
	if len(files) > 0 {
		ownership := make([]models.FileOwnership, 0, len(files))
		for path, fh := range files {
			ownership = append(ownership, models.FileOwnership{
				RepoID: repo.ID, Path: path, Author: fh.Author, Commits: fh.Commits,
			})
		}
		for i := 0; i < len(ownership); i += 500 {
			end := i + 500
			if end > len(ownership) {
				end = len(ownership)
			}
			if err := r.db.Create(ownership[i:end]).Error; err != nil {
				return err
			}
		}
		if err := r.applyFileGitInfo(repo.ID, files); err != nil {
			return err
		}
	}
	rep.SetMessage("git history analyzed")
	return rep.Checkpoint(ctx)
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
	if err := r.db.Where("repo_id = ?", repo.ID).Delete(&models.ImportEdge{}).Error; err != nil {
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
		if err := tx.Create(&res.Groups).Error; err != nil {
			return err
		}
		return tx.Create(&res.Blocks).Error
	})
}

func (r *Runner) contentIndex(ctx context.Context, repo *models.Repository, read func(string) (string, error)) error {
	var files []models.File
	if err := r.db.Where("repo_id = ?", repo.ID).Find(&files).Error; err != nil {
		return err
	}
	return search.New(r.db).Reindex(ctx, repo.ID, repo.Path, files, read)
}

func (r *Runner) updateSummary(repoID uint) error {
	var (
		fileCount, loc, code, comments, blank int
		size                                  int64
	)
	err := r.db.Raw(`SELECT COUNT(*), COALESCE(SUM(lines_total),0), COALESCE(SUM(lines_code),0),
		COALESCE(SUM(lines_comment),0), COALESCE(SUM(lines_blank),0), COALESCE(SUM(size),0)
		FROM files WHERE repo_id = ?`, repoID).Row().Scan(&fileCount, &loc, &code, &comments, &blank, &size)
	if err != nil {
		return err
	}
	var commits, contributors, deps, dupGroups int64
	r.db.Model(&models.Commit{}).Where("repo_id = ?", repoID).Count(&commits)
	r.db.Model(&models.Contributor{}).Where("repo_id = ?", repoID).Count(&contributors)
	r.db.Model(&models.Dependency{}).Where("repo_id = ?", repoID).Count(&deps)
	r.db.Model(&models.DuplicateGroup{}).Where("repo_id = ?", repoID).Count(&dupGroups)
	return r.db.Model(&models.Repository{}).Where("id = ?", repoID).Updates(map[string]any{
		"file_count": fileCount, "total_loc": loc, "total_code": code,
		"total_comments": comments, "total_blank": blank, "total_size": size,
		"commit_count": commits, "contributor_count": contributors,
		"dependency_count": deps, "dup_group_count": dupGroups,
	}).Error
}
