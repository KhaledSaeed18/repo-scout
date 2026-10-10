// Package gitrepo wraps the git CLI for repository metadata and history
// analysis. Using the native git binary keeps parsing robust on very large
// repositories and produces output identical to what developers see.
package gitrepo

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
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

// FileChange is one file touched by a commit. OldPath is set when the commit
// renamed the file. Binary files report zero added and deleted lines.
type FileChange struct {
	Path    string
	OldPath string
	Add     int
	Del     int
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

// logFormat is one header line per commit: hash, mapped author name and
// email (%aN and %aE apply .mailmap, so one person committing under several
// names counts once), strict ISO date with the author's offset, parents and
// subject, framed by record separators.
const logFormat = "--pretty=format:%x1e%H%x1f%aN%x1f%aE%x1f%aI%x1f%P%x1f%s%x1e"

// historyRefs selects the project's history: branches, tags, remotes and
// HEAD. Other refs (stashes, notes, tool checkpoints) are not part of it.
func (a *Analyzer) historyRefs(ctx context.Context, root string) []string {
	refs := []string{"--branches", "--tags", "--remotes"}
	// HEAD may be detached; include it only when it resolves so empty
	// repositories do not fail.
	if _, err := a.output(ctx, root, "rev-parse", "--verify", "--quiet", "HEAD"); err == nil {
		refs = append(refs, "HEAD")
	}
	return refs
}

// StreamLogs streams the commit history reachable from branches, tags,
// remotes, and HEAD. For each commit it invokes fn with the commit and the
// files it changed. The callbacks run in a single goroutine in history order;
// an error from fn stops the stream and is returned.
func (a *Analyzer) StreamLogs(ctx context.Context, root string, fn func(models.Commit, []FileChange) error) error {
	args := append([]string{"log", "--numstat", "--date-order", logFormat}, a.historyRefs(ctx, root)...)
	return a.runLog(ctx, root, args, nil, fn)
}

// runLog runs a git log with logFormat and calls fn for each commit with the
// file changes listed under it (none unless --numstat is passed). stdin, when
// set, feeds git's --stdin.
func (a *Analyzer) runLog(ctx context.Context, root string, args []string, stdin io.Reader, fn func(models.Commit, []FileChange) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	cmd.Stdin = stdin
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe git log: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start git log: %w", err)
	}

	var commit models.Commit
	var files []FileChange
	flush := func() error {
		if commit.Hash == "" {
			return nil
		}
		err := fn(commit, files)
		commit = models.Commit{}
		files = nil
		return err
	}
	// abort stops git and reaps it before reporting a callback error.
	abort := func(err error) error {
		cancel()
		_ = cmd.Wait()
		return err
	}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "\x1e") {
			if err := flush(); err != nil {
				return abort(err)
			}
			parseHeader(line, &commit)
			continue
		}
		if fc, ok := parseFileChange(line); ok {
			files = append(files, fc)
		}
	}
	if err := sc.Err(); err != nil {
		return abort(fmt.Errorf("read git log: %w", err))
	}
	if err := flush(); err != nil {
		return abort(err)
	}
	return cmd.Wait()
}

// ChangeCache supplies the file changes of commits an earlier scan analyzed,
// so a rescan only asks git to diff the commits it has not seen.
type ChangeCache interface {
	// Has reports whether the commit was analyzed before.
	Has(hash string) (bool, error)
	// Changes returns the files it changed, as recorded then.
	Changes(hash string) ([]FileChange, error)
}

// maxFreshCommits is how many unseen commits a cached stream holds in memory
// before a single full pass is the better choice.
const maxFreshCommits = 20000

// streamCached streams the same history as StreamLogs, taking the changes of
// known commits from cache and diffing only the rest. The commit list always
// comes from the current refs, so rewritten history is handled: commits no
// longer reachable simply do not appear.
func (a *Analyzer) streamCached(ctx context.Context, root string, cache ChangeCache, fn func(models.Commit, []FileChange) error) error {
	refs := a.historyRefs(ctx, root)
	headers := append([]string{"log", "--date-order", logFormat}, refs...)

	// Pass 1: find the commits the cache has not seen.
	var fresh []string
	err := a.runLog(ctx, root, headers, nil, func(c models.Commit, _ []FileChange) error {
		known, err := cache.Has(c.Hash)
		if err != nil {
			return err
		}
		if !known {
			fresh = append(fresh, c.Hash)
			if len(fresh) > maxFreshCommits {
				return errTooFresh
			}
		}
		return nil
	})
	if errors.Is(err, errTooFresh) {
		return a.StreamLogs(ctx, root, fn)
	}
	if err != nil {
		return err
	}

	diffs, err := a.changesOf(ctx, root, fresh)
	if err != nil {
		return err
	}

	// Pass 2: replay history with every commit's changes.
	return a.runLog(ctx, root, headers, nil, func(c models.Commit, _ []FileChange) error {
		changes, ok := diffs[c.Hash]
		switch {
		case ok:
			delete(diffs, c.Hash)
		default:
			known, err := cache.Has(c.Hash)
			if err != nil {
				return err
			}
			if known {
				if changes, err = cache.Changes(c.Hash); err != nil {
					return err
				}
				break
			}
			// Committed between the two passes.
			late, err := a.changesOf(ctx, root, []string{c.Hash})
			if err != nil {
				return err
			}
			changes = late[c.Hash]
		}
		return fn(c, changes)
	})
}

