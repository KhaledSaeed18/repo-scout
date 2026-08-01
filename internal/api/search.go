package api

import (
	"net/http"
	"strconv"

	"github.com/KhaledSaeed18/repo-scout/internal/search"
)

// handleSearch proxies the search engine.
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	repoID, err := strconv.ParseUint(q.Get("repo"), 10, 64)
	if err != nil || repoID == 0 {
		writeErr(w, http.StatusBadRequest, "repo id required")
		return
	}
	query := search.Query{
		RepoID:        uint(repoID),
		Text:          q.Get("query"),
		Mode:          q.Get("mode"),
		CaseSensitive: q.Get("case") == "true",
		WholeWord:     q.Get("word") == "true",
		ExtFilter:     q.Get("ext"),
		Limit:         queryInt(r, "limit", 50, 200),
		Offset:        queryInt(r, "offset", 0, 0),
	}
	result, err := search.New(s.db).Search(r.Context(), query, s.currentSettings())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "search: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}
