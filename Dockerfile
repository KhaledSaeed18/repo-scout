# syntax=docker/dockerfile:1

# Repo Scout as a container: the server with the interface embedded, plus git.
#
#   docker run --rm -p 127.0.0.1:8080:8080 \
#     -v repo-scout-data:/data -v "$HOME/code:/repos:ro" \
#     ghcr.io/khaledsaeed18/repo-scout
#
# Then scan folders under /repos from http://localhost:8080.

# The interface is platform independent; build it once on the build machine.
FROM --platform=$BUILDPLATFORM node:22-alpine AS ui
WORKDIR /src/frontend
RUN corepack enable
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm run build

# Cross-compile the server for the target platform without emulation.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS server
ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=ui /src/frontend/dist ./internal/webui/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -tags embedui -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/repo-scout ./cmd/repo-scout

FROM alpine:3.22
# git reads history; mounted repositories belong to another user, so mark
# them safe or git refuses to read them.
RUN apk add --no-cache git ca-certificates \
    && git config --system --add safe.directory '*' \
    && adduser -D -u 10001 scout \
    && mkdir /data && chown scout /data
COPY --from=server /out/repo-scout /usr/local/bin/repo-scout
USER scout
# Inside the container the server must listen on all interfaces to be
# reachable; publish the port on 127.0.0.1 only (see above).
ENV REPO_SCOUT_ADDR=0.0.0.0:8080 \
    REPO_SCOUT_DB=/data/reposcout.db
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/api/health >/dev/null || exit 1
ENTRYPOINT ["repo-scout"]
CMD ["serve"]
