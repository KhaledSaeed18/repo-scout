package api

import (
	"net/http"

	"github.com/KhaledSaeed18/repo-scout/internal/coupling"
)

// handleCoupling lists files that change together, strongest first. path
// limits it to one file's partners; hidden=true keeps only pairs with no
// import between them.
func (s *Server) handleCoupling(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	q := coupling.Query{
		Path:   r.URL.Query().Get("path"),
		Hidden: r.URL.Query().Get("hidden") == "true",
		Limit:  max(queryInt(r, "limit", 100, 500), 1),
	}
	pairs, err := coupling.List(s.db, id, q)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "coupling: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"pairs": pairs})
}
