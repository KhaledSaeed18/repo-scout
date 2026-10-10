package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// SARIF 2.1.0, the format GitHub code scanning and most CI dashboards read.
// Only the fields Repo Scout fills are modeled.

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	ShortDescription sarifMessage `json:"shortDescription"`
	DefaultConfig    sarifConfig  `json:"defaultConfiguration"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID           string          `json:"ruleId"`
	Level            string          `json:"level"`
	Message          sarifMessage    `json:"message"`
	Locations        []sarifLocation `json:"locations"`
	RelatedLocations []sarifLocation `json:"relatedLocations,omitempty"`
}

type sarifLocation struct {
	ID       int           `json:"id,omitempty"`
	Physical sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	Artifact sarifArtifact `json:"artifactLocation"`
	Region   sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine,omitempty"`
}

// Rule IDs are stable; dashboards track findings by them.
const (
	ruleCycle     = "RS001"
	ruleComplex   = "RS002"
	ruleDuplicate = "RS003"
	ruleCoupling  = "RS004"
	ruleHotspot   = "RS005"
	ruleGrowth    = "RS006"
)

func sarifRules() []sarifRule {
	rule := func(id, name, text, level string) sarifRule {
		return sarifRule{ID: id, Name: name, ShortDescription: sarifMessage{Text: text}, DefaultConfig: sarifConfig{Level: level}}
	}
	return []sarifRule{
		rule(ruleCycle, "CircularDependency", "Folders import each other in a loop.", "warning"),
		rule(ruleComplex, "ComplexFile", "File complexity is above the configured limit.", "error"),
		rule(ruleDuplicate, "DuplicateCode", "A block of code is repeated across files.", "warning"),
		rule(ruleCoupling, "HiddenDependency", "Files keep changing together with nothing linking them.", "note"),
		rule(ruleHotspot, "Hotspot", "A complex file that changes often.", "note"),
		rule(ruleGrowth, "ComplexityIncrease", "A file grew more complex than it is in the base.", "note"),
	}
}

func location(l Location) sarifLocation {
	region := sarifRegion{StartLine: max(l.StartLine, 1)}
	if l.EndLine > l.StartLine {
		region.EndLine = l.EndLine
	}
	return sarifLocation{Physical: sarifPhysical{Artifact: sarifArtifact{URI: l.Path}, Region: region}}
}

// WriteSARIF writes the findings as a SARIF 2.1.0 log. Findings that fail an
// enabled gate are errors; the rest keep their rule's default level.
func WriteSARIF(w io.Writer, r Report) error {
	failing := map[string]bool{}
	for _, g := range r.Gates {
		failing[g.Name] = !g.Passed
	}
	level := func(def, gate string) string {
		if failing[gate] {
			return "error"
		}
		return def
	}

	results := []sarifResult{}
	for _, c := range r.Cycles {
		if c.At == nil {
			continue
		}
		lvl := level("warning", "no-cycles")
		if c.New && failing["no-new-cycles"] {
			lvl = "error"
		}
		results = append(results, sarifResult{
			RuleID: ruleCycle, Level: lvl,
			Message:   sarifMessage{Text: fmt.Sprintf("Circular dependency: %s -> %s.", strings.Join(c.Folders, " -> "), c.Folders[0])},
			Locations: []sarifLocation{location(*c.At)},
		})
	}
	for _, f := range r.ComplexFiles {
		results = append(results, sarifResult{
			RuleID: ruleComplex, Level: "error",
			Message:   sarifMessage{Text: fmt.Sprintf("Complexity %d is above the limit.", f.Complexity)},
			Locations: []sarifLocation{location(Location{Path: f.Path})},
		})
	}
	for _, d := range r.Duplicates {
		if len(d.Locations) == 0 {
			continue
		}
		res := sarifResult{
			RuleID: ruleDuplicate, Level: level("warning", "max-duplicates"),
			Message: sarifMessage{Text: fmt.Sprintf("%d lines repeated in %d places (%.0f%% similar).",
				d.Lines, len(d.Locations), d.Similarity*100)},
			Locations: []sarifLocation{location(d.Locations[0])},
		}
		for i, l := range d.Locations[1:] {
			rel := location(l)
			rel.ID = i + 1
			res.RelatedLocations = append(res.RelatedLocations, rel)
		}
		results = append(results, res)
	}
	for _, p := range r.HiddenCoupling {
		results = append(results, sarifResult{
			RuleID: ruleCoupling, Level: level("note", "no-hidden-coupling"),
			Message: sarifMessage{Text: fmt.Sprintf("Changes together with %s in %d commits (%.0f%% coupling), with no import, test or lockfile linking them.",
				p.FileB, p.Shared, p.Degree*100)},
			Locations:        []sarifLocation{location(Location{Path: p.FileA})},
			RelatedLocations: []sarifLocation{{ID: 1, Physical: location(Location{Path: p.FileB}).Physical}},
		})
	}
	if c := r.Comparison; c != nil {
		for _, f := range c.ComplexityChanges {
			if f.After <= f.Before {
				continue
			}
			results = append(results, sarifResult{
				RuleID: ruleGrowth, Level: level("note", "max-complexity-increase"),
				Message: sarifMessage{Text: fmt.Sprintf("Complexity went from %d to %d against %s.",
					f.Before, f.After, c.Base)},
				Locations: []sarifLocation{location(Location{Path: f.Path})},
			})
		}
	}
	for _, h := range r.Hotspots {
		results = append(results, sarifResult{
			RuleID: ruleHotspot, Level: "note",
			Message: sarifMessage{Text: fmt.Sprintf("Hotspot: complexity %d, changed %d times in the last year.",
				h.Complexity, h.Revisions)},
			Locations: []sarifLocation{location(Location{Path: h.Path})},
		})
	}

	log := sarifLog{
		Schema:  "https://json.schemastore.org/sarif-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name: "Repo Scout", Version: r.Version,
				InformationURI: "https://github.com/KhaledSaeed18/repo-scout",
				Rules:          sarifRules(),
			}},
			Results: results,
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(log); err != nil {
		return fmt.Errorf("write sarif: %w", err)
	}
	return nil
}
