package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer
	log, err := newLogger(&buf, "warn", "json")
	if err != nil {
		t.Fatal(err)
	}
	log.Info("hidden")
	log.Warn("shown", "n", 1)
	out := buf.String()
	if strings.Contains(out, "hidden") || !strings.Contains(out, `"msg":"shown"`) || !strings.Contains(out, `"n":1`) {
		t.Fatalf("unexpected output %q", out)
	}
	if _, err := newLogger(&buf, "loud", "text"); err == nil {
		t.Fatal("expected an error for an unknown level")
	}
	if _, err := newLogger(&buf, "info", "xml"); err == nil {
		t.Fatal("expected an error for an unknown format")
	}
}
