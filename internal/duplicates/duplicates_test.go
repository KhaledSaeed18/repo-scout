package duplicates

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/KhaledSaeed18/repo-scout/internal/config"
	"github.com/KhaledSaeed18/repo-scout/internal/models"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const sharedBlock = `func handleRequest(req *Request) *Response {
	user := lookupUser(req.UserID)
	if user == nil {
		return errorResponse("user not found")
	}
	perms := user.Permissions()
	if !perms.Can(req.Action) {
		return errorResponse("forbidden")
	}
	return okResponse(user.Profile())
}
`

func TestDetectDuplicates(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package api\n\n"+sharedBlock+"\nfunc A() {}\n")
	write(t, root, "b.go", "package api\n\n"+sharedBlock+"\nfunc B() {}\n")
	write(t, root, "c.go", "package api\n\nfunc C() {\n\treturn 1\n}\n")
	write(t, root, "d.txt", "no language\n")

	files := []models.File{
		{Path: "a.go", Language: "Go", LinesTotal: 30},
		{Path: "b.go", Language: "Go", LinesTotal: 30},
		{Path: "c.go", Language: "Go", LinesTotal: 10},
		{Path: "d.txt", Language: "Plain Text", LinesTotal: 10},
	}

	d := New(config.Defaults())
	res, err := d.Detect(context.Background(), 1, root, files, nil)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(res.Groups) != 1 {
		t.Fatalf("expected 1 group, got %d (%+v)", len(res.Groups), res.Groups)
	}
	g := res.Groups[0]
	if g.FileCount != 2 {
		t.Fatalf("expected 2 files in group, got %d", g.FileCount)
	}
	if g.Similarity < 0.5 {
		t.Fatalf("expected high similarity, got %f", g.Similarity)
	}
	// two blocks, one per duplicated file
	if len(res.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(res.Blocks))
	}
	for _, b := range res.Blocks {
		if b.StartLine < 1 || b.EndLine < b.StartLine {
			t.Fatalf("invalid block range %d..%d", b.StartLine, b.EndLine)
		}
		if b.GroupID != 1 {
			t.Fatalf("expected group id 1, got %d", b.GroupID)
		}
	}
}

func TestNoDuplicates(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.go", "package a\nfunc a() { return 1 }\n")
	write(t, root, "b.go", "package b\nfunc b() { return 2 }\n")
	files := []models.File{
		{Path: "a.go", Language: "Go", LinesTotal: 3},
		{Path: "b.go", Language: "Go", LinesTotal: 3},
	}
	d := New(config.Defaults())
	res, err := d.Detect(context.Background(), 1, root, files, nil)
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(res.Groups) != 0 {
		t.Fatalf("expected no groups, got %d", len(res.Groups))
	}
}

func TestMergeRuns(t *testing.T) {
	got := mergeRuns([]int{0, 1, 2, 10}, 6)
	if len(got) != 2 {
		t.Fatalf("expected 2 runs, got %d: %+v", len(got), got)
	}
	if got[0].lo != 0 || got[0].hi != 7 {
		t.Fatalf("run 0 wrong: %+v", got[0])
	}
	if got[1].lo != 10 || got[1].hi != 15 {
		t.Fatalf("run 1 wrong: %+v", got[1])
	}
}
