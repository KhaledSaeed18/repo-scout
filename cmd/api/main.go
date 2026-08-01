// Command api runs the Repo Scout HTTP server.
package main

import (
	"context"
	"errors"
	"log"
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
	"github.com/KhaledSaeed18/repo-scout/internal/ws"
)

func main() {
	cfg := config.FromEnv()

	if dir := filepath.Dir(cfg.DBPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("create data dir: %v", err)
		}
	}
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
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
	mgr := jobs.New(db, runner, loadSettings, hub)

	server := api.New(db, mgr, hub, settings)
	srv := &http.Server{
		Addr:    cfg.Addr,
		Handler: server.Router(),
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	go func() {
		log.Printf("repo-scout listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("server error: %v", err)
		}
		close(done)
	}()

	workerErr := make(chan error, 1)
	go func() {
		workerErr <- mgr.Start(ctx)
	}()

	select {
	case <-ctx.Done():
		log.Printf("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	case err := <-workerErr:
		if err != nil {
			log.Fatalf("worker pool: %v", err)
		}
	case <-done:
	}

	// Stop the worker pool and wait briefly for in-flight jobs to check in.
	stop()
	select {
	case err := <-workerErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("worker pool exit: %v", err)
		}
	case <-time.After(3 * time.Second):
	}
}
