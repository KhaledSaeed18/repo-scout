// Package gitrepo wraps the git CLI for repository metadata and history
// analysis. Using the native git binary keeps parsing robust on very large
// repositories and produces output identical to what developers see.
package gitrepo

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Meta is top-level repository metadata.
type Meta struct {
	Remote        string `json:"remote"`
	Head          string `json:"head"`
	DefaultBranch string `json:"defaultBranch"`
	BranchCount   int    `json:"branchCount"`
	TagCount      int    `json:"tagCount"`
}

// FileChange is one file touched by a commit.
type FileChange struct {
	Path string
	Add  int
	Del  int
}

// FileHistory is the rolled-up git history of one file.
type FileHistory struct {
	Author  string    `json:"author"`
	First   time.Time `json:"first"`
	Last    time.Time `json:"last"`
	Commits int       `json:"commits"`
}

// ContributorStats is the rolled-up git activity of one author.
type ContributorStats struct {
	Name          string
	Email         string
	Commits       int
	Insertions    int
	Deletions     int
	FirstCommitAt time.Time
	LastCommitAt  time.Time
}

// Analyzer runs git commands against a repository.
type Analyzer struct{}

// New builds an Analyzer.
func New() *Analyzer { return &Analyzer{} }

// Available reports whether the git binary is on PATH.
func (a *Analyzer) Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// IsRepo reports whether root contains a git repository.
func (a *Analyzer) IsRepo(root string) bool {
	err := exec.Command("git", "-C", root, "rev-parse", "--git-dir").Run()
	return err == nil
}

// Meta returns top-level repository metadata.
func (a *Analyzer) Meta(ctx context.Context, root string) (Meta, error) {
	m := Meta{}
	remote, err := a.output(ctx, root, "remote", "get-url", "origin")
	if err == nil {
		m.Remote = strings.TrimSpace(remote)
	}
	head, err := a.output(ctx, root, "rev-parse", "HEAD")
	if err == nil {
		m.Head = strings.TrimSpace(head)
	}
	branch, err := a.output(ctx, root, "symbolic-ref", "--short", "HEAD")
	if err == nil {
		m.DefaultBranch = strings.TrimSpace(branch)
	}
	if out, err := a.output(ctx, root, "branch", "--list"); err == nil {
		m.BranchCount = countNonEmptyLines(out)
	}
	if out, err := a.output(ctx, root, "tag", "--list"); err == nil {
		m.TagCount = countNonEmptyLines(out)
	}
	return m, nil
}

// Branches lists branch names and their commit hashes.
func (a *Analyzer) Branches(ctx context.Context, root, current string) ([]models.Branch, error) {
	out, err := a.output(ctx, root, "for-each-ref", "--format=%(refname:short)%09%(objectname)", "refs/heads")
	if err != nil {
		return nil, fmt.Errorf("branches: %w", err)
	}
	var branches []models.Branch
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		branches = append(branches, models.Branch{Name: parts[0], CommitHash: parts[1], IsCurrent: parts[0] == current})
	}
	return branches, nil
}

// Tags lists tag names and their commit hashes.
func (a *Analyzer) Tags(ctx context.Context, root string) ([]models.Tag, error) {
	out, err := a.output(ctx, root, "for-each-ref", "--format=%(refname:short)%09%(objectname)", "refs/tags")
	if err != nil {
		return nil, fmt.Errorf("tags: %w", err)
	}
	var tags []models.Tag
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		tags = append(tags, models.Tag{Name: parts[0], CommitHash: parts[1]})
	}
	return tags, nil
}

// StreamLogs streams the full commit history across all refs. For each commit
// it invokes fn with the commit and the files it changed. The callbacks run
// in a single goroutine in history order.
func (a *Analyzer) StreamLogs(ctx context.Context, root string, fn func(models.Commit, []FileChange)) error {
	args := []string{
		"log", "--all", "--numstat", "--date-order",
		"--pretty=format:%x1e%H%x1f%an%x1f%ae%x1f%at%x1f%P%x1f%s%x1e",
	}
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe git log: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start git log: %w", err)
	}

	var commit models.Commit
	var files []FileChange
	flush := func() {
		if commit.Hash != "" {
			fn(commit, files)
			commit = models.Commit{}
			files = nil
		}
	}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "\x1e") {
			flush()
			parseHeader(line, &commit)
			continue
		}
		if fc, ok := parseFileChange(line); ok {
			files = append(files, fc)
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return fmt.Errorf("read git log: %w", err)
	}
	return cmd.Wait()
}

func parseHeader(line string, c *models.Commit) {
	h := strings.TrimSuffix(line, "\x1e")
	fields := strings.Split(h[1:], "\x1f")
	if len(fields) < 5 {
		return
	}
	ts, _ := strconv.ParseInt(fields[3], 10, 64)
	parents := strings.Fields(fields[4])
	c.Hash = fields[0]
	c.Author = fields[1]
	c.Email = fields[2]
	c.Date = time.Unix(ts, 0).UTC()
	c.IsMerge = len(parents) > 1
	c.Message = fields[5]
}

func parseFileChange(line string) (FileChange, bool) {
	parts := strings.Split(line, "\t")
	if len(parts) < 3 {
		return FileChange{}, false
	}
	add, err1 := strconv.Atoi(parts[0])
	del, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return FileChange{}, false
	}
	return FileChange{Path: parts[2], Add: add, Del: del}, true
}

// AnalyzeHistory streams the whole history and rolls up contributor and
// per-file statistics.
func (a *Analyzer) AnalyzeHistory(ctx context.Context, root string) (map[string]*ContributorStats, map[string]*FileHistory, error) {
	contrib := map[string]*ContributorStats{}
	files := map[string]*FileHistory{}

	err := a.StreamLogs(ctx, root, func(c models.Commit, changes []FileChange) {
		if c.Author == "" {
			c.Author = c.Email
		}
		key := c.Email
		if key == "" {
			key = c.Author
		}
		cs := contrib[key]
		if cs == nil {
			cs = &ContributorStats{Name: c.Author, Email: c.Email}
			contrib[key] = cs
		}
		cs.Commits++
		if cs.FirstCommitAt.IsZero() || c.Date.Before(cs.FirstCommitAt) {
			cs.FirstCommitAt = c.Date
		}
		if c.Date.After(cs.LastCommitAt) {
			cs.LastCommitAt = c.Date
		}
		for _, fc := range changes {
			cs.Insertions += fc.Add
			cs.Deletions += fc.Del
			fh := files[fc.Path]
			if fh == nil {
				fh = &FileHistory{}
				files[fc.Path] = fh
			}
			fh.Commits++
			if fh.First.IsZero() || c.Date.Before(fh.First) {
				fh.First = c.Date
			}
			if c.Date.After(fh.Last) {
				fh.Last = c.Date
				fh.Author = c.Author
			}
		}
	})
	if err != nil {
		return nil, nil, err
	}
	return contrib, files, nil
}

func (a *Analyzer) output(ctx context.Context, root string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func countNonEmptyLines(s string) int {
	n := 0
	for _, l := range strings.Split(s, "\n") {
		if strings.TrimSpace(l) != "" {
			n++
		}
	}
	return n
}
