package api

import (
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/export"
)

// handleExport streams repository data as CSV or JSON.
func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	repo, ok := s.loadRepo(w, id)
	if !ok {
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

	ext, contentType := "json", "application/json"
	if format == export.FormatCSV {
		ext, contentType = "csv", "text/csv; charset=utf-8"
	}
	// Prefix with the repository name so exports of different repos don't collide.
	name := fmt.Sprintf("%s-%s.%s", safeFilename(repo.Name), kind, ext)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
	if err := export.Export(s.db, id, export.Params{Kind: kind, Format: format}, w); err != nil {
		writeErr(w, http.StatusInternalServerError, "export: "+err.Error())
		return
	}
}

// safeFilename keeps letters, digits, dots, dashes and underscores.
func safeFilename(s string) string {
	out := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			return r
		}
		return '-'
	}, s)
	if strings.Trim(out, "-.") == "" {
		return "repository"
	}
	return out
}
