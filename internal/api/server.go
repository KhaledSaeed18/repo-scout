// Package api exposes the REST + WebSocket surface of Repo Scout.
package api

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	db           *gorm.DB
	jobs         *jobs.Manager
	hub          *ws.Hub
	settings     *database.SettingsStore
	allowedHosts map[string]bool
	ui           fs.FS
	log          *slog.Logger
}

// Deps is what the server needs.
type Deps struct {
	DB       *gorm.DB
	Jobs     *jobs.Manager
	Hub      *ws.Hub
	Settings *database.SettingsStore
	// AllowedHosts are the Host names the server answers to (ports ignored).
	AllowedHosts []string
	// UI is the built frontend to serve, or nil to serve the API alone.
	UI fs.FS
	// Logger receives one line per request; nil logs nothing.
	Logger *slog.Logger
}

// New builds the server.
func New(d Deps) *Server {
	hosts := make(map[string]bool, len(d.AllowedHosts))
	for _, h := range d.AllowedHosts {
		hosts[strings.ToLower(strings.Trim(h, "[]"))] = true
	}
	logger := d.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Server{db: d.DB, jobs: d.Jobs, hub: d.Hub, settings: d.Settings, allowedHosts: hosts, ui: d.UI, log: logger}
}

// Router assembles the chi router with all routes.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(s.allowHosts)
	r.Use(securityHeaders)
	r.Use(middleware.GetHead)
	r.Use(middleware.RequestID)
	r.Use(s.logRequests)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(requireJSONWrites)

	r.Get("/api/health", s.handleHealth)
	r.Get("/api/openapi.yaml", s.handleOpenAPI)
	r.Get("/api/ws", s.handleWS)
	r.Get("/api/browse", s.handleBrowse)
	r.Get("/api/portfolio", s.handlePortfolio)

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
		r.Get("/{id}/hotspots", s.handleHotspots)
		r.Get("/{id}/knowledge", s.handleKnowledge)
		r.Get("/{id}/coupling", s.handleCoupling)
		r.Get("/{id}/trends", s.handleTrends)
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

	if s.ui != nil {
		r.Get("/*", spa(s.ui))
	}
	return r
}

// logRequests writes one structured line per request. API calls log at info,
// interface files and health checks at debug, and server errors at error.
func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case r.URL.Path == "/api/health" || !strings.HasPrefix(r.URL.Path, "/api/"):
			level = slog.LevelDebug
		}
		s.log.LogAttrs(r.Context(), level, "request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.Int("bytes", ww.BytesWritten()),
			slog.String("took", time.Since(start).Round(time.Microsecond).String()),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)
	})
}

// allowHosts refuses requests addressed to an unknown host name. Without it a
// website could rebind its domain to 127.0.0.1 and read the API from the
// browser as a same-origin page.
func (s *Server) allowHosts(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if !s.allowedHosts[strings.ToLower(strings.Trim(host, "[]"))] {
			writeErr(w, http.StatusForbidden, "host not allowed: add it to REPO_SCOUT_ALLOWED_HOSTS to reach the API by this name")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireJSONWrites rejects body-carrying writes that are not JSON. Browsers
// send form and text/plain POSTs cross-site without a CORS preflight, so this
// keeps other websites from driving the local API.
func requireJSONWrites(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || mt != "application/json" {
				writeErr(w, http.StatusUnsupportedMediaType, "content type must be application/json")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
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
