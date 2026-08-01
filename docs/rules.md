# Engineering & Git Rules

These rules are hard requirements. They apply to every change in this
repository, including every micro-commit.

## Git rules (hard)

- **Micro-commits**: every small, self-contained unit of work gets its own
  commit immediately. One type, one file group, one behavior, one doc update.
  Never batch unrelated changes into one commit. Commits must be small enough
  to review at a glance.
- **Push cadence**: commit locally as you go. Push to `origin main` when a
  feature or a unit of work is complete. One push may carry multiple
  micro-commits.
- **Branching**: everything goes directly to `main`. No branches, no PRs for
  now.
- **Commit messages**: conventional-commit style.
  - Format: `type(scope): summary`
  - Imperative mood, lower case, no trailing period.
  - Body only when the *why* is not obvious from the diff.
  - Allowed types: `feat`, `fix`, `docs`, `chore`, `test`, `refactor`,
    `perf`, `style`.
- **Strictly forbidden in commits**:
  - co-author trailers
  - "Generated with" notes
  - any mention of an AI, agent, assistant, or tool as author/contributor.
  - The human is the sole author of record.
- **Never force-push. Never rewrite pushed history.**

## Engineering rules

- **Clean architecture**: keep `cmd/` (composition root) thin. Business logic
  lives in `internal/` behind interfaces. No business rules in HTTP handlers.
- **Dependency injection**: construct dependencies in the composition root and
  pass them down. No global singletons or package-level mutable state.
- **Modular design**: one concern per package. Packages must not import each
  other's internals; depend on interfaces, not concrete types.
- **Errors**: wrap errors with context (`fmt.Errorf("...: %w", err)`). Do not
  swallow errors. Log at the boundary.
- **No secrets**: never commit secrets, tokens, or keys. Use the `.env` files
  which are gitignored.
- **No dead code**: every exported symbol must be used or documented.
- **Concurrency**: goroutines must be cancellable via `context.Context`. Track
  progress and report it. Worker pools must handle pause/resume/cancel.
- **Performance**: the platform must stay responsive on a 100k-file repository
  and respect configurable memory limits. Prefer SQLite indexes and streaming
  over loading everything into RAM.
- **No comments unless they explain why.** Prefer self-documenting names.
- **No placeholder UI.** Every shipped page must be functional and
  production-looking.

## Code quality gates

- Go: `gofmt`, `go vet`, `golangci-lint` when available, `go test ./...`.
- Frontend: `tsc --noEmit`, `eslint`.
- Before pushing a unit of work, the affected component's tests must pass.
