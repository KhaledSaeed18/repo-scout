package coupling

import "testing"

func TestConventionLink(t *testing.T) {
	cases := []struct {
		a, b string
		want Link
	}{
		{"internal/api/server.go", "internal/api/data.go", LinkPackage},
		{"internal/api/server.go", "internal/jobs/jobs.go", ""},
		{"internal/scanner/scanner.go", "internal/scanner/scanner_test.go", LinkPackage},
		{"src/lib/format.ts", "src/lib/format.test.ts", LinkTest},
		{"tests/unit/api.spec.js", "src/api.js", LinkTest},
		{"pkg/test_util.py", "pkg/util.py", LinkTest},
		{"src/Parser.java", "test/ParserTest.java", LinkTest},
		{"frontend/package.json", "frontend/pnpm-lock.yaml", LinkLockfile},
		{"package.json", "frontend/pnpm-lock.yaml", ""},
		{"go.mod", "go.sum", LinkLockfile},
		{"CLAUDE.md", ".github/copilot-instructions.md", ""},
		{"Test.java", "Other.java", ""},
	}
	for _, c := range cases {
		if got := conventionLink(c.a, c.b); got != c.want {
			t.Errorf("conventionLink(%q, %q) = %q, want %q", c.a, c.b, got, c.want)
		}
	}
}
