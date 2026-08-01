# Architecture

Repo Scout is a local-first Git repository analytics platform. It scans any
repository on disk and produces an interactive dashboard for architecture,
code quality, dependencies, commits, contributors, and duplicate code.

Everything runs locally. No external APIs. Data lives in SQLite.

## High-level layout

```
repo-scout/
├── go.mod                 # Go module (module: github.com/KhaledSaeed18/repo-scout)
├── cmd/api/main.go        # composition root (thin)
├── internal/
│   ├── config/            # settings, env/flags, defaults, ignore rules
│   ├── models/            # GORM models (db schema)
│   ├── database/          # SQLite connection + migrations + indexes
│   ├── langdetect/        # language detection + LOC counting
│   ├── scanner/           # filesystem walker (files, folders, sizes)
│   ├── gitrepo/           # git history analysis (CLI-backed)
│   ├── deps/              # dependency manifest parsers
│   ├── metrics/           # complexity + structural metrics
│   ├── duplicates/        # similarity / duplicate block detection
│   ├── architecture/      # import graphs, cycles, unused/dead files
│   ├── search/            # indexed search (filename/folder/ext/content/regex)
│   ├── analysis/          # orchestrates a scan pipeline (stages)
│   ├── jobs/              # background job queue, worker pool, pause/resume/cancel
│   ├── ws/                # WebSocket hub + event bus
│   ├── api/               # chi router, HTTP handlers, REST + WS endpoints
│   └── exports/           # CSV/JSON exporters
├── frontend/              # React + Vite + TS + Tailwind + shadcn/ui
├── testdata/              # repository fixtures for tests
├── scripts/dev.sh         # single-command run
└── Makefile
```

## Technology choices

| Concern          | Choice                                            | Why |
|------------------|---------------------------------------------------|-----|
| Backend          | Go (chi router)                                   | Fast, great concurrency, static binary |
| Database         | SQLite via GORM                                   | Embedded, zero-setup, offline |
| Git parsing      | `git` CLI via `os/exec` (cached in SQLite)        | Robust on huge repos (50 GB+), faster than pure-Go parsers |
| File scanning    | Custom walker + worker pool                       | Full control over ignore rules, memory, progress |
| Background jobs  | Custom job manager (queue + pool + persistence)   | Pause/resume/cancel + crash recovery |
| Real time        | gorilla/websocket hub                             | Live progress and dashboard updates |
| Frontend         | React + Vite + TS + Tailwind + shadcn/ui          | Production-quality UI |
| Data fetching    | TanStack Query                                    | Caching, invalidation, pagination |
| Graphs           | React Flow                                        | Interactive node/edge graphs |
| Charts           | Recharts                                          | Dashboard + git heatmaps |

## Data model (SQLite)

- `repositories` — id, name, path, status, git info, summary stats.
- `files` — id, repo_id, path, name, folder, extension, language, size_bytes,
  lines_total, lines_code, lines_comments, lines_blank, complexity, author,
  created_at, modified_at (indexed on repo_id+path, repo_id+extension).
- `commits` — id, repo_id, hash, author, email, timestamp, message,
  files_changed, insertions, deletions, is_merge.
- `branches` / `tags` — name + commit hash per repo.
- `contributors` — name, email, commits, first/last activity (rolled up).
- `file_ownership` — repo_id, file path, primary author, ownership share.
- `dependencies` — repo_id, file_path, manager type, name, version, scope.
- `import_edges` — repo_id, from_file, to_file, resolved, import_type
  (for the architecture/import graph).
- `duplicate_groups` / `duplicate_blocks` — groups of duplicated blocks with
  similarity scores and per-file line ranges.
- `jobs` — id, repo_id, kind, status, progress, total, current, error,
  timestamps. **Status is persisted so jobs recover after a crash.**
- `settings` — key/value user preferences (ignore folders, extensions, max
  file size, worker count, theme).

SQLite FTS5 virtual tables back content search.

## Scan pipeline (analysis)

A repository scan runs as a background job through ordered stages:

1. **git metadata** — branches, tags, HEAD, remote, top-level commit stats.
2. **file scan** — walk the tree, apply ignore rules + size limits, count
   LOC/language per file with the worker pool, record into `files`.
3. **git history** — enumerate commits/authors, compute contributor rollups,
   streaks, heatmap, file ownership.
4. **dependencies** — parse manifests per language, write `dependencies`.
5. **import graph** — extract imports per file, resolve to targets, run cycle
   detection (Tarjan SCC), flag unused modules and dead files.
