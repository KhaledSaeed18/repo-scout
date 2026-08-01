# Repo Scout

Local-first Git repository analytics platform. Analyze any repository on disk
and get an interactive dashboard: architecture, code quality, dependencies,
commit history, contributors, and duplicate code — all offline.

## Features

- Repository scanner that handles 10-file projects up to 100k-file monorepos
- Language detection (Go, Rust, Python, Java, Kotlin, TypeScript, JavaScript,
  PHP, C#, C++, C, Swift) with LOC / comments / blank-line breakdown
- Git analytics: commit frequency, contributors, streaks, heatmap, largest
  commits, merges, file ownership
- Dependency graphs (package.json, go.mod, Cargo.toml, pom.xml, composer.json,
  requirements.txt)
- Lazy-loaded, searchable, virtualized file tree
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

## Requirements

- Go 1.22+
- Node 20+
- `git` on PATH (used for history analysis)
- Make

## Quick start

```sh
make dev
```

Open http://localhost:5173. The API runs on http://localhost:8080.

## Commands

- `make dev` — backend + frontend together
- `make backend` — build/run the Go API only
- `make frontend` — Vite dev server only
- `make test` — Go tests + frontend typecheck/tests
- `make lint` — go vet + golangci-lint (if present) + frontend eslint
- `make build` — production builds

## Documentation

- `docs/rules.md` — engineering and git rules
- `docs/architecture.md` — system design
- `docs/roadmap.md` — delivery plan
