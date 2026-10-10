SHELL := /bin/bash

# Development keeps its database in the repository; installed builds use the
# per-user default (see internal/config).
DEV_DB ?= data/reposcout.db
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: dev backend frontend test lint build ui vet fmt help

## dev: run backend + frontend together
dev:
	REPO_SCOUT_DB=$(DEV_DB) ./scripts/dev.sh

## backend: build and run the Go API
backend:
	@mkdir -p bin
	go build -o bin/repo-scout ./cmd/repo-scout
	REPO_SCOUT_DB=$(DEV_DB) ./bin/repo-scout

## frontend: run the Vite dev server
frontend:
	cd frontend && pnpm run dev

## test: Go tests, frontend typecheck + tests
test:
	go test ./...
	cd frontend && pnpm run typecheck && pnpm run test

## lint: go vet, golangci-lint (if present), frontend oxlint
lint:
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then golangci-lint run ./...; else echo "golangci-lint not found; skipping"; fi
	cd frontend && pnpm run lint

## vet: run go vet only
vet:
	go vet ./...

## fmt: format Go code
fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*' -not -path './frontend/*')

## ui: build the frontend and stage it for embedding
ui:
	cd frontend && pnpm run build
	rm -rf internal/webui/dist
	cp -R frontend/dist internal/webui/dist

## build: a single bin/repo-scout with the interface embedded
build: ui
	@mkdir -p bin
	go build -tags embedui -trimpath -ldflags "$(LDFLAGS)" -o bin/repo-scout ./cmd/repo-scout

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | awk -F':' '{printf "  %-12s %s\n", $$1, $$2}'
