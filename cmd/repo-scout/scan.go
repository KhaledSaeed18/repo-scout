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
	failOn := fs.String("fail-on", "", "comma-separated findings that fail the run: cycles, hidden-coupling")
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
	gates := report.Gates{MaxComplexity: *maxComplexity, MaxDuplicates: *maxDuplicates}
	for _, name := range strings.Split(*failOn, ",") {
		switch strings.TrimSpace(name) {
		case "":
		case "cycles":
			gates.NoCycles = true
		case "hidden-coupling":
			gates.NoHiddenCoupling = true
		default:
			_, _ = fmt.Fprintf(stderr, "unknown --fail-on value %q: use cycles or hidden-coupling\n", name)
			return exitUsage
		}
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
	rep, err := analyze(ctx, root, report.Options{Version: version, Top: *top, Gates: gates}, &progress{w: stderr, quiet: *quiet})
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

// analyze scans root into a temporary database and builds the report.
func analyze(ctx context.Context, root string, opts report.Options, rep *progress) (report.Report, error) {
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
	repo := models.Repository{Name: filepath.Base(root), Path: root}
	if err := db.Create(&repo).Error; err != nil {
		return report.Report{}, fmt.Errorf("create repository: %w", err)
	}
	if err := analysis.New(db).Run(ctx, repo.ID, 0, rep, config.Defaults()); err != nil {
		return report.Report{}, err
	}
	rep.done()
	return report.Build(db, repo.ID, opts)
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

func (p *progress) done() {
	if !p.quiet {
		_, _ = fmt.Fprintln(p.w, "  done")
	}
}
