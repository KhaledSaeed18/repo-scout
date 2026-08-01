// Package config holds runtime configuration and user-adjustable settings.
package config

import (
	"fmt"
	"os"
)

// Runner holds process-level configuration read from the environment.
type Runner struct {
	Addr   string
	DBPath string
}

// FromEnv builds the runner configuration from environment variables, falling
// back to sensible defaults.
func FromEnv() Runner {
	return Runner{
		Addr:   env("REPO_SCOUT_ADDR", ":8080"),
		DBPath: env("REPO_SCOUT_DB", "data/reposcout.db"),
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Settings are user-adjustable, persisted preferences that shape scanning and
// analysis behavior.
type Settings struct {
	IgnoreFolders    []string `json:"ignoreFolders"`
	IgnoreExtensions []string `json:"ignoreExtensions"`
	MaxFileSize      int64    `json:"maxFileSize"`
	MaxFileDepth     int      `json:"maxFileDepth"`
	WorkerCount      int      `json:"workerCount"`
	MaxSearchFiles   int      `json:"maxSearchFiles"`
	DupMinLines      int      `json:"dupMinLines"`
	DupMinSimilarity float64  `json:"dupMinSimilarity"`
	Theme            string   `json:"theme"`
}

// Defaults returns the baseline settings before any user customization.
func Defaults() Settings {
	return Settings{
		IgnoreFolders:    []string{".git", "node_modules", "vendor", ".venv", "venv", "__pycache__", "dist", "build", ".next", ".cache", ".idea", ".vscode", "bin", "obj", "target", "coverage", "Pods", "DerivedData"},
		IgnoreExtensions: []string{},
		MaxFileSize:      4 << 20,
		MaxFileDepth:     0,
		WorkerCount:      4,
		MaxSearchFiles:   20000,
		DupMinLines:      6,
		DupMinSimilarity: 0.6,
		Theme:            "dark",
	}
}

// WithDefaults fills zero-valued fields with defaults. It is a no-op on a fully
// populated settings value.
func (s Settings) WithDefaults() Settings {
	d := Defaults()
	if s.WorkerCount == 0 {
		s.WorkerCount = d.WorkerCount
	}
	if s.MaxFileSize == 0 {
		s.MaxFileSize = d.MaxFileSize
	}
	if s.MaxSearchFiles == 0 {
		s.MaxSearchFiles = d.MaxSearchFiles
	}
	if s.DupMinLines == 0 {
		s.DupMinLines = d.DupMinLines
	}
	if s.DupMinSimilarity == 0 {
		s.DupMinSimilarity = d.DupMinSimilarity
	}
	if s.Theme == "" {
		s.Theme = d.Theme
	}
	if s.IgnoreFolders == nil {
		s.IgnoreFolders = d.IgnoreFolders
	}
	if s.IgnoreExtensions == nil {
		s.IgnoreExtensions = d.IgnoreExtensions
	}
	return s
}

// Validate ensures the settings are usable.
func (s Settings) Validate() error {
	if s.WorkerCount < 1 || s.WorkerCount > 64 {
		return fmt.Errorf("worker count must be between 1 and 64")
	}
	if s.MaxFileSize < 0 {
		return fmt.Errorf("max file size cannot be negative")
	}
	if s.DupMinSimilarity <= 0 || s.DupMinSimilarity > 1 {
		return fmt.Errorf("duplicate similarity must be in (0, 1]")
	}
	if s.DupMinLines < 1 {
		return fmt.Errorf("duplicate min lines must be at least 1")
	}
	return nil
}
