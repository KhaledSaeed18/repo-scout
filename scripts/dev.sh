#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  kill "${BACKEND_PID:-}" "${FRONTEND_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

mkdir -p "$ROOT/bin"

echo "[dev] building backend"
go build -o "$ROOT/bin/repo-scout" ./cmd/repo-scout
BACKEND_PID=""
REPO_SCOUT_DB="${REPO_SCOUT_DB:-$ROOT/data/reposcout.db}" "$ROOT/bin/repo-scout" &
BACKEND_PID=$!

echo "[dev] starting frontend"
(cd "$ROOT/frontend" && pnpm run dev) &
FRONTEND_PID=$!

wait
