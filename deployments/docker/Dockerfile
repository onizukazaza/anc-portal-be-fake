# =============================================================================
# Multi-stage Dockerfile — anc-portal-be
# =============================================================================
# Build all:   docker build -f deployments/docker/Dockerfile -t anc-portal-be .
# Build worker: docker build -f deployments/docker/Dockerfile --target worker -t anc-portal-worker .
# Run:         docker run -p 20000:20000 --env-file .env.local anc-portal-be
# =============================================================================

# ---------------------------------------------------------------------------
# Stage 1: Build — compile all Go binaries
# ---------------------------------------------------------------------------
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build all binaries in a single layer to reduce image size
ARG GIT_COMMIT=dev
ARG BUILD_TIME=unknown
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w \
      -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.GitCommit=${GIT_COMMIT} \
      -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /app/api ./cmd/api && \
    go build -ldflags="-s -w \
      -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.GitCommit=${GIT_COMMIT} \
      -X github.com/onizukazaza/anc-portal-be-fake/pkg/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /app/worker ./cmd/worker && \
    go build -ldflags="-s -w" -o /app/migrate ./cmd/migrate && \
    go build -ldflags="-s -w" -o /app/seed ./cmd/seed && \
    go build -ldflags="-s -w" -o /app/sync ./cmd/sync

# ---------------------------------------------------------------------------
# Stage 2: Runtime base — shared runtime layer
# ---------------------------------------------------------------------------
FROM alpine:3.21 AS runtime-base

RUN apk add --no-cache ca-certificates tzdata curl && \
    addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# ---------------------------------------------------------------------------
# Stage 3: API — full image with all binaries (default target)
# ---------------------------------------------------------------------------
FROM runtime-base AS api

COPY --from=builder /app/api     ./api
COPY --from=builder /app/worker  ./worker
COPY --from=builder /app/migrate ./migrate
COPY --from=builder /app/seed    ./seed
COPY --from=builder /app/sync    ./sync

# Copy migration files + config
COPY migrations/ ./migrations/
COPY config/     ./config/

USER appuser
EXPOSE 20000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:20000/healthz || exit 1

ENTRYPOINT ["./api"]

# ---------------------------------------------------------------------------
# Stage 4: Worker — lightweight image with worker binary only
# ---------------------------------------------------------------------------
FROM runtime-base AS worker

COPY --from=builder /app/worker ./worker
COPY config/ ./config/

USER appuser

ENTRYPOINT ["./worker"]
