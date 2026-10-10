package gitrepo

import (
	"archive/tar"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ResolveCommit returns the commit hash a ref names, failing when the ref
// does not exist.
func (a *Analyzer) ResolveCommit(ctx context.Context, root, ref string) (string, error) {
	if strings.HasPrefix(ref, "-") {
		return "", fmt.Errorf("invalid ref %q", ref)
	}
	out, err := a.output(ctx, root, "rev-parse", "--verify", "--quiet", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("unknown ref %q", ref)
	}
	return strings.TrimSpace(out), nil
}

// Export writes the tree of a commit into dest, which must exist and be
// empty. It reads through git archive, so the repository and its working
// tree are never touched. Symbolic links are skipped and every path is
// checked to stay inside dest.
func (a *Analyzer) Export(ctx context.Context, root, commit, dest string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cmd := gitCommand(ctx, root, "archive", "--format=tar", commit)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("pipe git archive: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start git archive: %w", err)
	}
	if err := extract(tar.NewReader(stdout), dest); err != nil {
		cancel()
		_ = cmd.Wait()
		return err
	}
	// The tar reader stops at the end-of-archive marker, but git still writes
	// the record padding after it. Read it, or git blocks on a full pipe and
	// never exits (Windows pipes are too small to absorb it).
	if _, err := io.Copy(io.Discard, stdout); err != nil {
		return fmt.Errorf("read git archive: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive %s: %w", commit, err)
	}
	return nil
}

// extract unpacks regular files and folders from tr into dest.
func extract(tr *tar.Reader, dest string) error {
	base, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		target := filepath.Join(base, filepath.FromSlash(hdr.Name))
		if target != base && !strings.HasPrefix(target, base+string(filepath.Separator)) {
			return fmt.Errorf("archive entry %q escapes the destination", hdr.Name)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := createFile(target, tr, hdr.FileInfo().Mode().Perm()); err != nil {
				return err
			}
		}
		// Links and git's pax headers are not needed to analyze the code.
	}
}

func createFile(path string, r io.Reader, mode os.FileMode) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		_ = f.Close()
		return fmt.Errorf("write %s: %w", path, err)
	}
	return f.Close()
}
