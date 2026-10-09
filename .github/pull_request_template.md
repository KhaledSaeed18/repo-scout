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
- [ ] `gofmt -l .` prints nothing, and `go vet ./...`, `golangci-lint run ./...` and `go test ./... -race` pass.
- [ ] `pnpm --prefix frontend run typecheck`, `lint`, `test` and `build` pass.
- [ ] API changes are mirrored in `frontend/src/lib/types.ts`.
- [ ] UI changes use the design tokens and page components, and work with the keyboard, at 390px, and in both themes.
- [ ] Docs (README, CONTRIBUTING, AGENTS.md) are updated if behavior or setup changed.
