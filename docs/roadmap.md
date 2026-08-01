# Roadmap

Delivery is incremental. Every unit of work lands on `main` as micro-commits
and is pushed when the feature it belongs to is complete. Each phase below is
a push cadence boundary.

## Phase 0 — planning (done)

Rules, architecture, roadmap, agent instructions, README, Makefile.

## Phase 1 — backend core

- Go module, config with defaults + settings
- GORM models + SQLite connection + migrations + indexes
- Language detection + LOC counting
- File scanner with worker pool + ignore rules + size limits

Push after each feature completes.

## Phase 2 — jobs + git analytics

- Job queue, worker pool, pause/resume/cancel, crash recovery, progress
- Git history analysis: commits, contributors, streaks, heatmap, ownership
- Scan pipeline orchestrator + stage progress

Push.

## Phase 3 — static analysis

- Dependency manifest parsers (package.json, go.mod, Cargo.toml, pom.xml,
  composer.json, requirements.txt)
- Metrics: cyclomatic complexity, function length, nesting, imports/exports,
  largest/most complex files, deepest folder
- Duplicate code detection (shingle hashing + similarity)
- Architecture: import graph, Tarjan cycle detection, unused modules, dead
  files, SVG export

Push.

## Phase 4 — search + API + real-time

- Search engine (filename/folder/extension/content/regex, case, whole-word)
- WebSocket hub
- REST API: repositories, files, tree, commits, contributors, heatmap,
  dependencies, duplicates, architecture, metrics, jobs, settings, export
- Exporters (CSV/JSON)

Push.

## Phase 5 — backend tests + benchmarks

- Unit tests per package
- Integration tests (API + DB)
- Repository fixture tests (testdata/ git repos)
- Performance benchmarks (scan, duplicates, search)
- Memory-limit guard test

Push.

## Phase 6 — frontend

- Scaffold: Vite + TS + Tailwind + shadcn/ui + TanStack Query + React Flow +
  Recharts + router + theme
- Layout, nav, WebSocket live updates
- All pages: Dashboard, Scan, Git, Files, Search, Duplicates, Architecture,
  Metrics, Dependencies, Settings

Push.

## Phase 7 — integration + polish

- Single-command run (`make dev`)
- SVG graph export wired to frontend
- Final lint, typecheck, tests green
- README with usage
- Final push

## Acceptance checklist

- [ ] every page functional
- [ ] scanning works on repositories larger than 50 GB
- [ ] all graphs render correctly
- [ ] searching is instantaneous after indexing
- [ ] duplicate detection works
- [ ] exports work
- [ ] background workers recover from crashes
- [ ] all tests pass
- [ ] the application runs with a single command
