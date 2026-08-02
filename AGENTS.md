# AGENTS.md

Instructions for any agent or tool working in this repository. Read this file
first, before making changes.

## Project

Repo Scout — a local-first Git repository analytics platform.

- Backend: Go + chi + SQLite (GORM). Everything offline.
- Frontend: React + Vite + TypeScript + Tailwind + shadcn/ui + TanStack Query
  + React Flow + Recharts.

## Commands

From the repository root:

- `make dev` — run backend + frontend together (single command).
- `make backend` — build/run the Go API.
- `make frontend` — run the Vite dev server.
- `make test` — Go tests, frontend typecheck + tests.
- `make lint` — go vet, golangci-lint (if present locally; always enforced in CI),
  frontend eslint.
- `make build` — production builds for both halves.

Frontend commands run with `pnpm --prefix frontend run ...`.

## Layout

- `cmd/api` — thin composition root; wires dependencies, starts the server.
- `internal/*` — one package per concern. Packages depend on interfaces, not
  each other's internals.
- `frontend/` — the React app. Pages in `frontend/src/pages`, UI primitives in
  `frontend/src/components/ui` (shadcn). API + WebSocket clients in
  `frontend/src/lib`.
- `testdata/` — git repository fixtures used by integration tests.

## Conventions

- Clean architecture + dependency injection. No package-level mutable state.
- Errors wrapped with `%w`. Goroutines cancellable via `context.Context`.
- Every feature that scans or analyzes must report progress through the job
  system and broadcast via the WebSocket hub.
- Big scans must stay memory-bounded: stream, batch-insert, and index.
- SQLite: batched inserts during scans; indexes on hot query columns.
- No placeholder UI, no unfinished pages, no dead exports.
- TypeScript: strict mode, shared types generated alongside the API contracts
  in `frontend/src/lib/types.ts`.

## Git rules

- Micro-commits: one small self-contained unit per commit, committed
  immediately. Never batch unrelated changes.
- Conventional commits: `type(scope): summary`, imperative, lower case, no
  trailing period.
- Everything to `main` directly. No branches, no PRs.
- Push when a feature/unit of work completes.
- Never rewrite pushed history. Never force-push.
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

Both must pass before merging.
