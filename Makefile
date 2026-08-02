SHELL := /bin/bash

.PHONY: dev backend frontend test lint build vet fmt help

## dev: run backend + frontend together
dev:
	./scripts/dev.sh

## backend: build and run the Go API
backend:
	@mkdir -p bin
	go build -o bin/api ./cmd/api
	./bin/api

## frontend: run the Vite dev server
frontend:
	pnpm --prefix frontend run dev

## test: Go tests, frontend typecheck + tests
test:
	go test ./...
	cd frontend && pnpm run typecheck && pnpm run test

## lint: go vet, golangci-lint (if present), frontend eslint
lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not found; skipping"; fi
	cd frontend && pnpm run lint

## vet: run go vet only
vet:
	go vet ./...

## fmt: format Go code
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

## build: production builds for both halves
build:
	go build -o bin/api ./cmd/api
	pnpm --prefix frontend run build

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F':' '{printf "  %-12s %s\n", $$1, $$2}'
