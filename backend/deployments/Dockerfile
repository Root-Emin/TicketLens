# TicketLens backend image. Build context is backend/ (see .dockerignore).
#
# Ships two binaries from one build so the migration stage of a deploy and the
# server it precedes can never be built from different sources:
#   /app/ticketlens   the HTTP server
#   /app/migrate      one-shot migration runner (cmd/migrate)

# Build stage
FROM golang:1.26.4-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Stamped into the binary so a running container can report which release it is.
# Defaults keep a plain `docker build` working; the release workflow passes the
# git tag and sha.
ARG VERSION=0.1.0-dev
ARG COMMIT=""

ENV CGO_ENABLED=0 GOOS=linux
RUN go build \
        -trimpath \
        -ldflags "-s -w -X github.com/Root-Emin/TicketLens/internal/shared/version.Version=${VERSION} -X github.com/Root-Emin/TicketLens/internal/shared/version.Commit=${COMMIT}" \
        -o /bin/ticketlens ./cmd/server \
 && go build \
        -trimpath \
        -ldflags "-s -w -X github.com/Root-Emin/TicketLens/internal/shared/version.Version=${VERSION} -X github.com/Root-Emin/TicketLens/internal/shared/version.Commit=${COMMIT}" \
        -o /bin/migrate ./cmd/migrate

# Runtime stage
FROM alpine:3.24

# wget serves the container healthcheck below; busybox already provides it, but
# naming it here keeps the dependency explicit if the base image ever changes.
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# The SQL migrations are compiled into both binaries with //go:embed
# (internal/infrastructure/postgres/migrate.go), so the image carries no
# migrations directory. Copying one would create a second, silently stale copy.
COPY --from=builder /bin/ticketlens /app/ticketlens
COPY --from=builder /bin/migrate /app/migrate

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

# Readiness, not liveness: this is what the compose dependency waits on, and a
# server whose database is gone must not be reported as usable.
HEALTHCHECK --interval=15s --timeout=5s --start-period=20s --retries=4 \
    CMD wget -qO- http://127.0.0.1:8080/health/ready >/dev/null 2>&1 || exit 1

ENTRYPOINT ["/app/ticketlens"]
