package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/KhaledSaeed18/repo-scout/internal/analysis"
	"github.com/KhaledSaeed18/repo-scout/internal/api"
	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/jobs"
	"github.com/KhaledSaeed18/repo-scout/internal/webui"
	"github.com/KhaledSaeed18/repo-scout/internal/ws"
)

// serve runs the API, the scan workers and, in embedded builds, the
// interface, until interrupted.
func serve(args []string, stderr io.Writer) int {
	cfg := config.FromEnv()
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "address to listen on (REPO_SCOUT_ADDR)")
	fs.StringVar(&cfg.DBPath, "db", cfg.DBPath, "SQLite database file (REPO_SCOUT_DB)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() > 0 {
		_, _ = fmt.Fprintf(stderr, "serve takes no arguments, got %q\n", fs.Args())
		return exitUsage
	}
	logger, err := newLogger(stderr, cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return exitUsage
	}
	// The default logger also carries database warnings (see database.Open).
	slog.SetDefault(logger)
	fail := func(msg string, err error) int {
		logger.Error(msg, "err", err)
		return exitError
	}

	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fail("create data folder", err)
		}
	}
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return fail("open database", err)
	}
	if err := database.Migrate(db); err != nil {
		return fail("migrate database", err)
	}
	// Scans that stopped with the process left staged rows behind; they are
	// re-queued and start over, so the partial results are dropped.
	if err := database.ClearStagingData(db); err != nil {
		return fail("clear unfinished scans", err)
	}

	settings := database.NewSettingsStore(db)
	hub := ws.New()

	loadSettings := func() config.Settings {
		st, err := settings.Load()
		if err != nil {
			return config.Defaults()
		}
		return st
	}

	runner := analysis.New(db)
	mgr := jobs.New(db, runner, loadSettings, hub, logger)

	assets, err := webui.Assets()
	if err != nil {
		return fail("load interface", err)
	}
	server := api.New(api.Deps{DB: db, Jobs: mgr, Hub: hub, Settings: settings, AllowedHosts: cfg.AllowedHosts, UI: assets, Logger: logger})
	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: server.Router(),
		// Bounds how long a client may take to send headers (slowloris).
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		logger.Info("repo-scout listening", "version", version, "url", "http://"+cfg.Addr,
			"database", cfg.DBPath, "interface", assets != nil)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "err", err)
		}
		close(done)
	}()

	workerErr := make(chan error, 1)
	go func() {
		workerErr <- mgr.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	case err := <-workerErr:
		if err != nil {
			return fail("worker pool", err)
		}
	case <-done:
	}

	// Stop the worker pool and wait briefly for in-flight jobs to check in.
	stop()
	select {
	case err := <-workerErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("worker pool exit", "err", err)
		}
	case <-time.After(3 * time.Second):
	}
	return exitOK
}
