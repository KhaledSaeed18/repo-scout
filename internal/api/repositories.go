package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/gitanalytics"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"gorm.io/gorm"
)

type createRepoRequest struct {
	Path string `json:"path"`
}

func (s *Server) handleCreateRepo(w http.ResponseWriter, r *http.Request) {
	var req createRepoRequest
	if err := decodeJSON(w, r, &req); err != nil {
		return
	}
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		writeErr(w, http.StatusBadRequest, "path is required")
		return
	}
	abs, err := filepath.Abs(req.Path)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid path")
		return
	}
	repo := models.Repository{Name: filepath.Base(abs), Path: abs}
	if err := s.db.Where("path = ?", abs).FirstOrCreate(&repo).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "create repository: "+err.Error())
		return
	}
	job, err := s.jobs.Enqueue(repo.ID, "scan")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "enqueue scan: "+err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"repository": repo, "job": job})
}

func (s *Server) handleListRepos(w http.ResponseWriter, r *http.Request) {
	var repos []models.Repository
	if err := s.db.Order("updated_at DESC").Find(&repos).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "list repositories: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"repositories": repos})
}

func (s *Server) handleGetRepo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	repo, ok := s.loadRepo(w, id)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, repo)
}

func (s *Server) handleDeleteRepo(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if _, ok := s.loadRepo(w, id); !ok {
		return
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := database.ClearRepoData(tx, id); err != nil {
			return err
		}
		if err := tx.Where("repo_id = ?", id).Delete(&models.Job{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Repository{}, id).Error
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "delete repository: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": true})
}

// handleFiles returns a paginated, filterable file list.
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	q := s.db.Model(&models.File{}).Where("repo_id = ?", id)
	if f := r.URL.Query().Get("folder"); f != "" {
		q = q.Where("folder = ?", f)
	}
	if f := r.URL.Query().Get("ext"); f != "" {
		q = q.Where("extension = ?", f)
	}
	if f := r.URL.Query().Get("language"); f != "" {
		q = q.Where("language = ?", f)
	}
	sortBy := r.URL.Query().Get("sort")
	switch sortBy {
	case "loc":
		q = q.Order("lines_code DESC")
	case "complexity":
		q = q.Order("complexity DESC")
	case "name":
		q = q.Order("name ASC")
	case "":
		q = q.Order("path ASC")
	default:
		writeErr(w, http.StatusBadRequest, "unsupported sort")
		return
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "count files: "+err.Error())
		return
	}
	limit := queryInt(r, "limit", 100, 1000)
	offset := queryInt(r, "offset", 0, 0)
	var files []models.File
	if err := q.Offset(offset).Limit(limit).Find(&files).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "list files: "+err.Error())
		return
	}
	if files == nil {
		files = []models.File{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": files, "total": total})
}

// handleTree returns the children of a folder (lazy tree).
func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = ""
	}
	var dirs []string
	if err := s.db.Model(&models.File{}).
		Where("repo_id = ? AND folder = ?", id, folder).
		Distinct().Pluck("name", &dirs).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "tree folders: "+err.Error())
		return
	}
	// Subfolders: distinct top-level segments under the requested folder.
	prefix := folder
	if prefix != "" {
		prefix += "/"
	}
	var subs []struct {
		Folder string
	}
	if err := s.db.Model(&models.File{}).
		Where("repo_id = ? AND folder LIKE ?", id, prefix+"%").Distinct().Pluck("folder", &subs).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "tree subfolders: "+err.Error())
		return
	}
	subSet := map[string]bool{}
	for _, sub := range subs {
		rest := strings.TrimPrefix(sub.Folder, prefix)
		if rest == "" {
			continue
		}
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			subSet[rest[:i]] = true
		} else {
			subSet[rest] = true
		}
	}
	subfolders := sortedKeys(subSet)

	var files []models.File
	if err := s.db.Where("repo_id = ? AND folder = ?", id, folder).
		Order("name ASC").Find(&files).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "tree files: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folder": folder, "folders": subfolders, "files": files})
}

// handleCommits returns the commit feed.
func (s *Server) handleCommits(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	commits, err := gitanalytics.CommitFeed(s.db, id, queryInt(r, "limit", 50, 500), queryInt(r, "offset", 0, 0))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "list commits: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"commits": commits})
}

// handleContributors returns the contributor leaderboard.
func (s *Server) handleContributors(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var rows []models.Contributor
	if err := s.db.Where("repo_id = ?", id).Order("commits DESC").Find(&rows).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "list contributors: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contributors": rows})
}

// handleLargestCommits returns the heaviest commits by total lines changed.
func (s *Server) handleLargestCommits(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	commits, err := gitanalytics.LargestCommits(s.db, id, queryInt(r, "limit", 20, 200))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "largest commits: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"commits": commits})
}

// handleOwnership returns per-author file ownership.
func (s *Server) handleOwnership(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ownership, err := gitanalytics.ComputeOwnership(s.db, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "ownership: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ownership)
}

// handleBranches returns the branches for a repository.
func (s *Server) handleBranches(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var branches []models.Branch
	if err := s.db.Where("repo_id = ?", id).Order("name ASC").Find(&branches).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "list branches: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": branches})
}

// handleTags returns the tags for a repository.
func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var tags []models.Tag
	if err := s.db.Where("repo_id = ?", id).Order("name ASC").Find(&tags).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "list tags: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tags": tags})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return err
	}
	return nil
}

func queryInt(r *http.Request, key string, def, max int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	if max > 0 && n > max {
		return max
	}
	return n
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
