package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/gitrepo"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/report"
)

// scan analyzes one repository into a throwaway database and writes a
// report. It exits non-zero when a quality gate fails, so CI can block on it.
func scan(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		_, _ = fmt.Fprint(stderr, "Usage: repo-scout scan [flags] [path]\n\nScans path (default: the current folder) and prints a report.\nExit codes: 0 passed, 1 a quality gate failed, 2 bad usage, 3 the scan failed.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	format := fs.String("format", "text", "report format: text, json or sarif")
	output := fs.String("output", "", "write the report to this file instead of standard output")
	top := fs.Int("top", 10, "how many hotspots, duplicates and hidden dependencies to list")
	maxComplexity := fs.Int("max-complexity", 0, "fail when any file's complexity is above this (0: off)")
	maxDuplicates := fs.Int("max-duplicates", 0, "fail when there are more duplicate groups than this (0: off)")
	failOn := fs.String("fail-on", "", "comma-separated findings that fail the run: cycles, hidden-coupling, new-cycles")
	base := fs.String("base", "", "compare against this ref, such as origin/main (path must be a Git repository)")
	maxGrowth := fs.Int("max-complexity-increase", 0, "with --base, fail when total complexity grew by more than this (0: off)")
	quiet := fs.Bool("quiet", false, "do not print progress to standard error")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() > 1 {
		_, _ = fmt.Fprintln(stderr, "scan takes one path")
		return exitUsage
	}
	write, ok := writers()[*format]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "unknown format %q: use text, json or sarif\n", *format)
		return exitUsage
	}
	gates := report.Gates{MaxComplexity: *maxComplexity, MaxDuplicates: *maxDuplicates, MaxComplexityIncrease: *maxGrowth}
	for _, name := range strings.Split(*failOn, ",") {
		switch strings.TrimSpace(name) {
		case "":
		case "cycles":
			gates.NoCycles = true
		case "hidden-coupling":
			gates.NoHiddenCoupling = true
		case "new-cycles":
			gates.NoNewCycles = true
		default:
			_, _ = fmt.Fprintf(stderr, "unknown --fail-on value %q: use cycles, hidden-coupling or new-cycles\n", name)
			return exitUsage
		}
	}
	if *base == "" && (gates.NoNewCycles || gates.MaxComplexityIncrease > 0) {
		_, _ = fmt.Fprintln(stderr, "--fail-on new-cycles and --max-complexity-increase need --base")
		return exitUsage
	}
	target := "."
	if fs.NArg() == 1 {
		target = fs.Arg(0)
	}
	root, err := filepath.Abs(target)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "resolve %s: %v\n", target, err)
		return exitUsage
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		_, _ = fmt.Fprintf(stderr, "%s is not a folder\n", root)
		return exitUsage
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	rep, err := analyze(ctx, root, *base, report.Options{Version: version, Top: *top, Gates: gates}, &progress{w: stderr, quiet: *quiet})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "scan failed: %v\n", err)
		return exitError
	}

	out := stdout
	if *output != "" {
		f, err := os.Create(*output)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "create %s: %v\n", *output, err)
			return exitError
		}
		defer func() { _ = f.Close() }()
		out = f
	}
	if err := write(out, rep); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return exitError
	}
	if !rep.Passed() {
		return exitGateFailed
	}
	return exitOK
}

func writers() map[string]func(io.Writer, report.Report) error {
	return map[string]func(io.Writer, report.Report) error{
		"text": report.WriteText, "json": report.WriteJSON, "sarif": report.WriteSARIF,
	}
}

// analyze scans root into a temporary database and builds the report. With
// a base ref it also scans the tree of that ref, exported read-only next to
// the database, and compares the two.
func analyze(ctx context.Context, root, baseRef string, opts report.Options, rep *progress) (report.Report, error) {
	dir, err := os.MkdirTemp("", "repo-scout-")
	if err != nil {
		return report.Report{}, fmt.Errorf("create temp folder: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	db, err := database.Open(filepath.Join(dir, "scan.db"))
	if err != nil {
		return report.Report{}, err
	}
	if sqlDB, err := db.DB(); err == nil {
		defer func() { _ = sqlDB.Close() }()
	}
	if err := database.Migrate(db); err != nil {
		return report.Report{}, err
	}
	scanOne := func(name, path string) (uint, error) {
		repo := models.Repository{Name: name, Path: path}
		if err := db.Create(&repo).Error; err != nil {
			return 0, fmt.Errorf("create repository: %w", err)
		}
		return repo.ID, analysis.New(db).Run(ctx, repo.ID, 0, rep, config.Defaults())
	}

	if baseRef != "" {
		git := gitrepo.New()
		if !git.IsRepo(root) {
			return report.Report{}, fmt.Errorf("--base needs a Git repository, and %s is not one", root)
		}
		commit, err := git.ResolveCommit(ctx, root, baseRef)
		if err != nil {
			return report.Report{}, err
		}
		tree := filepath.Join(dir, "base")
		if err := os.Mkdir(tree, 0o700); err != nil {
			return report.Report{}, err
		}
		rep.section("base " + baseRef)
		if err := git.Export(ctx, root, commit, tree); err != nil {
			return report.Report{}, err
		}
		baseID, err := scanOne(filepath.Base(root)+"@"+baseRef, tree)
		if err != nil {
			return report.Report{}, fmt.Errorf("scan %s: %w", baseRef, err)
		}
		opts.Base = &report.Base{RepoID: baseID, Ref: baseRef, Commit: commit}
		rep.section("current code")
	}

	headID, err := scanOne(filepath.Base(root), root)
	if err != nil {
		return report.Report{}, err
	}
	rep.done()
	return report.Build(db, headID, opts)
}

// progress prints each scan stage once to standard error. Stages report
// from several goroutines, so it locks.
type progress struct {
	w     io.Writer
	quiet bool
	mu    sync.Mutex
	stage string
}

func (p *progress) SetTotal(int)        {}
func (p *progress) SetProgress(float64) {}
func (p *progress) Inc(int)             {}

func (p *progress) SetMessage(msg string) {
	if p.quiet {
		return
	}
	// "scanning files (12/300)" and "scanning files" are one stage.
	stage, _, _ := strings.Cut(msg, " (")
	p.mu.Lock()
	defer p.mu.Unlock()
	if stage != p.stage {
		p.stage = stage
		_, _ = fmt.Fprintf(p.w, "  %s\n", stage)
	}
}

func (p *progress) Checkpoint(ctx context.Context) error { return ctx.Err() }

// section prints a heading before the stages of the next scan.
func (p *progress) section(name string) {
	if p.quiet {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stage = ""
	_, _ = fmt.Fprintf(p.w, "%s\n", name)
}

func (p *progress) done() {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.w, "  done")
	}
}
