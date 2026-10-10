package api

import (
	"net/http"

	"github.com/KhaledSaeed18/repo-scout/internal/portfolio"
)

// handlePortfolio rolls every scanned repository up into one view.
func (s *Server) handlePortfolio(w http.ResponseWriter, _ *http.Request) {
	p, err := portfolio.Build(s.db)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "portfolio: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}
