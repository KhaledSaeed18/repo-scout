package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// WriteJSON writes the report as indented JSON.
func WriteJSON(w io.Writer, r Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("write json: %w", err)
	}
	return nil
}

// WriteText writes a summary meant for a terminal or a CI log.
func WriteText(w io.Writer, r Report) error {
	var b strings.Builder
	s := r.Summary
	fmt.Fprintf(&b, "Repo Scout %s: %s\n\n", r.Version, r.Repository)
	fmt.Fprintf(&b, "  %s, %s of code, %s, total complexity %d\n",
		count(s.Files, "file"), count(s.LinesOfCode, "line"), count(s.Functions, "function"), s.Complexity)
	fmt.Fprintf(&b, "  %s by %s, %s, %s\n",
		count(s.Commits, "commit"), count(s.Contributors, "contributor"),
		count(s.Dependencies, "dependency"), count(s.DuplicateGroups, "duplicate group"))

	if len(r.Hotspots) > 0 {
		b.WriteString("\nHotspots (complex files changed most in the last year)\n")
		for _, h := range r.Hotspots {
			fmt.Fprintf(&b, "  %3.0f  %s  (%d changes, complexity %d)\n", h.Score*100, h.Path, h.Revisions, h.Complexity)
		}
	}
	if len(r.Cycles) > 0 {
		b.WriteString("\nCircular dependencies\n")
		for _, c := range r.Cycles {
			fmt.Fprintf(&b, "  %s -> %s", strings.Join(c.Folders, " -> "), c.Folders[0])
			if c.At != nil {
				fmt.Fprintf(&b, "  (from %s)", c.At.Path)
			}
			b.WriteString("\n")
		}
	}
	if len(r.HiddenCoupling) > 0 {
		b.WriteString("\nHidden dependencies (change together, nothing links them)\n")
		for _, p := range r.HiddenCoupling {
			fmt.Fprintf(&b, "  %3.0f%%  %s <-> %s  (%d shared commits)\n", p.Degree*100, p.FileA, p.FileB, p.Shared)
		}
	}
	if len(r.Duplicates) > 0 {
		b.WriteString("\nLargest duplicates\n")
		for _, d := range r.Duplicates {
			places := make([]string, 0, len(d.Locations))
			for _, l := range d.Locations {
				places = append(places, fmt.Sprintf("%s:%d", l.Path, l.StartLine))
			}
			fmt.Fprintf(&b, "  %d lines, %.0f%% similar: %s\n", d.Lines, d.Similarity*100, strings.Join(places, ", "))
		}
	}
	if len(r.Gates) > 0 {
		b.WriteString("\nQuality gates\n")
		for _, g := range r.Gates {
			status := "pass"
			if !g.Passed {
				status = "FAIL"
			}
			fmt.Fprintf(&b, "  %s  %s: %s\n", status, g.Name, g.Detail)
		}
	}
	if _, err := io.WriteString(w, b.String()); err != nil {
		return fmt.Errorf("write text: %w", err)
	}
	return nil
}

// count pairs a number with its noun: count(1, "file") is "1 file".
func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	if strings.HasSuffix(noun, "y") {
		return fmt.Sprintf("%d %sies", n, strings.TrimSuffix(noun, "y"))
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
