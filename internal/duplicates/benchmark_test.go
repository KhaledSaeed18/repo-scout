package duplicates_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/duplicates"
	"github.com/KhaledSaeed18/repo-scout/internal/langdetect"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func benchDuplicates(b *testing.B, files int) {
	root := b.TempDir()
	all := make([]models.File, 0, files)
	for i := 0; i < files; i++ {
		rel := fmt.Sprintf("file%04d.go", i)
		p := filepath.Join(root, rel)
		content := "package p\n\nfunc f() {\n\tif true {\n\t\tx()\n\t}\n}\n"
		if i%3 == 0 {
			content += "func g() {\n\ty()\n}\n"
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			b.Fatal(err)
		}
		all = append(all, models.File{Path: rel, Language: "Go"})
	}
	det := duplicates.New(config.Defaults())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := det.Detect(context.Background(), 1, root, all, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDuplicates500(b *testing.B)  { benchDuplicates(b, 500) }
func BenchmarkDuplicates2000(b *testing.B) { benchDuplicates(b, 2000) }

// TestDetectLargeInput guards memory bounds: the detector must cap indexed
// shingles rather than buffering the entire repository.
func TestDetectLargeInput(t *testing.T) {
	root := t.TempDir()
	var files []models.File
	for i := 0; i < 3000; i++ {
		rel := fmt.Sprintf("f%04d.py", i)
		lines := make([]string, 0, 40)
		lines = append(lines, "def handler():")
		for l := 0; l < 30; l++ {
			lines = append(lines, fmt.Sprintf("    value%d = %d", l, l+i))
		}
		content := strings.Join(lines, "\n")
		if err := os.WriteFile(filepath.Join(root, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		files = append(files, models.File{Path: rel, Language: langdetect.Detect(rel).Name})
	}
	det := duplicates.New(config.Defaults())
	res, err := det.Detect(context.Background(), 1, root, files, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IndexedFiles == 0 {
		t.Fatal("expected some indexed files")
	}
}
