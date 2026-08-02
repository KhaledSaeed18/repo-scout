package api

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type browseEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type browseResponse struct {
	Path    string        `json:"path"`
	Parent  string        `json:"parent"`
	Entries []browseEntry `json:"entries"`
}

// handleBrowse lists the subdirectories of a local filesystem path, for the
// scan page's folder picker. It defaults to the user's home directory.
func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "resolve home dir: "+err.Error())
			return
		}
		reqPath = home
	}
	abs, err := filepath.Abs(reqPath)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid path")
		return
	}
	info, err := os.Stat(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "path not accessible: "+err.Error())
		return
	}
	if !info.IsDir() {
		writeErr(w, http.StatusBadRequest, "path is not a directory")
		return
	}
	dirEntries, err := os.ReadDir(abs)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "read directory: "+err.Error())
		return
	}
	entries := make([]browseEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		name := e.Name()
		if !e.IsDir() || strings.HasPrefix(name, ".") {
			continue
		}
		entries = append(entries, browseEntry{Name: name, Path: filepath.Join(abs, name)})
	}
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	parent := filepath.Dir(abs)
	if parent == abs {
		parent = ""
	}
	writeJSON(w, http.StatusOK, browseResponse{Path: abs, Parent: parent, Entries: entries})
}
