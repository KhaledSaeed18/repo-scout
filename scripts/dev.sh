#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  kill "${BACKEND_PID:-}" "${FRONTEND_PID:-}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

mkdir -p "$ROOT/bin"

echo "[dev] building backend"
go build -o "$ROOT/bin/api" ./cmd/api
BACKEND_PID=""
"$ROOT/bin/api" &
BACKEND_PID=$!

echo "[dev] starting frontend"
pnpm --prefix "$ROOT/frontend" run dev &
FRONTEND_PID=$!

wait
