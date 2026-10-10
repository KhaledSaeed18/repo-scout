package main

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// newLogger builds the process logger from REPO_SCOUT_LOG_LEVEL (debug, info,
// warn, error) and REPO_SCOUT_LOG_FORMAT (text, or json for log collectors).
func newLogger(w io.Writer, level, format string) (*slog.Logger, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("log level %q: use debug, info, warn or error", level)
	}
	opts := &slog.HandlerOptions{Level: lvl}
	switch strings.ToLower(format) {
	case "text", "":
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	}
	return nil, fmt.Errorf("log format %q: use text or json", format)
}
