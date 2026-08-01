package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/architecture"
	"github.com/KhaledSaeed18/repo-scout/internal/gitanalytics"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
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
	var emails []string
	s.db.Model(&models.Contributor{}).Where("repo_id = ?", id).Pluck("email", &emails)
	streaks := make([]gitanalytics.StreaksResult, 0, len(emails))
	for _, email := range emails {
		sr, err := gitanalytics.Streaks(s.db, id, email)
		if err == nil {
			streaks = append(streaks, sr)
		}
	}
	sort.Slice(streaks, func(i, j int) bool { return streaks[i].Longest.Days > streaks[j].Longest.Days })
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
	out := make([]groupWithBlocks, 0, len(groups))
	for _, g := range groups {
		var blocks []models.DuplicateBlock
		s.db.Where("group_id = ?", g.ID).Order("file_path ASC").Find(&blocks)
		out = append(out, groupWithBlocks{DuplicateGroup: g, Blocks: blocks})
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

// handleMetrics aggregates quality signals across the repository.
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var files []models.File
	if err := s.db.Where("repo_id = ?", id).Find(&files).Error; err != nil {
		writeErr(w, http.StatusInternalServerError, "metrics: "+err.Error())
		return
	}

	byLang := map[string]struct {
		Files int `json:"files"`
		LOC   int `json:"loc"`
		Code  int `json:"code"`
	}{}
	totals := struct {
		Files      int `json:"files"`
		LOC        int `json:"loc"`
		Code       int `json:"code"`
		Comments   int `json:"comments"`
		Blank      int `json:"blank"`
		Complexity int `json:"complexity"`
		Funcs      int `json:"funcs"`
		Imports    int `json:"imports"`
		Exports    int `json:"exports"`
	}{}
	maxComplexity := 0.0
	maxLoc := 0
	deepest := ""
	maxDepth := 0
	var largest []models.File
	var mostComplex []models.File

	for _, f := range files {
		l := byLang[f.Language]
		l.Files++
		l.LOC += f.LinesTotal
		l.Code += f.LinesCode
		byLang[f.Language] = l
		totals.Files++
		totals.LOC += f.LinesTotal
		totals.Code += f.LinesCode
		totals.Comments += f.LinesComment
		totals.Blank += f.LinesBlank
		totals.Complexity += f.Complexity
		totals.Funcs += f.FuncCount
		totals.Imports += f.Imports
		totals.Exports += f.Exports
		if f.LinesCode > maxLoc {
			maxLoc = f.LinesCode
		}
		if float64(f.Complexity) > maxComplexity {
			maxComplexity = float64(f.Complexity)
		}
		if depth := strings.Count(f.Path, "/"); depth > maxDepth {
			maxDepth = depth
			deepest = f.Path
		}
		if f.Language != "" {
			largest = append(largest, f)
			mostComplex = append(mostComplex, f)
		}
	}
	sort.Slice(largest, func(i, j int) bool { return largest[i].LinesCode > largest[j].LinesCode })
	sort.Slice(mostComplex, func(i, j int) bool { return mostComplex[i].Complexity > mostComplex[j].Complexity })

	if largest == nil {
		largest = []models.File{}
	}
	if mostComplex == nil {
		mostComplex = []models.File{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"totals":           totals,
		"languages":        byLang,
		"maxDepth":         maxDepth,
		"deepestFile":      deepest,
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
	w.Write([]byte(svg))
}
