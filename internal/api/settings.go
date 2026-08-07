package api

import (
	"encoding/json"
	"net/http"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
)

// handleGetSettings returns the effective scanning settings.
func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	st, err := s.settings.Load()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "load settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// handlePutSettings persists updated settings.
func (s *Server) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	var st config.Settings
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	st = st.WithDefaults()
	if err := st.Validate(); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid settings: "+err.Error())
		return
	}
	if err := s.settings.Save(st); err != nil {
		writeErr(w, http.StatusInternalServerError, "save settings: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, st)
}
