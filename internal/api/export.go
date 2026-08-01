package api

import (
	"net/http"

	"github.com/KhaledSaeed18/repo-scout/internal/export"
)

// handleExport streams repository data as CSV or JSON.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadRepo(w, id); !ok {
		return
	}
	kind := export.Kind(r.URL.Query().Get("kind"))
	format := export.Format(r.URL.Query().Get("format"))
	if format != export.FormatCSV && format != export.FormatJSON {
		format = export.FormatCSV
	}
	switch kind {
	case export.KindFiles, export.KindCommits, export.KindContributors:
	default:
		writeErr(w, http.StatusBadRequest, "unsupported export kind")
		return
	}

	ext := "json"
	if format == export.FormatCSV {
		ext = "csv"
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename="+string(kind)+"."+ext)
	if err := export.Export(s.db, id, export.Params{Kind: kind, Format: format}, w); err != nil {
		writeErr(w, http.StatusInternalServerError, "export: "+err.Error())
		return
	}
}
