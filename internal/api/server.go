// Package api exposes the REST + WebSocket surface of Repo Scout.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/jobs"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"github.com/KhaledSaeed18/repo-scout/internal/ws"
)

// Server wires handlers to a database and background job manager.
type Server struct {
	db       *gorm.DB
	jobs     *jobs.Manager
	hub      *ws.Hub
	settings *database.SettingsStore
}

// New builds the server.
func New(db *gorm.DB, mgr *jobs.Manager, hub *ws.Hub, settings *database.SettingsStore) *Server {
	return &Server{db: db, jobs: mgr, hub: hub, settings: settings}
}

// Router assembles the chi router with all routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Get("/api/health", s.handleHealth)
	r.Get("/api/ws", s.handleWS)

	r.Route("/api/repositories", func(r chi.Router) {
		r.Post("/", s.handleCreateRepo)
		r.Get("/", s.handleListRepos)
		r.Get("/{id}", s.handleGetRepo)
		r.Delete("/{id}", s.handleDeleteRepo)
		r.Get("/{id}/files", s.handleFiles)
		r.Get("/{id}/tree", s.handleTree)
		r.Get("/{id}/commits", s.handleCommits)
		r.Get("/{id}/contributors", s.handleContributors)
		r.Get("/{id}/largest-commits", s.handleLargestCommits)
		r.Get("/{id}/ownership", s.handleOwnership)
		r.Get("/{id}/branches", s.handleBranches)
		r.Get("/{id}/tags", s.handleTags)
		r.Get("/{id}/heatmap", s.handleHeatmap)
		r.Get("/{id}/dependencies", s.handleDependencies)
		r.Get("/{id}/duplicates", s.handleDuplicates)
		r.Get("/{id}/architecture", s.handleArchitecture)
		r.Get("/{id}/metrics", s.handleMetrics)
		r.Get("/{id}/svg", s.handleSVG)
		r.Get("/{id}/export", s.handleExport)
	})

	r.Route("/api/search", func(r chi.Router) {
		r.Get("/", s.handleSearch)
	})

	r.Route("/api/jobs", func(r chi.Router) {
		r.Get("/", s.handleListJobs)
		r.Post("/{id}/pause", s.handlePauseJob)
		r.Post("/{id}/resume", s.handleResumeJob)
		r.Post("/{id}/cancel", s.handleCancelJob)
	})

	r.Route("/api/settings", func(r chi.Router) {
		r.Get("/", s.handleGetSettings)
		r.Put("/", s.handlePutSettings)
	})

	return r
}

type errorBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, errorBody{Error: msg})
}

func parseID(w http.ResponseWriter, r *http.Request) (uint, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return uint(id), true
}

func (s *Server) loadRepo(w http.ResponseWriter, id uint) (*models.Repository, bool) {
	var repo models.Repository
	if err := s.db.First(&repo, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			writeErr(w, http.StatusNotFound, "repository not found")
		} else {
			writeErr(w, http.StatusInternalServerError, fmt.Sprintf("load repo: %v", err))
		}
		return nil, false
	}
	return &repo, true
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	s.hub.HandleUpgrade(w, r)
}

// currentSettings loads the effective scanning settings.
func (s *Server) currentSettings() config.Settings {
	st, err := s.settings.Load()
	if err != nil {
		return config.Defaults()
	}
	return st
}
