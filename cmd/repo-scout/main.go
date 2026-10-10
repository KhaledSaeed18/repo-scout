// Command repo-scout runs Repo Scout. With no command, or with serve, it runs
// the server: the API, the scan workers and, in release builds, the web
// interface. scan analyzes one repository and prints a report, for scripts
// and CI.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
)

// version is stamped at build time with -ldflags "-X main.version=...";
// otherwise it is resolved from the module version (see resolveVersion).
var version = "dev"

// Exit codes. Scripts and CI rely on them.
const (
	exitOK         = 0
	exitGateFailed = 1 // scan: a quality gate failed
	exitUsage      = 2 // bad command line
	exitError      = 3 // the work itself failed
)

const usage = `Repo Scout: local-first analytics for any Git repository.

Usage:
  repo-scout [serve] [--addr host:port] [--db file]
  repo-scout scan [flags] [path]
  repo-scout version

Run "repo-scout <command> -h" for a command's flags.
Documentation: https://github.com/KhaledSaeed18/repo-scout
`

func main() {
	info, ok := debug.ReadBuildInfo()
	version = resolveVersion(version, info, ok)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// resolveVersion keeps a version stamped by the release build, and otherwise
// uses the module version Go records in binaries built with go install
// (v1.2.0 becomes 1.2.0, matching release builds). Source builds stay "dev".
func resolveVersion(stamped string, info *debug.BuildInfo, ok bool) string {
	if stamped != "dev" || !ok || info == nil {
		return stamped
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return strings.TrimPrefix(v, "v")
	}
	return stamped
}

// run dispatches to a command and returns the process exit code.
func run(args []string, stdout, stderr io.Writer) int {
	cmd := "serve"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	if len(args) > 0 && cmd == "serve" {
		switch args[0] {
		case "-v", "-version", "--version":
			cmd, args = "version", nil
		case "-h", "-help", "--help":
			cmd, args = "help", nil
		}
	}
	switch cmd {
	case "serve":
		return serve(args, stderr)
	case "scan":
		return scan(args, stdout, stderr)
	case "version":
		_, _ = fmt.Fprintf(stdout, "repo-scout %s\n", version)
		return exitOK
	case "help":
		_, _ = fmt.Fprint(stdout, usage)
		return exitOK
	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, usage)
		return exitUsage
	}
}
