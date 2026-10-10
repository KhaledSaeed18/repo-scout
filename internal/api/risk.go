package api

import (
	"net/http"

	"github.com/KhaledSaeed18/repo-scout/internal/risk"
)

// handleHotspots ranks files by complexity times how often they changed in
// the last `months` of history (0 for all of it).
func (s *Server) handleHotspots(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	months := queryInt(r, "months", 12, 240)
	limit := max(queryInt(r, "limit", 50, 200), 1)
	rep, err := risk.Hotspots(s.db, id, months, limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "hotspots: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

// handleKnowledge reports how main authorship of the code is spread across
// people, grouping folders `depth` levels deep.
func (s *Server) handleKnowledge(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	depth := max(queryInt(r, "depth", 2, 6), 1)
	inactive := max(queryInt(r, "inactiveMonths", 6, 120), 1)
	rep, err := risk.Knowledge(s.db, id, depth, inactive)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "knowledge: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