var errTooFresh = errors.New("too many commits to cache")

// changesOf diffs exactly the given commits.
func (a *Analyzer) changesOf(ctx context.Context, root string, hashes []string) (map[string][]FileChange, error) {
	out := make(map[string][]FileChange, len(hashes))
	if len(hashes) == 0 {
		return out, nil
	}
	args := []string{"log", "--no-walk=unsorted", "--stdin", "--numstat", logFormat}
	stdin := strings.NewReader(strings.Join(hashes, "\n") + "\n")
	err := a.runLog(ctx, root, args, stdin, func(c models.Commit, files []FileChange) error {
		out[c.Hash] = files
		return nil
	})
	return out, err
}

func parseHeader(line string, c *models.Commit) {
	h := strings.TrimSuffix(line, "\x1e")
	fields := strings.Split(h[1:], "\x1f")
	if len(fields) < 5 {
		return
	}
	// %aI is strict ISO 8601 with the author's offset, e.g. 2024-01-01T23:30:00+03:00.
	authored, _ := time.Parse(time.RFC3339, fields[3])
	_, offset := authored.Zone()
	parents := strings.Fields(fields[4])
	c.Hash = fields[0]
	c.Author = fields[1]
	c.Email = fields[2]
	c.Date = authored.UTC()
	c.TZOffset = offset / 60
	c.IsMerge = len(parents) > 1
	c.Message = fields[5]
}

func parseFileChange(line string) (FileChange, bool) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) < 3 {
		return FileChange{}, false
	}
	fc := FileChange{}
	// Binary files show "-" instead of line counts.
	if parts[0] != "-" || parts[1] != "-" {
		add, err1 := strconv.Atoi(parts[0])
		del, err2 := strconv.Atoi(parts[1])
		if err1 != nil || err2 != nil {
			return FileChange{}, false
		}
		fc.Add, fc.Del = add, del
	}
	fc.OldPath, fc.Path = splitRename(parts[2])
	return fc, true
}

// splitRename reads a numstat path, which for renames is either
// "old => new" or "dir/{old => new}/file" with either side possibly empty.
// It returns the old path (empty when not a rename) and the new path.
func splitRename(p string) (oldPath, newPath string) {
	if open := strings.Index(p, "{"); open >= 0 {
		if end := strings.Index(p[open:], "}"); end > 0 {
			inner := p[open+1 : open+end]
			if from, to, ok := strings.Cut(inner, " => "); ok {
				prefix, suffix := p[:open], p[open+end+1:]
				join := func(mid string) string {
					// An empty side leaves a doubled slash: "a/{ => b}/c" is "a/c".
					return strings.ReplaceAll(prefix+mid+suffix, "//", "/")
				}
				return join(from), join(to)
			}
		}
	}
	if from, to, ok := strings.Cut(p, " => "); ok {
		return from, to
	}
	return "", p
}

// AnalyzeHistory streams the whole history and rolls up contributor and
// per-file statistics.
func (a *Analyzer) AnalyzeHistory(ctx context.Context, root string) (map[string]*ContributorStats, map[string]*FileHistory, error) {
	contrib, files, err := a.AnalyzeHistoryWithCommits(ctx, root, nil, nil)
	return contrib, files, err
}

// AnalyzeHistoryWithCommits is like AnalyzeHistory but also invokes onCommit
// for every commit and its file changes as they stream, so callers can persist
// commits in one pass. An error from onCommit stops the stream. With a cache,
// only commits it has not seen are diffed (see ChangeCache).
func (a *Analyzer) AnalyzeHistoryWithCommits(ctx context.Context, root string, cache ChangeCache, onCommit func(models.Commit, []FileChange) error) (map[string]*ContributorStats, map[string]*FileHistory, error) {
	contrib := map[string]*ContributorStats{}
	files := map[string]*FileHistory{}
	// renamedTo maps a path to the name it was later renamed to. History
	// streams newest first, so a rename is seen before the older commits
	// that touched the file under its earlier name.
	renamedTo := map[string]string{}
	resolve := func(p string) string {
		for range 64 { // bounded in case a history renames in a loop
			next, ok := renamedTo[p]
			if !ok || next == p {
				return p
			}
			p = next
		}
		return p
	}

	stream := a.StreamLogs
	if cache != nil {
		stream = func(ctx context.Context, root string, fn func(models.Commit, []FileChange) error) error {
			return a.streamCached(ctx, root, cache, fn)
		}
	}
	err := stream(ctx, root, func(c models.Commit, changes []FileChange) error {
		if c.Author == "" {
			c.Author = c.Email
		}
		for i, fc := range changes {
			changes[i].Path = resolve(fc.Path)
			if fc.OldPath == "" || fc.OldPath == changes[i].Path {
				continue
			}
			if _, seen := renamedTo[fc.OldPath]; !seen {
				renamedTo[fc.OldPath] = changes[i].Path
			}
		}
		if onCommit != nil {
			if err := onCommit(c, changes); err != nil {
				return err
			}
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
		return nil
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
