package gitrepo

import (
	"archive/tar"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportWritesTheTreeOfACommit(t *testing.T) {
	root := makeRepo(t)
	a := New()
	ctx := context.Background()
	base, err := a.ResolveCommit(ctx, root, "v1.0")
	if err != nil {
		t.Fatal(err)
	}
	// Later work must not show up in the exported tree.
	writeFile(t, root, "later.go", "package later\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "later")

	dest := t.TempDir()
	if err := a.Export(ctx, root, base, dest); err != nil {
		t.Fatalf("export: %v", err)
	}
	for _, name := range []string{"a.go", "b.go", "c.go"} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Errorf("expected %s exported: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dest, "later.go")); err == nil {
		t.Error("later.go is not part of v1.0")
	}
	if status := gitOut(t, root, "status", "--porcelain"); status != "" {
		t.Errorf("export must not touch the repository, status: %q", status)
	}
}

func TestResolveCommitRejectsUnknownAndOptionLikeRefs(t *testing.T) {
	root := makeRepo(t)
	for _, ref := range []string{"no-such-branch", "--output=/tmp/x"} {
		if _, err := New().ResolveCommit(context.Background(), root, ref); err == nil {
			t.Errorf("expected %q to be rejected", ref)
		}
	}
}

func TestExtractRefusesPathsOutsideTheDestination(t *testing.T) {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	_ = tw.WriteHeader(&tar.Header{Name: "../escape.txt", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	err := extract(tar.NewReader(&buf), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected the entry to be refused, got %v", err)
	}
}
