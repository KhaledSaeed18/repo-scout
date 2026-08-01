package api

import (
	"net/http"
)

// handleListJobs lists recent jobs, newest first.
func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.jobs.List(queryInt(r, "limit", 100, 1000))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list jobs: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) handlePauseJob(w http.ResponseWriter, r *http.Request) {
	s.jobAction(w, r, s.jobs.Pause)
}

func (s *Server) handleResumeJob(w http.ResponseWriter, r *http.Request) {
	s.jobAction(w, r, s.jobs.Resume)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	s.jobAction(w, r, s.jobs.Cancel)
}

type jobActionFunc func(jobID uint) error

func (s *Server) jobAction(w http.ResponseWriter, r *http.Request, fn jobActionFunc) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := fn(id); err != nil {
		writeErr(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
