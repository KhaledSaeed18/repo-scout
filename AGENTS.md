# AGENTS.md

Instructions for any agent or tool working in this repository. Read this file
first, before making changes.

## Project

Repo Scout — a local-first Git repository analytics platform.

- Backend: Go + chi + SQLite (GORM). Everything offline.
- Frontend: React + Vite + TypeScript + Tailwind + shadcn/ui (Base UI) +
  TanStack Query + React Flow. Charts are plain SVG/CSS; Vitest for unit tests.

## Commands

From the repository root:

- `make dev` — run backend + frontend together (single command).
- `make backend` — build/run the Go API.
- `make frontend` — run the Vite dev server.
- `make test` — Go tests, frontend typecheck + tests.
- `make check` — everything CI checks except the browser tests. Run it before
  every commit.
- `make e2e` — build the binary and run the Playwright end-to-end suite
  (`frontend/e2e`): every page, both themes at 390px, and key flows.
- `make lint` — go vet, golangci-lint (if present locally; always enforced in CI),
  frontend oxlint.
- `make build` — one `bin/repo-scout` with the frontend embedded (`-tags embedui`).

Frontend commands run with `cd frontend && pnpm run ...`.

## Layout

- `cmd/repo-scout` — thin composition root; wires dependencies, starts the server.
- `internal/webui` — the built frontend, embedded only when built with
  `-tags embedui` (see `make build`).
- `internal/*` — one package per concern. Packages depend on interfaces, not
  each other's internals.
- `frontend/` — the React app. Pages in `frontend/src/pages`, grouped as the
  sidebar is: `history/`, `code/`, `structure/`. Shared page building blocks in
  `frontend/src/components` (`layout.tsx`, `states.tsx`, `RequireRepo.tsx`), UI
  primitives in `frontend/src/components/ui` (shadcn). API + WebSocket clients
  and pure, unit-tested helpers in `frontend/src/lib`.
- `brand/` — logo, icon and social preview sources (SVG) with `build.sh` to
  render every PNG/ICO. Edit the SVGs, then rebuild; never hand-edit the PNGs.

## Conventions

- Clean architecture + dependency injection. No package-level mutable state.
- Errors wrapped with `%w`. Goroutines cancellable via `context.Context`.
- Every feature that scans or analyzes must report progress through the job
  system and broadcast via the WebSocket hub.
- Big scans must stay memory-bounded: stream, batch-insert, and index.
- SQLite: batched inserts during scans; indexes on hot query columns.
- Schema: additive changes on the models (AutoMigrate); everything else is a
  new numbered migration in `internal/database/migrations.go`. Never edit a
  shipped migration.
- No placeholder UI, no unfinished pages, no dead exports.
- API contract: `internal/api/openapi.yaml` describes every endpoint; contract
  tests check routes and responses against it. `frontend/src/lib/types.ts`
  mirrors it. Change handler, spec and types together. Lists are `[]`, never
  `null`.
- TypeScript: strict mode.
- Pure frontend logic (formatting, calendar, graph layout, highlighting) lives
  in `frontend/src/lib/*.ts` with a `*.test.ts` beside it.

## Interface design

The UI follows a "survey map" system. Keep new work inside it:

- Colors come only from the tokens in `frontend/src/index.css` (light and dark).
  Ultramarine `primary` marks interactive things; the `contour-*` ramp is for
  activity intensity; `chart-*` for categories. No hard-coded colors.
- Type: Barlow for UI, Barlow Semi Condensed for headings and figures,
  JetBrains Mono only for code, paths and hashes. Numbers use tabular figures.
- Pages are a `PageHeader` followed by `Section`s, not grids of cards. Data
  states go through `QueryView` / `Loading` / `ErrorNotice` / `Empty`, and
  repository pages are wrapped in `RequireRepo`.
- Copy is sentence case and plain. No all-caps labels, no "A · B" meta strings.
  Empty states say what to do next; errors say what failed.
- Every page must work at 390px wide and in both themes.
- The in-app logo is `BrandMark` (theme colors); follow `brand/README.md` for
  any other use of the mark.

## Git rules

These rules are for the maintainer and agents working in this repository.
Outside contributors work in forks and open pull requests, as described in
`CONTRIBUTING.md`; keep that guide in sync when conventions here change.

- Micro-commits: one small self-contained unit per commit, committed
  immediately. Never batch unrelated changes.
- Conventional commits: `type(scope): summary`, imperative, lower case, no
  trailing period.
- Everything to `main` directly. No branches, no PRs.
- Push when a feature/unit of work completes.
- Never rewrite pushed history. Never force-push.
- Releases are `vX.Y.Z` tags pushed by the maintainer only; the release
  workflow builds and publishes everything. Never tag without being asked.
- No co-author trailers, no "Generated with", no mention of AI or tools as
  authors. The human is the sole author of record.

## Definition of done for a change

1. `gofmt` clean and `go vet ./...` passes (backend changes).
2. Affected package tests pass (`go test ./...`).
3. Frontend typecheck passes (`tsc --noEmit`).
4. Change is committed as its own micro-commit with a conventional message.
5. The feature/unit is pushed to `origin main` once complete.
6. CI (`.github/workflows/ci.yml`) is green on `main`.

## CI

`.github/workflows/ci.yml` runs on every push and pull request against `main`:

- **Backend**: `gofmt` check, `go vet`, `golangci-lint` (config in
  `.golangci.yml`), `go build ./...`, `go test ./... -race` with coverage.
- **Frontend**: `pnpm install --frozen-lockfile`, typecheck, lint, test, build.
- **Single binary and end-to-end**: `make build` with the interface embedded,
  a smoke test that the binary serves the API and the interface, then the
  Playwright suite against it.

All must pass before merging.
