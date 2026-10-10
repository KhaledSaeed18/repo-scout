package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/coupling"
	"github.com/KhaledSaeed18/repo-scout/internal/risk"
)

func sample() Report {
	return Report{
		Tool: "repo-scout", Version: "1.2.3", Repository: "/src/demo",
		Summary: Summary{Files: 1, LinesOfCode: 120, Functions: 4, Complexity: 30, Commits: 12,
			Contributors: 1, Dependencies: 3, DuplicateGroups: 2},
		Hotspots: []risk.Hotspot{{Path: "core/a.go", Revisions: 9, Complexity: 25, Score: 1}},
		Cycles: []Cycle{
			{Folders: []string{"a", "b"}, At: &Location{Path: "a/a.go"}},
			{Folders: []string{"c", "d"}},
		},
		HiddenCoupling: []coupling.Pair{{FileA: "x.ts", FileB: "y.ts", Shared: 4, Degree: 0.8}},
		Duplicates: []Duplicate{{Lines: 8, Similarity: 1, Locations: []Location{
			{Path: "p/one.go", StartLine: 3, EndLine: 10}, {Path: "p/two.go", StartLine: 5, EndLine: 12},
		}}},
		ComplexFiles: []ComplexFile{{Path: "core/a.go", Complexity: 25}},
	}
}

func TestEvaluateGates(t *testing.T) {
	r := sample()
	got := evaluate(r, Gates{MaxComplexity: 20, MaxDuplicates: 5, NoCycles: true, NoHiddenCoupling: true})
	want := map[string]bool{"max-complexity": false, "max-duplicates": true, "no-cycles": false, "no-hidden-coupling": false}
	if len(got) != len(want) {
		t.Fatalf("expected %d gates, got %+v", len(want), got)
	}
	for _, g := range got {
		if g.Passed != want[g.Name] {
			t.Errorf("%s: passed=%v, want %v (%s)", g.Name, g.Passed, want[g.Name], g.Detail)
		}
	}
	if len(evaluate(r, Gates{})) != 0 {
		t.Fatal("expected no gates when none are enabled")
	}
	r.Gates = got
	if r.Passed() {
		t.Fatal("expected a failing report")
	}
}

func TestWriteSARIF(t *testing.T) {
	r := sample()
	r.Gates = evaluate(r, Gates{MaxComplexity: 20, NoCycles: true})
	var buf bytes.Buffer
	if err := WriteSARIF(&buf, r); err != nil {
		t.Fatal(err)
	}
	var log sarifLog
	if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if log.Version != "2.1.0" || len(log.Runs) != 1 || log.Runs[0].Tool.Driver.Version != "1.2.3" {
		t.Fatalf("unexpected envelope: %+v", log)
	}
	levels := map[string]string{}
	count := map[string]int{}
	for _, res := range log.Runs[0].Results {
		levels[res.RuleID] = res.Level
		count[res.RuleID]++
		for _, l := range res.Locations {
			if l.Physical.Region.StartLine < 1 || l.Physical.Artifact.URI == "" {
				t.Errorf("%s: location needs a file and a start line, got %+v", res.RuleID, l)
			}
		}
	}
	// The cycle without a known import has no location and is left out.
	if count[ruleCycle] != 1 || levels[ruleCycle] != "error" {
		t.Errorf("expected one failing cycle result, got %d at %q", count[ruleCycle], levels[ruleCycle])
	}
	if levels[ruleComplex] != "error" || levels[ruleDuplicate] != "warning" ||
		levels[ruleCoupling] != "note" || levels[ruleHotspot] != "note" {
		t.Errorf("unexpected levels %v", levels)
	}
	for _, res := range log.Runs[0].Results {
		if res.RuleID == ruleDuplicate {
			if len(res.RelatedLocations) != 1 || res.RelatedLocations[0].Physical.Region.EndLine != 12 {
				t.Errorf("expected the second copy as a related location, got %+v", res.RelatedLocations)
			}
		}
	}
}

func TestWriteText(t *testing.T) {
	r := sample()
	r.Gates = evaluate(r, Gates{MaxDuplicates: 1})
	var buf bytes.Buffer
	if err := WriteText(&buf, r); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"1 file, 120 lines of code, 4 functions",
		"12 commits by 1 contributor, 3 dependencies, 2 duplicate groups",
		"a -> b -> a  (from a/a.go)",
		"x.ts <-> y.ts",
		"p/one.go:3, p/two.go:5",
		"FAIL  max-duplicates",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in:\n%s", want, out)
		}
	}
}

func TestWriteJSONRoundTrips(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, sample()); err != nil {
		t.Fatal(err)
	}
	var back Report
	if err := json.Unmarshal(buf.Bytes(), &back); err != nil {
		t.Fatal(err)
	}
	if back.Summary.LinesOfCode != 120 || back.Duplicates[0].Locations[1].Path != "p/two.go" {
		t.Fatalf("unexpected round trip: %+v", back)
	}
}
