# ── Stage 1: Build the Go backend ────────────────────────────────────────────
FROM golang:alpine AS builder

WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source
COPY . .

# Build stripped binary
RUN go build -ldflags="-s -w" -o server ./cmd/main.go


# ── Stage 2: Production image ────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache wget ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Copy essential database files
COPY db_schema.sql .
COPY simulation_seed.sql .

# Create directory for persisted databases
RUN mkdir -p /data/cansat

# Persistence Volume
VOLUME ["/data"]

EXPOSE 8000

CMD ["./server"]
