package api

import (
	"fmt"
	"net/http"
	"slices"
	"sort"
	"time"

	"github.com/KhaledSaeed18/repo-scout/internal/architecture"
	"github.com/KhaledSaeed18/repo-scout/internal/gitanalytics"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
	"gorm.io/gorm"
)

// handleHeatmap returns daily activity, weekday/hour heatmap, and streaks.
func (s *Server) handleHeatmap(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	h, err := gitanalytics.ComputeHeatmap(s.db, id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "heatmap: "+err.Error())
		return
	}
	streaks, err := gitanalytics.AllStreaks(s.db, id, time.Now())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "streaks: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"heatmap": h, "streaks": streaks})
}

// handleDependencies lists manifests and their packages grouped by manager.
func (s *Server) handleDependencies(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var deps []models.Dependency
	if err := s.db.Where("repo_id = ?", id).Order("manager ASC, name ASC").Find(&deps).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "dependencies: "+err.Error())
		return
	}
	byManager := map[string][]models.Dependency{}
	for _, d := range deps {
		byManager[d.Manager] = append(byManager[d.Manager], d)
	}
	managers := make([]string, 0, len(byManager))
	for m := range byManager {
		managers = append(managers, m)
	}
	sort.Strings(managers)
	writeJSON(w, http.StatusOK, map[string]any{"managers": managers, "dependencies": byManager, "total": len(deps)})
}

// handleDuplicates returns duplicate groups with their source blocks.
func (s *Server) handleDuplicates(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var groups []models.DuplicateGroup
	if err := s.db.Where("repo_id = ?", id).Order("lines DESC").Find(&groups).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "duplicates: "+err.Error())
		return
	}
	type groupWithBlocks struct {
		models.DuplicateGroup
		Blocks []models.DuplicateBlock `json:"blocks"`
	}
	var blocks []models.DuplicateBlock
	if err := s.db.Where("repo_id = ?", id).Order("file_path ASC, start_line ASC").Find(&blocks).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "duplicate blocks: "+err.Error())
		return
	}
	byGroup := make(map[uint][]models.DuplicateBlock, len(groups))
	for _, b := range blocks {
		byGroup[b.GroupID] = append(byGroup[b.GroupID], b)
	}
	out := make([]groupWithBlocks, 0, len(groups))
	for _, g := range groups {
		gb := byGroup[g.ID]
		if gb == nil {
			gb = []models.DuplicateBlock{}
		}
		out = append(out, groupWithBlocks{DuplicateGroup: g, Blocks: gb})
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": out, "total": len(out)})
}

// handleArchitecture returns the import graph, cycles, and dead/unused files.
func (s *Server) handleArchitecture(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var edgeRows []models.ImportEdge
	if err := s.db.Where("repo_id = ?", id).Find(&edgeRows).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "architecture: "+err.Error())
		return
	}
	edges := make([]architecture.Edge, 0, len(edgeRows))
	for _, e := range edgeRows {
		edges = append(edges, architecture.Edge{From: e.FromFile, To: e.ToFile, Kind: e.ImportType, Resolved: e.Resolved})
	}
	var files []models.File
	if err := s.db.Select("path", "language").Where("repo_id = ?", id).Find(&files).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "architecture: "+err.Error())
		return
	}
	rep := architecture.ReportFromGraph(edges, files)
	writeJSON(w, http.StatusOK, rep)
}

// metricsTotals is the repository-wide sum of per-file metrics.
type metricsTotals struct {
	Files      int `json:"files"`
	LOC        int `json:"loc"`
	Code       int `json:"code"`
	Comments   int `json:"comments"`
	Blank      int `json:"blank"`
	Complexity int `json:"complexity"`
	Funcs      int `json:"funcs"`
	Imports    int `json:"imports"`
	Exports    int `json:"exports"`
}

// languageTotals is the per-language slice of metricsTotals.
type languageTotals struct {
	Files int `json:"files"`
	LOC   int `json:"loc"`
	Code  int `json:"code"`
}

