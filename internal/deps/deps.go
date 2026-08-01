// Package deps parses dependency manifests (npm, Go modules, Cargo, Maven,
// Composer, pip) into a uniform dependency record.
package deps

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Entry is one declared dependency.
type Entry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Scope   string `json:"scope"`
}

// Managers supported by the parser.
const (
	ManagerNPM      = "npm"
	ManagerGo       = "go"
	ManagerCargo    = "cargo"
	ManagerMaven    = "maven"
	ManagerComposer = "composer"
	ManagerPip      = "pip"
)

// Parse detects the manifest type from the file name and extracts its
// dependencies. It returns an error only for structurally unreadable files;
// unknown files return an empty slice with a nil error.
func Parse(path, content string) (manager string, entries []Entry, err error) {
	base := filepath.Base(path)
	switch base {
	case "package.json":
		manager = ManagerNPM
		entries, err = parseJSONDeps(content, "dependencies", "devDependencies")
	case "composer.json":
		manager = ManagerComposer
		entries, err = parseJSONDeps(content, "require", "require-dev")
	case "go.mod":
		manager = ManagerGo
		entries, err = parseGoMod(content)
	case "Cargo.toml":
		manager = ManagerCargo
		entries, err = parseCargo(content)
	case "pom.xml":
		manager = ManagerMaven
		entries, err = parseMaven(content)
	case "requirements.txt", "requirements-dev.txt":
		manager = ManagerPip
		entries, err = parseRequirements(content)
	default:
		if strings.HasSuffix(base, ".mod") {
			manager = ManagerGo
			entries, err = parseGoMod(content)
		}
	}
	return manager, entries, err
}

func parseJSONDeps(content string, prodKeys ...string) ([]Entry, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("parse json manifest: %w", err)
	}
	var out []Entry
	for _, key := range prodKeys {
		scope := "production"
		if key == "devDependencies" || key == "require-dev" {
			scope = "development"
		}
		var deps map[string]json.RawMessage
		if err := json.Unmarshal(doc[key], &deps); err != nil {
			continue
		}
		for name, raw := range deps {
			var version any
			if err := json.Unmarshal(raw, &version); err != nil {
				continue
			}
			switch v := version.(type) {
			case string:
				out = append(out, Entry{Name: name, Version: v, Scope: scope})
			case map[string]any:
				if s, ok := v["version"].(string); ok {
					out = append(out, Entry{Name: name, Version: s, Scope: scope})
				}
			}
		}
	}
	return out, nil
}

func parseGoMod(content string) ([]Entry, error) {
	var out []Entry
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		if line == "" {
			continue
		}
		if line == "require (" {
			inBlock = true
			continue
		}
		if line == ")" {
			inBlock = false
			continue
		}
		if strings.HasPrefix(line, "require ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "require "))
		} else if !inBlock {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] != "" {
			out = append(out, Entry{Name: fields[0], Version: fields[1], Scope: "production"})
		}
	}
	return out, nil
}

func parseRequirements(content string) ([]Entry, error) {
	var out []Entry
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.SplitN(line, "#", 2)[0]
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "==", 2)
		op := "=="
		if len(parts) == 1 {
			parts = strings.SplitN(line, ">=", 2)
			op = ">="
		}
		if len(parts) == 1 {
			parts = strings.SplitN(line, "~=", 2)
			op = "~="
		}
		if len(parts) == 1 {
			parts = strings.SplitN(line, "!=", 2)
			op = "!="
		}
		name := strings.TrimSpace(parts[0])
		version := ""
		if len(parts) > 1 {
			version = op + strings.TrimSpace(parts[1])
		}
		if name != "" {
			out = append(out, Entry{Name: name, Version: version, Scope: "production"})
		}
	}
	return out, nil
}

var cargoSectionRe = regexp.MustCompile(`^\s*\[([a-z-]+)\]\s*$`)

func parseCargo(content string) ([]Entry, error) {
	var out []Entry
	scope := "production"
	for _, line := range strings.Split(content, "\n") {
		if m := cargoSectionRe.FindStringSubmatch(line); m != nil {
			switch m[1] {
			case "dependencies":
				scope = "production"
			case "dev-dependencies", "build-dependencies":
				scope = "development"
			default:
				scope = ""
			}
			continue
		}
		line = strings.TrimSpace(line)
		if scope == "" || line == "" || strings.HasPrefix(line, "#") || line == "[" {
			continue
		}
		name, version, ok := parseCargoLine(line)
		if ok {
			out = append(out, Entry{Name: name, Version: version, Scope: scope})
		}
	}
	return out, nil
}

func parseCargoLine(line string) (name, version string, ok bool) {
	line = strings.TrimSuffix(line, ",")
	if strings.HasPrefix(line, "\"") {
		return "", "", false
	}
	if eq := strings.Index(line, "="); eq >= 0 {
		name = strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, "\"")
		if strings.HasPrefix(val, "{") {
			// table dependency: version = "1.0", path = "..."
			if m := regexp.MustCompile(`version\s*=\s*"([^"]+)"`).FindStringSubmatch(val); m != nil {
				version = m[1]
			}
		} else if val != "" {
			version = val
		}
		return name, version, true
	}
	if ws := strings.IndexByte(line, ' '); ws >= 0 {
		return strings.TrimSpace(line[:ws]), strings.TrimSpace(line[ws+1:]), true
	}
	return line, "", true
}

func parseMaven(content string) ([]Entry, error) {
	var pom struct {
		Dependencies []struct {
			GroupID    string `xml:"groupId"`
			ArtifactID string `xml:"artifactId"`
			Version    string `xml:"version"`
			Scope      string `xml:"scope"`
		} `xml:"dependencies>dependency"`
	}
	if err := xml.Unmarshal([]byte(content), &pom); err != nil {
		return nil, fmt.Errorf("parse pom.xml: %w", err)
	}
	var out []Entry
	for _, d := range pom.Dependencies {
		name := d.GroupID + ":" + d.ArtifactID
		scope := d.Scope
		if scope == "" {
			scope = "production"
		}
		out = append(out, Entry{Name: name, Version: d.Version, Scope: scope})
	}
	return out, nil
}
