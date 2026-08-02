<div align="center">

<img src="https://shieldcn.dev/header/graph.svg?title=Repo%20Scout&subtitle=Local-first%20analytics%20for%20any%20Git%20repository&theme=cyan&logo=https%3A%2F%2Fraw.githubusercontent.com%2FKhaledSaeed18%2Frepo-scout%2Fmain%2Ffrontend%2Fpublic%2Fapple-touch-icon.png&size=lg&align=center" width="820" alt="Repo Scout" />

<p>
  <img src="https://shieldcn.dev/badge/backend-Go%20%2B%20SQLite-cyan.svg?variant=secondary&logo=go&logoColor=ffffff" alt="Backend: Go + SQLite" />
  <img src="https://shieldcn.dev/badge/frontend-React%20%2B%20Vite-cyan.svg?variant=secondary&logo=react&logoColor=ffffff" alt="Frontend: React + Vite" />
  <img src="https://shieldcn.dev/badge/mode-local--first-cyan.svg?variant=secondary" alt="Mode: local-first" />
  <a href="https://github.com/KhaledSaeed18/repo-scout/actions/workflows/ci.yml"><img src="https://shieldcn.dev/github/ci/KhaledSaeed18/repo-scout.svg?workflow=ci.yml&branch=main&variant=secondary" alt="CI status" /></a>
</p>

<strong>Point it at a folder. See the whole codebase.</strong>

</div>

Repo Scout scans any Git repository on disk and turns it into an interactive
dashboard: architecture graphs, code quality metrics, dependency trees,
commit history, contributor activity, and duplicate code, all computed
locally and kept in a SQLite file next to it.

There is no upload step and no account. You give it a path, it walks the
tree and the git log, and the result is a set of pages you can actually
click through instead of a wall of terminal output.

## Why Repo Scout

- **One scan, the whole picture.** Architecture, metrics, dependencies, git
  history, and duplicates all come from a single scan, not five different
  tools with five different output formats to reconcile.
- **It never leaves your machine.** No external APIs, no telemetry, no
  uploading source code anywhere. The Go backend and the SQLite database sit
  on disk; the frontend just talks to `localhost`.
- **Built for real repository sizes.** The scanner streams and batch-inserts
  instead of holding everything in memory, so a 10-file toy project and a
  100k-file monorepo go through the same pipeline.
- **You watch it work, not wait for it.** Every scan is a background job
  broadcasting progress over a WebSocket, with pause / resume / cancel and
  crash recovery, not a frozen progress bar.
- **It tells you what's actually wrong.** Circular imports, dead files,
  unused modules, duplicated blocks with similarity scores, and complexity
  hot spots, surfaced directly, not left for you to infer from a diagram.

## Features

- Repository scanner that handles 10-file projects up to 100k-file monorepos
- Language detection (Go, Rust, Python, Java, Kotlin, TypeScript, JavaScript,
  PHP, C#, C++, C, Swift) with LOC / comments / blank-line breakdown
- Git analytics: commit frequency, contributors, streaks, heatmap, largest
  commits, merges, file ownership
- Dependency graphs (package.json, go.mod, Cargo.toml, pom.xml, composer.json,
  requirements.txt)
- Lazy-loaded, searchable, virtualized file tree, with a local folder picker
  to choose a repository to scan
- Instant indexed search: filename, folder, extension, content, regex,
  case-sensitive, whole-word
- Duplicate code detection with similarity scores and line highlighting
- Architecture graphs (folder / module / import) with circular dependency
  detection, unused modules, and dead files, exportable as SVG
- Metrics: cyclomatic complexity, function length, nesting, imports/exports,
  largest and most complex files
- Background jobs with pause / resume / cancel, queue, worker pool, progress,
  and crash recovery
- Live WebSocket updates
- CSV/JSON export, dark mode, resizable panels

## How it works

A scan runs as a background job through ordered stages, each reporting
progress over the WebSocket hub as it goes:

1. **git metadata**: branches, tags, HEAD, remote, top-level commit stats
2. **file scan**: walk the tree, apply ignore rules + size limits, count
   LOC/language per file with a worker pool
3. **git history**: commits, authors, contributor rollups, streaks, heatmap,
   file ownership
4. **dependencies**: parse manifests per language
5. **import graph**: resolve imports, detect cycles (Tarjan SCC), flag
   unused modules and dead files
6. **metrics**: cyclomatic complexity, function length, nesting, largest /
   most complex files
7. **duplicates**: shingle-hash normalized lines, cluster similar blocks
8. **content index**: populate SQLite FTS5 for instant search

Jobs are queued, run through a configurable worker pool, and persist their
state: pause, resume, and cancel all just mutate that state, and a crash
mid-scan resumes cleanly instead of leaving a half-written repository behind.

## Requirements

- Go 1.26+
- Node 22.13+ and pnpm 11+
- `git` on PATH (used for history analysis)
- Make

## Quick start

```sh
make dev
```

Open http://localhost:5173. The API runs on http://localhost:8080.

## Commands

- `make dev`: backend + frontend together
- `make backend`: build/run the Go API only
- `make frontend`: Vite dev server only
- `make test`: Go tests + frontend typecheck/tests
- `make lint`: go vet + golangci-lint (if present) + frontend eslint
- `make build`: production builds

## Architecture

```
repo-scout/
├── cmd/api/main.go        # composition root (thin)
├── internal/
│   ├── config/            # settings, env/flags, defaults, ignore rules
│   ├── models/             # GORM models (db schema)
│   ├── database/          # SQLite connection + migrations + indexes
│   ├── langdetect/         # language detection + LOC counting
│   ├── scanner/            # filesystem walker (files, folders, sizes)
│   ├── gitrepo/            # git history analysis (CLI-backed)
│   ├── deps/               # dependency manifest parsers
│   ├── metrics/            # complexity + structural metrics
│   ├── duplicates/         # similarity / duplicate block detection
│   ├── architecture/       # import graphs, cycles, unused/dead files
│   ├── search/             # indexed search (filename/folder/ext/content/regex)
│   ├── analysis/           # orchestrates a scan pipeline (stages)
│   ├── jobs/               # background job queue, worker pool, pause/resume/cancel
│   ├── ws/                 # WebSocket hub + event bus
│   ├── api/                # chi router, HTTP handlers, REST + WS endpoints
│   └── exports/            # CSV/JSON exporters
├── frontend/               # React + Vite + TS + Tailwind + shadcn/ui
├── testdata/               # repository fixtures for tests
└── scripts/dev.sh          # single-command run
```

| Concern         | Choice                                          |
|-----------------|--------------------------------------------------|
| Backend         | Go (chi router)                                   |
| Database        | SQLite via GORM, FTS5 for content search          |
| Git parsing     | `git` CLI via `os/exec`, cached in SQLite         |
| Background jobs | Custom job manager (queue + pool + persistence)   |
| Real time       | gorilla/websocket hub                             |
| Frontend        | React + Vite + TypeScript + Tailwind + shadcn/ui  |
| Data fetching   | TanStack Query                                    |
| Graphs          | React Flow                                        |
| Charts          | Recharts                                          |