// handleMetrics aggregates quality signals across the repository. All
// rollups run in SQL so memory stays flat regardless of repository size.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	limit := queryInt(r, "limit", 20, 100)
	if limit == 0 {
		limit = 20
	}
	files := func() *gorm.DB { return s.db.Model(&models.File{}).Where("repo_id = ?", id) }

	var totals metricsTotals
	if err := files().Select(`COUNT(*) AS files,
		COALESCE(SUM(lines_total), 0) AS loc,
		COALESCE(SUM(lines_code), 0) AS code,
		COALESCE(SUM(lines_comment), 0) AS comments,
		COALESCE(SUM(lines_blank), 0) AS blank,
		COALESCE(SUM(complexity), 0) AS complexity,
		COALESCE(SUM(func_count), 0) AS funcs,
		COALESCE(SUM(imports), 0) AS imports,
		COALESCE(SUM(exports), 0) AS exports`).Scan(&totals).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "metrics totals: "+err.Error())
		return
	}

	var langRows []struct {
		Language string
		Files    int
		LOC      int `gorm:"column:loc"`
		Code     int
	}
	if err := files().Select(`language, COUNT(*) AS files,
		COALESCE(SUM(lines_total), 0) AS loc,
		COALESCE(SUM(lines_code), 0) AS code`).
		Group("language").Scan(&langRows).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "metrics languages: "+err.Error())
		return
	}
	byLang := make(map[string]languageTotals, len(langRows))
	for _, row := range langRows {
		byLang[row.Language] = languageTotals{Files: row.Files, LOC: row.LOC, Code: row.Code}
	}

	var deepest struct {
		Path  string
		Depth int
	}
	const depthExpr = "LENGTH(path) - LENGTH(REPLACE(path, '/', ''))"
	if err := files().Select("path, " + depthExpr + " AS depth").
		Order("depth DESC, path ASC").Limit(1).Scan(&deepest).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "metrics depth: "+err.Error())
		return
	}

	topBy := func(order string) ([]models.File, error) {
		out := []models.File{}
		err := files().Where("language <> ''").Order(order).Limit(limit).Find(&out).Error
		return out, err
	}
	largest, err := topBy("lines_code DESC, path ASC")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "metrics largest: "+err.Error())
		return
	}
	mostComplex, err := topBy("complexity DESC, path ASC")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "metrics complex: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"totals":           totals,
		"languages":        byLang,
		"maxDepth":         deepest.Depth,
		"deepestFile":      deepest.Path,
		"largestFiles":     largest,
		"mostComplexFiles": mostComplex,
	})
}

// handleSVG renders the import graph as an SVG document.
func (s *Server) handleSVG(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var edgeRows []models.ImportEdge
	if err := s.db.Where("repo_id = ? AND resolved = ?", id, true).Find(&edgeRows).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "svg: "+err.Error())
		return
	}
	if len(edgeRows) == 0 {
		writeErr(w, http.StatusNotFound, "no graph to render")
		return
	}
	nodes := map[string]bool{}
	for _, e := range edgeRows {
		nodes[e.FromFile] = true
		nodes[e.ToFile] = true
	}
	svg := renderSVG(nodes, edgeRows)
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=architecture-%d.svg", id))
	_, _ = w.Write([]byte(svg))
}

// handleTrends returns the summaries of the latest scans, oldest first.
func (s *Server) handleTrends(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	limit := max(queryInt(r, "limit", 50, 500), 1)
	snapshots := []models.ScanSnapshot{}
	if err := s.db.Where("repo_id = ?", id).Order("scanned_at DESC, id DESC").Limit(limit).
		Find(&snapshots).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "trends: "+err.Error())
		return
	}
	slices.Reverse(snapshots)
	writeJSON(w, http.StatusOK, map[string]any{"snapshots": snapshots})
}
