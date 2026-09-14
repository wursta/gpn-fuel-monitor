# syntax=docker/dockerfile:1

# ---- Stage 1: Build ----
FROM golang:1.27-bookworm AS builder

WORKDIR /src

# Copy go module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /usr/local/bin/gpn-fuel-monitor .

# ---- Stage 2: Runtime ----
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S appgroup && \
    adduser -S appuser -G appgroup -h /app -s /sbin/nologin

WORKDIR /app

COPY --from=builder /usr/local/bin/gpn-fuel-monitor /app/gpn-fuel-monitor

# Create data directory owned by non-root user
RUN mkdir -p /app/data && chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Default config (will be overridden by volume mount)
COPY config.yaml.example config.yaml

ENTRYPOINT ["/app/gpn-fuel-monitor"]
