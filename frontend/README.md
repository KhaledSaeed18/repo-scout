# Repo Scout frontend

The React app for Repo Scout. It talks to the Go API on `localhost:8080`
through the Vite proxy (HTTP and the `/api/ws` WebSocket share one origin).

## Commands

```sh
pnpm install
pnpm run dev        # Vite dev server on :5173, proxies /api to :8080
pnpm run typecheck  # tsc in strict mode
pnpm run test       # Vitest unit tests for src/lib
pnpm run lint       # oxlint
pnpm run build      # typecheck + production build into dist/
```

From the repository root, `make dev` runs the API and this app together.

## Layout

```
src/
├── main.tsx              # routes (pages are lazy-loaded)
├── App.tsx               # shell: sidebar, mobile menu, page error boundary
├── index.css             # design tokens for light and dark themes
├── pages/
│   ├── Overview.tsx
│   ├── Repositories.tsx  # add, rescan, remove; scan history
│   ├── Settings.tsx
│   ├── history/          # Activity, Commits, Contributors, Branches & tags
│   ├── code/             # Files, Search, Metrics, Duplicates
│   └── structure/        # Architecture, Dependencies
├── components/
│   ├── layout.tsx        # PageHeader, Section
│   ├── states.tsx        # Loading, ErrorNotice, Empty, QueryView
│   ├── RequireRepo.tsx   # gate for pages that need a scanned repository
│   └── ui/               # shadcn primitives (Base UI)
└── lib/
    ├── api.ts            # fetch client and TanStack Query hooks
    ├── ws.ts             # live updates from the job WebSocket
    ├── types.ts          # API contracts
    └── *.ts / *.test.ts  # pure helpers with unit tests
```

## Design

See "Interface design" in the repository's `AGENTS.md`. In short: tokens only,
Barlow for text, JetBrains Mono for code, sections instead of card grids,
sentence-case copy, and every page checked at 390px in both themes.
