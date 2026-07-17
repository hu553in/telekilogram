# syntax=docker/dockerfile:1

FROM golang:1.26.5-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./

RUN --mount=type=cache,target=/go/pkg/mod \
  go mod download

COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN --mount=type=cache,target=/root/.cache/go-build \
  CGO_ENABLED=1 GOFLAGS="-buildvcs=false" \
  go build -trimpath -ldflags="-s -w" -o /dist/telekilogram ./cmd

FROM debian:bookworm-slim AS runner

RUN --mount=type=cache,target=/var/cache/apt \
  --mount=type=cache,target=/var/lib/apt/lists \
  apt-get update && \
  apt-get install -y --no-install-recommends \
  ca-certificates

RUN groupadd --gid 10001 app \
  && useradd --uid 10001 --gid 10001 -M app \
  && install -d -m 0750 -o app -g app /data

COPY --from=builder /dist/telekilogram /usr/local/bin/telekilogram

WORKDIR /data
USER app
ENTRYPOINT ["/usr/local/bin/telekilogram"]