6. **metrics** — cyclomatic complexity, function length, nesting, largest/
   most complex files, deepest folder.
7. **duplicates** — shingle-hash normalized lines, cluster similar blocks,
   compute similarity scores.
8. **content index** — populate FTS5 for instant search.

Each stage reports progress to the job; the WebSocket hub broadcasts it. Any
stage can be skipped via a config/settings toggle.

## Background jobs

- **Queue**: jobs are inserted with `status=queued`.
- **Worker pool**: N workers pull the next queued job. N from settings.
- **Pause**: signals workers to stop picking new items and pauses the current
  stage loop at a safe checkpoint (progress preserved in `jobs`).
- **Resume**: re-enqueues the paused job.
- **Cancel**: sets `status=cancelling`, workers drain and stop; partial data is
  deleted for that repo.
- **Crash recovery**: on startup, jobs stuck in `running` are marked
  `interrupted` and, unless configured otherwise, re-queued. Scan stages are
  idempotent per repo (delete repo rows before re-running).

## Real-time updates

`/api/ws` upgrades to a WebSocket. The hub fans out event messages:

- `job.progress`
- `job.state_changed` (queued/running/paused/cancelled/completed/failed)
- `repository.updated` (summary changed after a stage)
- `repository.completed`

The frontend subscribes once and invalidates the relevant TanStack Query keys.

## Search engine

- **File metadata index** (files table): filename, folder, extension, language,
  size — supports `filename`, `folder`, `extension` query modes instantly via
  indexed columns.
- **Content index** (FTS5): tokenized file content for instant substring
  search.
- **Regex mode**: after the FTS5 prefilter (or over the whole repo when
  required), content is scanned in Go with `regexp`. Respects
  `max-search-files` to bound work. Case-sensitivity and whole-word flags are
  applied at query time.

## API surface (REST)

```
POST   /api/repositories                 create/scan a repo from a local path
GET    /api/repositories                 list scanned repos
GET    /api/repositories/{id}            repo detail + summary
DELETE /api/repositories/{id}            remove repo + its data
GET    /api/repositories/{id}/files      paginated file list (filters: folder, ext, language, sort)
GET    /api/repositories/{id}/tree       lazy file tree (children per folder)
GET    /api/repositories/{id}/commits    commit list + stats
GET    /api/repositories/{id}/contributors
GET    /api/repositories/{id}/heatmap    commit heatmap (day x hour) + streaks
GET    /api/repositories/{id}/dependencies
GET    /api/repositories/{id}/duplicates
GET    /api/repositories/{id}/architecture   nodes/edges + cycles + unused/dead
GET    /api/repositories/{id}/metrics
GET    /api/repositories/{id}/svg?kind=...   export graph as SVG
GET    /api/search?repo=...&query=...&mode=...
GET    /api/jobs                           list jobs
POST   /api/jobs/{id}/pause|resume|cancel
GET    /api/settings                       PUT /api/settings
GET    /api/export/{id}?kind=files|commits|contributors&format=csv|json
GET    /api/health
GET    /api/ws                              websocket upgrade
```

## Frontend pages

- **Dashboard** — cards, charts, tables with sorting/filtering, CSV/JSON export.
- **Scan** — pick a folder, start a job, live progress, pause/resume/cancel.
- **Git** — commit frequency, contributors, heatmap, largest commits, merges,
  ownership, contribution graph, streaks.
- **Files** — lazy-loaded, searchable, virtualized file tree (size, language,
  last modified, git author, git changes).
- **Search** — file/content search with all modes.
- **Duplicates** — similarity table, highlight duplicated blocks.
- **Architecture** — React Flow graph (folder/module/import), cycle +
  unused/dead lists, SVG export.
- **Metrics** — complexity, largest/most complex files, deepest folder.
- **Dependencies** — graph of manifests + packages.
- **Settings** — ignore rules, limits, worker count, theme.

All pages are functional. No placeholders.

## Performance strategy

- SQLite indexes on the hot query columns; batched inserts during scans.
- Worker pool bounds file processing; configurable `max-file-size` skips
  binary/giant files.
- LOC/analysis per file is streamed, not held in memory.
- The file tree is lazy: children loaded per folder on demand.
- Search results are paginated; regex search is bounded.
- Duplicate detection is O(lines) with hash-based shingles and a lower bound
  on block size to keep the map small.
- Jobs persist their state so a restart does not lose progress.
