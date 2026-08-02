// Package scanner walks a repository tree, measures each eligible source file,
// and persists the results in batches. It is memory-bounded: only file paths
// are held between the walk and the worker pool, and contents are streamed
// one file at a time.
package scanner

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"gorm.io/gorm"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/langdetect"
	"github.com/KhaledSaeed18/repo-scout/internal/metrics"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// ProgressFunc reports scan progress. done and total are file counts.
type ProgressFunc func(done, total int)

// LangStats is a per-language rollup.
type LangStats struct {
	Files    int   `json:"files"`
	LOC      int   `json:"loc"`
	Code     int   `json:"code"`
	Comments int   `json:"comments"`
	Blank    int   `json:"blank"`
	Size     int64 `json:"size"`
}

// Stats aggregates a full scan.
type Stats struct {
	FileCount     int                   `json:"fileCount"`
	TotalSize     int64                 `json:"totalSize"`
	TotalLOC      int                   `json:"totalLOC"`
	TotalCode     int                   `json:"totalCode"`
	TotalComments int                   `json:"totalComments"`
	TotalBlank    int                   `json:"totalBlank"`
	BinaryCount   int                   `json:"binaryCount"`
	ByLanguage    map[string]*LangStats `json:"byLanguage"`
}

// Scanner walks repositories and records file measurements.
type Scanner struct {
	db *gorm.DB
}

// New builds a Scanner that persists to db.
func New(db *gorm.DB) *Scanner {
	return &Scanner{db: db}
}

// Scan walks root and writes one models.File row per eligible file. It honors
// the ignore rules, size limits, and depth limits in settings, and reports
// progress through progress.
func (s *Scanner) Scan(ctx context.Context, repoID uint, root string, settings config.Settings, progress ProgressFunc) (Stats, error) {
	ignoreDirs := toSet(settings.IgnoreFolders)
	ignoreExts := toSet(settings.IgnoreExtensions)

	paths := make([]string, 0, 1024)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if rel == "" || folderIgnored(rel, ignoreDirs) {
				return filepath.SkipDir
			}
			if settings.MaxFileDepth > 0 && depthOf(rel) >= settings.MaxFileDepth {
				return filepath.SkipDir
			}
			return nil
		}
		if _, ok := ignoreExts[strings.ToLower(filepath.Ext(path))]; ok {
			return nil
		}
		if settings.MaxFileSize > 0 {
			info, err := d.Info()
			if err == nil && info.Size() > settings.MaxFileSize {
				return nil
			}
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return Stats{}, fmt.Errorf("walk %s: %w", root, err)
	}

	total := len(paths)
	var done atomic.Int64

	workers := settings.WorkerCount
	if workers < 1 {
		workers = config.Defaults().WorkerCount
	}

	files := make(chan string, workers)
	results := make(chan models.File, workers*2)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range files {
				if ctx.Err() != nil {
					continue
				}
				if f, ok := process(repoID, root, path, settings); ok {
					results <- f
				}
				n := done.Add(1)
				if progress != nil {
					progress(int(n), total)
				}
			}
		}()
	}
	go func() {
	feed:
		for _, p := range paths {
			select {
			case files <- p:
			case <-ctx.Done():
				break feed
			}
		}
		close(files)
		wg.Wait()
		close(results)
	}()

	stats := Stats{ByLanguage: map[string]*LangStats{}}
	batch := make([]models.File, 0, batchSize)
	for f := range results {
		accumulate(&stats, f)
		batch = append(batch, f)
		if len(batch) >= batchSize {
			if err := s.insert(ctx, repoID, batch); err != nil {
				return Stats{}, err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := s.insert(ctx, repoID, batch); err != nil {
			return Stats{}, err
		}
	}
	if ctx.Err() != nil {
		return Stats{}, ctx.Err()
	}
	return stats, nil
}

const batchSize = 500

func (s *Scanner) insert(ctx context.Context, repoID uint, batch []models.File) error {
	if err := s.db.WithContext(ctx).Create(&batch).Error; err != nil {
		return fmt.Errorf("insert files for repo %d: %w", repoID, err)
	}
	return nil
}

func process(repoID uint, root, path string, settings config.Settings) (models.File, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return models.File{}, false
	}
	info, err := os.Lstat(path)
	if err != nil {
		return models.File{}, false
	}
	rel = filepath.ToSlash(rel)
	f := models.File{
		RepoID:    repoID,
		Path:      rel,
		Name:      filepath.Base(rel),
		Folder:    folderOf(rel),
		Extension: extNoDot(rel),
		Size:      info.Size(),
	}
	if lang := langdetect.Detect(rel); lang != nil {
		f.Language = lang.Name
	}
	if info.IsDir() {
		return models.File{}, false
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return f, true
	}
	if isBinary(content) {
		return f, true
	}
	if lang := langdetect.Detect(rel); lang != nil {
		loc := lang.Count(string(content))
		f.LinesTotal = loc.Lines
		f.LinesCode = loc.Code
		f.LinesComment = loc.Comments
		f.LinesBlank = loc.Blank
		metrics.Analyze(f, string(content)).Apply(&f)
	}
	return f, true
}

func accumulate(stats *Stats, f models.File) {
	stats.FileCount++
	stats.TotalSize += f.Size
	if f.LinesTotal > 0 {
		stats.TotalLOC += f.LinesTotal
		stats.TotalCode += f.LinesCode
		stats.TotalComments += f.LinesComment
		stats.TotalBlank += f.LinesBlank
	} else {
		stats.BinaryCount++
	}
	if f.Language == "" {
		return
	}
	ls, ok := stats.ByLanguage[f.Language]
	if !ok {
		ls = &LangStats{}
		stats.ByLanguage[f.Language] = ls
	}
	ls.Files++
	ls.LOC += f.LinesTotal
	ls.Code += f.LinesCode
	ls.Comments += f.LinesComment
	ls.Blank += f.LinesBlank
	ls.Size += f.Size
}

func toSet(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, it := range items {
		out[strings.ToLower(it)] = struct{}{}
	}
	return out
}

// folderIgnored reports whether any path segment is ignored.
func folderIgnored(rel string, ignored map[string]struct{}) bool {
	for _, seg := range strings.Split(rel, string(filepath.Separator)) {
		if _, ok := ignored[strings.ToLower(seg)]; ok {
			return true
		}
	}
	return false
}

func depthOf(rel string) int {
	if rel == "." {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

func folderOf(rel string) string {
	idx := strings.LastIndex(rel, "/")
	if idx < 0 {
		return ""
	}
	return rel[:idx]
}

func extNoDot(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if len(ext) > 0 {
		return ext[1:]
	}
	return ""
}

func isBinary(content []byte) bool {
	head := content
	if len(head) > 512 {
		head = head[:512]
	}
	return bytes.IndexByte(head, 0) >= 0
}
