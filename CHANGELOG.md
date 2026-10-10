# Changelog

All notable changes to Repo Scout are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and Repo Scout
follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html); see
[Versioning](README.md#versioning) for what counts as its public interface.

## [Unreleased]

## [1.0.0] - 2026-10-10

The first public release. Repo Scout scans any Git repository on your
machine and turns it into an interactive dashboard of its structure, quality,
history and risks, with nothing uploaded anywhere. It ships as one binary for
macOS, Linux and Windows that serves the web interface, and as a command line
that brings the same analysis to CI.

### Highlights

- **One scan, the whole picture.** Architecture, metrics, dependencies,
  history, ownership, duplicates and change coupling come from a single scan,
  computed locally and stored in SQLite.
- **Risk you can act on.** Hotspots rank complex files that keep changing,
  Knowledge shows the bus factor and code whose main author has gone
  inactive, and Change coupling exposes files that change together with
  nothing in the code linking them.
- **Built for CI.** `repo-scout scan` writes text, JSON or SARIF and fails a
  build on quality gates. With `--base origin/main` it judges a pull request
  by what it changes rather than by the whole codebase.
- **Safe to rescan, cheap to rescan.** Results are swapped in only when a
  scan succeeds, and rescans diff only the commits they have not seen.

### Added

#### Analysis

- Repository scanner with ignore rules, size and depth limits, and a worker
  pool, streaming and batch-inserting so memory stays flat on large
  repositories.
- Language detection and line counts (code, comments, blank lines) for Go,
  Rust, Python, Java, Kotlin, TypeScript, JavaScript, PHP, C#, C, C++ and
  Swift.
- Cyclomatic complexity, function count and length, nesting, imports and
  exports per file. Go is measured with the standard library parser; other
  languages with language-aware patterns.
- Import graph for Go, JavaScript, TypeScript, Python, Rust, Java, Kotlin,
  C#, C, C++ and PHP, with circular dependencies between folders, unused
  folders, entry points and possibly dead files, exportable as SVG.
- Dependency manifests for npm, Go modules, Cargo, Maven, Composer and pip.
- Duplicate code detection with similarity scores and linked locations.
- Change coupling: pairs of files that keep changing in the same commits,
  explained by an import, a shared Go package, a test or a lockfile, or
  flagged as a hidden dependency.
- Indexed search by file name, folder, extension, text or regular expression,
  case-sensitive and whole-word.

#### History and people

- Commit calendar, hour-of-week punch card, streaks, largest commits, merges,
  branches and tags, all on each author's local clock.
- Contributors merged through `.mailmap`, and file ownership by the author
  with the most commits to each file, following renames. Bot accounts never
  count as owners.
- Knowledge: bus factor, main authors per folder, and the largest files whose
  main author has stopped committing.
- Hotspots: files ranked by complexity times change frequency over a window
  you choose.

#### Interface

- Pages for overview, activity, commits, contributors, knowledge, branches
  and tags, files, search, metrics, hotspots, duplicates, architecture, change
  coupling, dependencies, portfolio, repositories and settings.
- Portfolio of every scanned repository with size, change since the last
  scan, bus factor, inactive authors, cycles, hidden dependencies and top
  hotspot.
- Trends: every successful scan is kept, with sparklines and a table of how
  each headline number moved.
- Live progress over a WebSocket, light, dark and system themes, and layouts
  that work down to 390 pixels wide.
- CSV and JSON exports of files, commits and contributors.

#### Command line and CI

- `repo-scout serve` (the default) runs the server; `repo-scout scan` analyzes
  a folder without one; `repo-scout version` and `--version` print the
  version.
- Reports as text, JSON or SARIF 2.1.0 for GitHub code scanning.
- Quality gates: `--max-complexity`, `--max-duplicates` and
  `--fail-on cycles,hidden-coupling`.
- Comparison against a base ref with `--base`: files added and removed,
  complexity moves, new and resolved cycles and dependency changes, gated by
  `--max-complexity-increase` and `--fail-on new-cycles`.
- Documented exit codes: 0 passed, 1 a quality gate failed, 2 bad usage,
  3 the scan failed.

#### Operations

- Background jobs with a queue, worker pool, pause, resume and cancel, and
  recovery after a crash.
- Atomic rescans: results are built on the side and promoted in one
  transaction, so a cancelled or failed rescan leaves the previous results
  untouched.
- Incremental history: rescans reuse what the last scan stored and diff only
  new commits.
- Versioned database migrations; a database written by a newer version is
  refused instead of being downgraded.
- Structured logs as text or JSON (`REPO_SCOUT_LOG_FORMAT`,
  `REPO_SCOUT_LOG_LEVEL`), with per-request and per-job records.
- A local JSON API described in OpenAPI 3.0 and served at `/api/openapi.yaml`.
- Results stored in the user configuration folder by default, or wherever
  `REPO_SCOUT_DB` points.

### Security

- The server listens on loopback only and answers only requests addressed to
  `localhost`, `127.0.0.1` or `::1`, which blocks DNS rebinding; more names
  can be allowed with `REPO_SCOUT_ALLOWED_HOSTS`.
- State-changing requests must be JSON, and the WebSocket accepts only
  same-origin connections, so other websites cannot drive a local instance.
- The interface is served with a strict content security policy and refuses
  to be framed.
- CSV exports neutralize spreadsheet formulas in repository data.
- Scans only read: git is run read-only, and comparing against a base ref
  extracts it with `git archive` without touching the repository.
- Release archives carry checksums, an SBOM and signed build provenance.

[Unreleased]: https://github.com/KhaledSaeed18/repo-scout/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/KhaledSaeed18/repo-scout/releases/tag/v1.0.0
