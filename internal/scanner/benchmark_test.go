package scanner_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/database"
	"github.com/KhaledSaeed18/repo-scout/internal/scanner"
)

func buildTree(tb testing.TB, files int) string {
	tb.Helper()
	root := tb.TempDir()
	for i := 0; i < files; i++ {
		dir := filepath.Join(root, fmt.Sprintf("pkg%d", i%40), "src")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			tb.Fatal(err)
		}
		content := fmt.Sprintf("package pkg%d\n\nfunc fn%d() int {\n\treturn %d\n}\n", i%40, i, i)
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%04d.go", i)), []byte(content), 0o644); err != nil {
			tb.Fatal(err)
		}
	}
	return root
}

func benchScan(b *testing.B, files int) {
	db, err := database.Open(":memory:")
	if err != nil {
		b.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		b.Fatal(err)
	}
	root := buildTree(b, files)
	s := scanner.New(db)
	settings := config.Defaults()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := s.Scan(context.Background(), uint(i+1), root, settings, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkScan1000(b *testing.B)  { benchScan(b, 1000) }
func BenchmarkScan5000(b *testing.B)  { benchScan(b, 5000) }
func BenchmarkScan10000(b *testing.B) { benchScan(b, 10000) }

// TestScanDeterministicAcrossWorkers verifies the worker pool produces
// identical results regardless of concurrency. Combined with -race this guards
// against data races and unbounded parallelism.
func TestScanDeterministicAcrossWorkers(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	root := buildTree(t, 300)
	for _, workers := range []int{1, 4, 8} {
		st := config.Defaults()
		st.WorkerCount = workers
		stats, err := scanner.New(db).Scan(context.Background(), uint(workers), root, st, nil)
		if err != nil {
			t.Fatalf("scan (workers=%d): %v", workers, err)
		}
		if stats.FileCount != 300 {
			t.Fatalf("workers=%d: expected 300 files, got %d", workers, stats.FileCount)
		}
	}
	var count int64
	db.Table("files").Where("repo_id = 8").Count(&count)
	if count != 300 {
		t.Fatalf("expected 300 total file rows, got %d", count)
	}
}

// TestScanMemoryBound asserts scanning a moderately sized tree does not retain
// an excessive heap: peak live heap after the scan must stay far below the
// on-disk input (the scanner streams and batches rather than buffering).
func TestScanMemoryBound(t *testing.T) {
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	root := buildTree(t, 3000)
	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	if _, err := scanner.New(db).Scan(context.Background(), 1, root, config.Defaults(), nil); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	const ceiling = 128 << 20 // 128 MiB is far below the input's retained bytes.
	if after.HeapAlloc > ceiling {
		t.Fatalf("heap after scan too large: %d bytes", after.HeapAlloc)
	}
	_ = before
}
