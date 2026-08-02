// Package duplicates finds duplicated code blocks across files using
// hash-based shingling on normalized code lines.
package duplicates

import (
	"context"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/langdetect"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// Detector finds duplicated blocks. It is memory-bounded: the number of
// indexed shingles is capped so very large repositories cannot exhaust RAM.
type Detector struct {
	Window        int     // shingle size in normalized lines
	MinSimilarity float64 // minimum pair similarity to form a group
	MaxHashFiles  int     // hashes shared by more files than this are ignored
	MaxShingles   int64   // total shingles indexed before stopping
}

// New builds a Detector from settings.
func New(settings config.Settings) *Detector {
	return &Detector{
		Window:        settings.DupMinLines,
		MinSimilarity: settings.DupMinSimilarity,
		MaxHashFiles:  30,
		MaxShingles:   4_000_000,
	}
}

// Result reports the duplicate groups and their blocks.
type Result struct {
	Groups       []models.DuplicateGroup
	Blocks       []models.DuplicateBlock
	IndexedFiles int
	SkippedFiles int
}

type occurrence struct {
	file int
	line int // start index of the shingle in normalized lines
}

// Detect scans the given files (which must have a detected language) and
// returns duplicate groups and blocks. read reads file content by path.
func (d *Detector) Detect(ctx context.Context, repoID uint, root string, files []models.File, read func(rel string) (string, error)) (Result, error) {
	res := Result{}
	if read == nil {
		read = func(rel string) (string, error) {
			data, err := os.ReadFile(filepath.Join(root, rel))
			return string(data), err
		}
	}
	if d.Window < 2 {
		d.Window = 2
	}

	// Select candidate files and normalize them.
	type candidate struct {
		file         models.File
		norms        []string
		shingleCount int
	}
	var cands []candidate
	fileOrig := map[int][]int{} // indexed file index -> original line numbers

	var totalShingles int64
	for _, f := range files {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		lang := langdetect.Detect(f.Path)
		if lang == nil || f.Language == "" {
			res.SkippedFiles++
			continue
		}
		content, err := read(f.Path)
		if err != nil {
			res.SkippedFiles++
			continue
		}
		norm := langdetect.Normalize(lang, content)
		if len(norm) < d.Window {
			res.SkippedFiles++
			continue
		}
		norms := make([]string, len(norm))
		orig := make([]int, len(norm))
		for i, l := range norm {
			norms[i] = l.Text
			orig[i] = l.Number
		}
		idx := len(cands)
		cands = append(cands, candidate{file: f, norms: norms, shingleCount: len(norm) - d.Window + 1})
		fileOrig[idx] = orig
		res.IndexedFiles++
		totalShingles += int64(len(norm) - d.Window + 1)
		if totalShingles >= d.MaxShingles {
			break
		}
	}

	// Build the inverted shingle index.
	hashOcc := map[uint64][]occurrence{}
	hashText := map[uint64]string{}
	for ci, c := range cands {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		for j := 0; j+d.Window <= len(c.norms); j++ {
			h := shingleHash(c.norms[j : j+d.Window])
			if len(hashOcc[h]) == 0 {
				hashText[h] = strings.Join(c.norms[j:j+d.Window], "\n")
			}
			hashOcc[h] = append(hashOcc[h], occurrence{file: ci, line: j})
		}
	}

	// Count shared shingles between file pairs.
	type pairKey struct{ lo, hi int }
	pairShared := map[pairKey]int{}
	for _, occs := range hashOcc {
		if len(occs) < 2 || len(occs) > d.MaxHashFiles {
			continue
		}
		for a := 0; a < len(occs); a++ {
			for b := a + 1; b < len(occs); b++ {
				if occs[a].file == occs[b].file {
					continue
				}
				lo, hi := occs[a].file, occs[b].file
				if lo > hi {
					lo, hi = hi, lo
				}
				pairShared[pairKey{lo, hi}]++
			}
		}
	}

	// Union files with sufficient similarity.
	uf := newUnionFind(len(cands))
	pairSim := map[pairKey]float64{}
	compBestSim := map[int]float64{}
	for key, shared := range pairShared {
		la := cands[key.lo].shingleCount
		lb := cands[key.hi].shingleCount
		if la == 0 || lb == 0 {
			continue
		}
		sim := 2 * float64(shared) / float64(la+lb)
		if sim < d.MinSimilarity {
			continue
		}
		pairSim[key] = sim
		uf.union(key.lo, key.hi)
		comp := uf.find(key.lo)
		if sim > compBestSim[comp] {
			compBestSim[comp] = sim
		}
	}

	// Assign component ids and collect windows per component.
	compOf := make([]int, len(cands))
	compID := map[int]int{}
	nextID := 0
	for i := range cands {
		r := uf.find(i)
		if _, ok := compID[r]; !ok {
			compID[r] = nextID
			nextID++
		}
		compOf[i] = compID[r]
	}
	valid := make([]bool, nextID)
	for key := range pairSim {
		valid[compOf[key.lo]] = true
		valid[compOf[key.hi]] = true
	}

	compWindows := make([][]occurrence, nextID)
	for _, occs := range hashOcc {
		if len(occs) < 2 {
			continue
		}
		first := compOf[occs[0].file]
		same := true
		for _, o := range occs[1:] {
			if compOf[o.file] != first {
				same = false
				break
			}
		}
		if !same {
			continue
		}
		if !valid[first] {
			continue
		}
		for _, o := range occs {
			if compOf[o.file] == first {
				compWindows[first] = append(compWindows[first], o)
			}
		}
	}

	// Emit groups and blocks.
	for cid := 0; cid < nextID; cid++ {
		windows := compWindows[cid]
		if len(windows) == 0 {
			continue
		}
		filesInComp := map[int]bool{}
		for _, w := range windows {
			filesInComp[w.file] = true
		}
		if len(filesInComp) < 2 {
			continue
		}

		// Representative fragment from the most common shared shingle.
		repText := ""
		repBest := -1
		for h, occs := range hashOcc {
			inComp := 0
			for _, o := range occs {
				if compOf[o.file] == cid {
					inComp++
				}
			}
			if inComp > repBest {
				repBest = inComp
				repText = hashText[h]
			}
		}

		lines := 0
		if repText != "" {
			lines = d.Window
		}
		group := models.DuplicateGroup{
			RepoID:     repoID,
			Fragment:   repText,
			Lines:      lines,
			Similarity: compBestSim[cid],
			FileCount:  len(filesInComp),
		}
		res.Groups = append(res.Groups, group)
		gid := uint(len(res.Groups))

		for file := range filesInComp {
			var starts []int
			for _, w := range windows {
				if w.file == file {
					starts = append(starts, w.line)
				}
			}
			for _, block := range mergeRuns(starts, d.Window) {
				orig := fileOrig[file]
				lo := orig[block.lo]
				hi := orig[block.hi]
				res.Blocks = append(res.Blocks, models.DuplicateBlock{
					GroupID:   gid,
					RepoID:    repoID,
					FilePath:  cands[file].file.Path,
					StartLine: lo,
					EndLine:   hi,
				})
			}
		}
	}
	return res, nil
}

type run struct{ lo, hi int }

// mergeRuns merges overlapping or adjacent shingle windows into maximal runs
// and returns the normalized-line range of each run.
func mergeRuns(starts []int, window int) []run {
	sort.Ints(starts)
	if len(starts) == 0 {
		return nil
	}
	var runs []run
	cur := run{lo: starts[0], hi: starts[0] + window - 1}
	for _, s := range starts[1:] {
		if s <= cur.hi {
			if s+window-1 > cur.hi {
				cur.hi = s + window - 1
			}
			continue
		}
		runs = append(runs, cur)
		cur = run{lo: s, hi: s + window - 1}
	}
	runs = append(runs, cur)
	return runs
}

func shingleHash(lines []string) uint64 {
	h := fnv.New64a()
	for _, l := range lines {
		_, _ = h.Write([]byte(l))
		_, _ = h.Write([]byte{0})
	}
	return h.Sum64()
}

// unionFind is a size-tracking disjoint-set.
type unionFind struct {
	parent []int
	size   []int
}

func newUnionFind(n int) *unionFind {
	uf := &unionFind{parent: make([]int, n), size: make([]int, n)}
	for i := 0; i < n; i++ {
		uf.parent[i] = i
		uf.size[i] = 1
	}
	return uf
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return
	}
	if u.size[ra] < u.size[rb] {
		ra, rb = rb, ra
	}
	u.parent[rb] = ra
	u.size[ra] += u.size[rb]
}
