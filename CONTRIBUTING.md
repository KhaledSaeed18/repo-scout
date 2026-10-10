# Contributing to Repo Scout

Thanks for helping. This guide covers how to set up the project, how it is put
together, the conventions the code follows, and how to get a change merged.

By taking part you agree to follow the [code of conduct](CODE_OF_CONDUCT.md).
Security problems go through the [security policy](SECURITY.md), never public
issues.

## Contents

- [Ways to help](#ways-to-help)
- [Setting up](#setting-up)
- [How Repo Scout works](#how-repo-scout-works)
- [Backend conventions](#backend-conventions)
- [Frontend conventions](#frontend-conventions)
- [Testing and checks](#testing-and-checks)
- [Common changes, step by step](#common-changes-step-by-step)
- [Commits and pull requests](#commits-and-pull-requests)
- [Releases](#releases)

## Ways to help

- **Report a bug** with the [bug report form](https://github.com/KhaledSaeed18/repo-scout/issues/new?template=bug_report.yml).
  A repository that reproduces it (or its shape: size, languages, history
  length) helps most.
- **Suggest a feature** with the [feature request form](https://github.com/KhaledSaeed18/repo-scout/issues/new?template=feature_request.yml).
  Explain the question you are trying to answer about a codebase.
- **Fix something.** Issues labelled `good first issue` are small and well
  scoped. For anything larger, comment on the issue first so we can agree on
  the approach before you spend time on it.
- **Improve language support.** New languages, manifest formats and import
  resolvers are self-contained and very welcome; see
  [common changes](#common-changes-step-by-step).

## Setting up

### Requirements

| Tool | Version | Used for |
| --- | --- | --- |
| Go | 1.26+ | The API (`go.mod` pins the version) |
| Node.js | 22.13+ | The frontend toolchain |
| pnpm | 11 (`packageManager` in `frontend/package.json`) | Frontend dependencies |
| git | any recent | History analysis runs the `git` CLI |
| make | any | Task shortcuts |
| golangci-lint | v2 | Optional locally, always run in CI |

### First run

```sh
git clone https://github.com/<you>/repo-scout.git
cd repo-scout
(cd frontend && pnpm install)
make dev
```

`make dev` builds and starts the API on `127.0.0.1:8080` and the Vite dev
server on `http://localhost:5173`, which proxies `/api` (including the
WebSocket) to the API. Open the app, choose **Add a repository**, and scan
any Git folder; this repository itself is a good first target.

### Make targets

| Command | What it does |
| --- | --- |
| `make dev` | API and frontend together, with hot reload for the frontend |
| `make backend` | Build and run only the API |
| `make frontend` | Run only the Vite dev server |
| `make test` | Go tests, then frontend typecheck and unit tests |
| `make e2e` | `make build`, then the Playwright end-to-end tests against that binary (run `pnpm exec playwright install chromium` in `frontend/` once first) |
| `make lint` | `go vet`, golangci-lint if installed, and oxlint |
| `make fmt` | `gofmt -w` over the Go code |
| `make ui` | Build the frontend and stage it in `internal/webui/dist` for embedding |
| `make build` | `make ui`, then one `bin/repo-scout` binary with the interface embedded (`-tags embedui`) |

Frontend scripts can also be run directly with
`cd frontend && pnpm run <dev|typecheck|test|lint|build>`.

Plain `go build` produces an API-only binary, which is what `make dev` uses
with Vite serving the interface. Only builds tagged `embedui` carry the
interface, and they need `make ui` first.

### Configuration

| Variable | Default | Meaning |
| --- | --- | --- |
| `REPO_SCOUT_ADDR` | `127.0.0.1:8080` | Where the API listens. Keep it on loopback; the API has no authentication. |
| `REPO_SCOUT_DB` | `repo-scout/reposcout.db` in the user configuration folder; `data/reposcout.db` under `make dev` | SQLite database file. Delete it to start from scratch. |
| `REPO_SCOUT_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error`. Debug adds interface file and health check requests. |
| `REPO_SCOUT_LOG_FORMAT` | `text` | `text`, or `json` for log collectors. |
| `REPO_SCOUT_ALLOWED_HOSTS` | (none) | Extra host names the API answers to, comma separated. `localhost`, `127.0.0.1` and `::1` are always allowed. |

User preferences (ignored folders, size limits, workers, duplicate thresholds,
theme) live in the database and are edited on the Settings page.

Tip: point `REPO_SCOUT_DB` at a throwaway path such as
`/tmp/rs/reposcout.db` while developing so experiments do not touch your real
scan data.

## How Repo Scout works

```
browser ──HTTP/WS──▶ Vite proxy ──▶ chi router (internal/api)
                                        │
                     enqueue scan ──▶ jobs.Manager ──▶ analysis.Runner (stages)
                                        │                     │
                       WebSocket hub ◀──┘          SQLite via GORM (internal/database)
```

### Backend packages

| Package | Responsibility |
| --- | --- |
| `cmd/repo-scout` | Composition root and command line: `serve` wires dependencies and starts the server and worker pool; `scan` runs one analysis into a temporary database and writes a report. Keep it thin. |
| `internal/config` | Environment config and user settings with defaults and validation |
| `internal/database` | Opening the SQLite database, schema migrations, clearing and promoting scan results; settings store |
| `internal/models` | GORM models shared by all packages |
| `internal/api` | HTTP handlers, middleware, JSON helpers, SVG and export endpoints |
| `internal/jobs` | Persistent job queue, worker pool, pause, resume, cancel and crash recovery |
| `internal/analysis` | The scan pipeline: runs each stage and reports progress |
| `internal/scanner` | Walks the tree with ignore rules and limits, counts lines per file with a worker pool |
| `internal/langdetect` | Language table and line counting (code, comments, blank) |
| `internal/metrics` | Per-file complexity, function length, nesting, imports and exports |
| `internal/gitrepo` | Wraps the `git` CLI: metadata, branches, tags, streamed history |
| `internal/gitanalytics` | Heatmaps, streaks, ownership, largest commits over stored history |
| `internal/risk` | Hotspots (complexity times change frequency) and knowledge concentration |
| `internal/coupling` | Change coupling: files that change together, and why |
| `internal/report` | `repo-scout scan` reports in text, JSON and SARIF, and the quality gates |
| `internal/deps` | Manifest parsers (npm, Go, Cargo, Maven, Composer, pip) |
| `internal/architecture` | Import extraction and resolution, cycles (Tarjan), dead files, unused folders |
| `internal/duplicates` | Shingle hashing and clustering of similar blocks |
| `internal/search` | Filename, folder and extension queries and FTS5 content search |
| `internal/export` | CSV and JSON exports |
| `internal/ws` | WebSocket hub that broadcasts job and repository events |
| `internal/webui` | The built frontend, embedded only in `embedui` builds |

### A scan, end to end

1. `POST /api/repositories` with a path creates the repository row (or finds
   it) and asks `jobs.Manager.Enqueue` for a scan. A repository has at most one
   live scan; asking again returns the existing job.
2. A worker claims the oldest queued job and calls `analysis.Runner.Run`, which
   runs these stages in order: **git metadata**, **file scan** (line counts
   and metrics per file), **git history** (commits and the files each one
   changed), **dependencies**, **import graph**, **change coupling**,
   **duplicates**, and **content index** (SQLite FTS5). Results are written
   under a staging ID and swapped in with one transaction when every stage
   succeeds, along with a snapshot for the trend history; a failed or
   cancelled scan leaves the previous results untouched.
3. Stages report progress through the `jobs.Reporter` interface. The reporter
   throttles database writes and the manager broadcasts `job.progress` and
   `job.state_changed` events over the WebSocket. When the job ends the
   manager announces the repository so open pages refresh.
4. The frontend (`frontend/src/lib/ws.ts`) turns those events into TanStack
   Query invalidations.

Jobs survive restarts: running or paused jobs are re-queued at startup, and
jobs that were being cancelled are finished as cancelled.

### Frontend

```
frontend/src/
├── main.tsx          routes; pages are lazy-loaded
├── App.tsx           shell: sidebar, mobile menu, page error boundary
├── index.css         design tokens (light and dark)
├── pages/            Overview, Repositories, Settings,
│   ├── history/      Activity, Commits, Contributors, Knowledge, Branches & tags
│   ├── code/         Files, Search, Metrics, Hotspots, Duplicates
│   └── structure/    Architecture, Change coupling, Dependencies
├── components/       shared building blocks; ui/ holds shadcn primitives
└── lib/              API client and hooks, WebSocket, types, pure helpers + tests
```

## Backend conventions

- **Dependency injection, no globals.** Packages receive what they need
  (`*gorm.DB`, interfaces such as `jobs.Runner` and `jobs.EventSink`) through
  constructors. No package-level mutable state.
- **Errors** are wrapped with `%w` and carry context (`fmt.Errorf("load repo: %w", err)`).
  Handlers return JSON errors through `writeErr` with an honest status code:
  400 for bad input, 404 for missing records, 409 for invalid state changes,
  500 only for real failures.
- **Long work is cancellable.** Anything that loops over files or history
  takes a `context.Context` and checks it; scans call `Reporter.Checkpoint`
  so pause and cancel take effect.
- **Stay memory-bounded.** Stream (`git log` is read line by line), insert in
  batches (`CreateInBatches`), aggregate in SQL rather than loading whole
  tables, and add indexes for columns you filter or sort on.
- **Every analysis reports progress** through the job system.
- **Schema changes.** New tables, columns and indexes go on the models in
  `internal/models` and are added by `AutoMigrate`. Anything it cannot do
  (backfills, renames, drops, type changes) is a new numbered migration in
  `internal/database/migrations.go`. Never edit or renumber a shipped
  migration. A database written by a newer build is refused rather than
  downgraded.
- **SQL:** user text in `LIKE` must be escaped (see `escapeLike`), paged
  queries need a unique tie-breaker in `ORDER BY`, and new queries should be
  scoped by `repo_id`.
- **API contracts** are described in `internal/api/openapi.yaml` (served at
  `/api/openapi.yaml`) and mirrored by `frontend/src/lib/types.ts`. Contract
  tests fail when a route is missing from the spec or a response does not
  match its schema, so change the handler, the spec and the types together.
  Lists are always `[]`, never `null`.
- Format with `gofmt`; `go vet` and golangci-lint (config in `.golangci.yml`)
  must pass.

## Frontend conventions

- **TypeScript strict mode.** No `any`, no non-null assertions on data that
  can really be missing.
- **Data fetching** goes through hooks in `src/lib/api.ts`. Query keys must
  include every parameter the request depends on, starting with the
  repository ID.
- **Page structure:** a `PageHeader`, then `Section`s. Loading, error and
  empty states use `QueryView`, `Loading`, `ErrorNotice` and `Empty` from
  `src/components/states.tsx`. Pages that need a scanned repository are
  wrapped in `RequireRepo`.
- **Pure logic** (formatting, layout, parsing) goes in `src/lib/*.ts` with a
  `*.test.ts` beside it.
- **Design system.** Colors come only from the tokens in `src/index.css`;
  ultramarine `primary` marks interactive elements, `contour-*` shows activity
  intensity, `chart-*` separates categories. Barlow is the text face, Barlow
  Semi Condensed is for headings and figures, and JetBrains Mono is only for
  code, paths and hashes.
- **Copy** is sentence case and plain: say what something does, make empty
  states suggest the next step, and make errors say what failed. No all-caps
  labels.
- **Accessibility and layout.** Everything must work with the keyboard, have
  visible focus, and fit a 390px-wide screen in both light and dark themes.
- The logo inside the app is `BrandMark`. Brand files live in `brand/`; see
  [`brand/README.md`](brand/README.md).

## Testing and checks

Run everything CI runs before you push:

```sh
gofmt -l .                     # must print nothing
go vet ./...
golangci-lint run ./...
go test ./... -race
(cd frontend && pnpm run typecheck && pnpm run lint && pnpm run test && pnpm run build)
make e2e                       # builds the binary and runs the browser tests
```

What to test:

- **Bug fixes** start with a test that fails without the fix.
- **Go packages** use real SQLite in memory (`database.Open(":memory:")`) and
  real git repositories built in `t.TempDir()`; see the helpers at the top of
  existing `_test.go` files. Set author and committer dates explicitly when
  the result depends on time.
- **API handlers** are tested end to end through `httptest` in
  `internal/api/api_test.go`.
- **Frontend helpers** are tested with Vitest. Anything involving dates should
  pass in any timezone (try `TZ=America/Los_Angeles` and `TZ=Pacific/Kiritimati`).
- **Pages and flows** are covered by Playwright in `frontend/e2e`, against
  the real binary and a fixture repository built in `setup.e2e.ts`. A new
  page goes in `e2e/routes.ts`, which checks that it renders without errors
  and fits 390px in both themes; a new interaction gets a test in
  `flows.e2e.ts`.
- **UI changes:** check the page in both themes, at 390px wide, and with the
  keyboard, and add screenshots to the pull request.

## Common changes, step by step

### Add a language

1. Add an entry to `languages` in `internal/langdetect/languages.go` with its
   extensions, file names and comment syntax.
2. Add a case to the tests in `internal/langdetect` covering line counting.
3. If the language has functions and branches, register it in
   `internal/metrics/metrics.go` with an `add("Name", &langConfig{...})` call:
   function start patterns (`funcStart`), decision points such as `if` and
   loops (`decisions`), import and export patterns, and comment patterns.
   Cover it in `metrics_test.go`.

### Support a new manifest format

1. Add a parser in `internal/deps/deps.go`, a `Manager…` constant and a case
   in `Parse` keyed by the manifest's file name.
2. Map its dependency groups to the `production`, `development` and
   `indirect` scopes.
3. Add cases to `TestParse`, including a malformed file.
4. Give the ecosystem a display name in `frontend/src/pages/structure/Dependencies.tsx`.

### Resolve imports for a language

1. Extract import specifiers in `internal/architecture/imports.go` and list
   the language in `extractors` in `architecture.go`.
2. Resolve specifiers to files or folders in `builder.resolve` in
   `architecture.go`.
3. Add a graph test like `TestGoImportGraph` with a small fixture repository.

### Add an API endpoint

1. Write the handler in the matching file in `internal/api`, register it in
   `Router()`, and validate input with `parseID`, `queryInt` and `writeErr`.
2. Add a test in `internal/api/api_test.go`.
3. Describe it in `internal/api/openapi.yaml` and add a call to
   `TestResponsesMatchSpec` in `internal/api/openapi_test.go`.
4. Add the response type to `frontend/src/lib/types.ts`, a function to `api`,
   and a hook with a complete query key in `frontend/src/lib/api.ts`.

### Add a page

1. Create it in the matching folder under `frontend/src/pages`, wrapped in
   `RequireRepo` if it needs a repository.
2. Add a lazy route in `src/main.tsx` and an entry in the `groups` list in
   `src/components/Sidebar.tsx`.
3. Use `PageHeader`, `Section` and `QueryView`, and check it at 390px in both
   themes.

### Change the logo or icons

Edit the SVGs in `brand/`, then run `./brand/build.sh` to regenerate every PNG,
the favicons and the install icons. Never edit the generated PNGs by hand.

## Commits and pull requests

### Commits

Use [Conventional Commits](https://www.conventionalcommits.org): a type, an
optional scope, and an imperative, lower-case summary without a final period.

```
fix(search): reject invalid patterns as bad requests
feat(frontend): add a punch card to the activity page
docs(readme): describe the scan pipeline
```

Types: `feat`, `fix`, `perf`, `refactor`, `test`, `docs`, `style`, `build`,
`ci`, `chore`. Scopes are usually a Go package (`api`, `jobs`, `scanner`) or
`frontend`, `brand`, `readme`.

Keep commits small and self-contained: one logical change each, every commit
building and passing tests. Never mix unrelated changes in one commit.

### Pull requests

1. Fork the repository and create a branch from `main`
   (`fix/search-invalid-regex`, `feat/kotlin-imports`).
2. Make your change in focused commits, with tests.
3. Run the checks above.
4. Open a pull request against `main` and fill in the template. Link the issue
   it closes and add screenshots for anything visible.
5. CI must be green. A maintainer will review; expect questions and small
   requests, and push follow-up commits rather than force-pushing over review
   comments.

Pull requests are squash-merged or rebased onto `main`, so the final history
stays a clean sequence of conventional commits.

By contributing, you agree that your contributions are licensed under the
project's [MIT License](LICENSE).

## Releases

Releases are cut by the maintainer from `main` by pushing a semantic version
tag:

```sh
git tag -a v1.2.0 -m "v1.2.0"
git push origin v1.2.0
```

`.github/workflows/release.yml` then runs the tests, builds binaries for
macOS, Linux and Windows (amd64 and arm64) with GoReleaser
(`.goreleaser.yaml`), publishes them with checksums and a changelog grouped
from the conventional commits, pushes a multi-platform image to
`ghcr.io/khaledsaeed18/repo-scout`, and attests the provenance of every
artifact. Tags with a suffix such as `v1.2.0-rc.1` are published as
prereleases and do not move the `latest` image.

To try the release build locally: `goreleaser release --snapshot --clean`, or
`docker build -t repo-scout .` for the image.
