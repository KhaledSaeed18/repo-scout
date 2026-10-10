// Command repo-scout runs Repo Scout. With no command, or with serve, it runs
// the server: the API, the scan workers and, in release builds, the web
// interface. scan analyzes one repository and prints a report, for scripts
// and CI.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// version is stamped at build time with -ldflags "-X main.version=...".
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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
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
