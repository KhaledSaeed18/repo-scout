package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

// renderSVG produces a simple layered SVG of the import graph. Nodes are laid
// out in columns by longest-path depth so edges flow left to right.
func renderSVG(nodes map[string]bool, edges []models.ImportEdge) string {
	names := make([]string, 0, len(nodes))
	for n := range nodes {
		names = append(names, n)
	}
	sort.Strings(names)

	idx := make(map[string]int, len(names))
	for i, n := range names {
		idx[n] = i
	}

	// Group edges by source to reduce clutter.
	adj := map[string][]string{}
	for _, e := range edges {
		adj[e.FromFile] = append(adj[e.FromFile], e.ToFile)
	}

	const (
		w        = 160
		h        = 36
		padX     = 24
		padY     = 16
		colSpace = 220
		rowSpace = 52
	)
	pos := map[string][2]float64{}
	cols := map[int][]string{}
	for _, n := range names {
		d := longestPathDepth(n, adj)
		cols[d] = append(cols[d], n)
	}
	maxCol := 0
	for c := range cols {
		if c > maxCol {
			maxCol = c
		}
	}
	row := map[string]int{}
	for c := 0; c <= maxCol; c++ {
		for i, n := range cols[c] {
			row[n] = i
			pos[n] = [2]float64{float64(padX + c*colSpace), float64(padY + i*rowSpace)}
		}
	}
	maxRows := 1
	for _, list := range cols {
		if len(list) > maxRows {
			maxRows = len(list)
		}
	}
	totalW := padX*2 + maxCol*colSpace + w
	totalH := padY*2 + maxRows*rowSpace

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="monospace" font-size="11">`,
		totalW, totalH, totalW, totalH)
	b.WriteString(`<rect width="100%" height="100%" fill="#0b0f19"/>`)

	for _, e := range edges {
		from, fok := pos[e.FromFile]
		to, tok := pos[e.ToFile]
		if !fok || !tok {
			continue
		}
		x1 := from[0] + w
		y1 := from[1] + h/2
		x2 := to[0]
		y2 := to[1] + h/2
		mid := (x1 + x2) / 2
		fmt.Fprintf(&b, `<path d="M %0.1f %0.1f C %0.1f %0.1f %0.1f %0.1f %0.1f %0.1f" fill="none" stroke="#38bdf8" stroke-opacity="0.45" stroke-width="1.2"/>`,
			x1, y1, mid, y1, mid, y2, x2, y2)
	}

	for _, n := range names {
		x, y := pos[n][0], pos[n][1]
		label := n
		if len(label) > 22 {
			label = label[:22]
		}
		fmt.Fprintf(&b, `<rect x="%0.1f" y="%0.1f" width="%d" height="%d" rx="6" fill="#111a2e" stroke="#334155" stroke-width="1"/>`, x, y, w, h)
		fmt.Fprintf(&b, `<text x="%0.1f" y="%0.1f" fill="#e2e8f0" dominant-baseline="middle">%s</text>`, x+8, y+h/2, escapeXML(label))
	}

	b.WriteString(`</svg>`)
	return b.String()
}

func longestPathDepth(node string, adj map[string][]string) int {
	depth := 0
	seen := map[string]bool{}
	var walk func(n string, d int)
	walk = func(n string, d int) {
		if d > depth {
			depth = d
		}
		if seen[n] {
			return
		}
		seen[n] = true
		for _, next := range adj[n] {
			walk(next, d+1)
		}
	}
	walk(node, 0)
	return depth
}

func escapeXML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
