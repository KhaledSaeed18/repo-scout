## What this changes

<!-- One or two sentences on what changes for users or contributors, and why. -->

Closes #

## How it was tested

<!-- Tests added or updated, and anything checked by hand (which repository you scanned, which pages). -->

## Screenshots

<!-- For visible changes: before and after, light and dark theme, and 390px wide. Delete if not applicable. -->

## Checklist

- [ ] Commits follow [Conventional Commits](https://www.conventionalcommits.org) and each one builds and passes tests.
- [ ] Bug fixes include a test that fails without the fix.
- [ ] `make check` passes (gofmt, go vet, golangci-lint, Go tests with `-race`, frontend typecheck, lint and tests), and `make e2e` for interface changes.
- [ ] API changes are described in `internal/api/openapi.yaml` and mirrored in `frontend/src/lib/types.ts`.
- [ ] User-visible changes have a line under `[Unreleased]` in `CHANGELOG.md`.
- [ ] UI changes use the design tokens and page components, and work with the keyboard, at 390px, and in both themes.
- [ ] Docs (README, CONTRIBUTING, AGENTS.md) are updated if behavior or setup changed.
