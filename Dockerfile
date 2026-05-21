# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.26.2-alpine@sha256:c2a1f7b2095d046ae14b286b18413a05bb82c9bca9b25fe7ff5efef0f0826166 AS builder
RUN go version

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

ARG VERSION=dev

# Force module mode so builds are not affected by any stale vendor directory.
ENV GOFLAGS=-mod=mod

WORKDIR /app

# Copy dependency files for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of application source code (including docs)
COPY . .

# Ensure module graph is in sync with imports on fresh machines/clean Docker caches.
RUN go mod tidy && go mod download

# Build the application with optimizations
# Note: Swagger docs in /docs are embedded in the binary at compile time
# Single-line RUN avoids line-continuation parse issues on some Docker setups.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s -extldflags \"-static\" -X main.version=${VERSION}" -a -installsuffix cgo -o main ./cmd/server/main.go

# ==============================================================================
# Production Stage
# ==============================================================================
FROM alpine:3.20.6@sha256:de4fe7064d8f98419ea6b49190df1abbf43450c1702eeb864fe9ced453c1cc5f

# Install runtime dependencies including PostgreSQL client
RUN apk --no-cache add ca-certificates tzdata curl postgresql-client

WORKDIR /app

# Copy binary and required files from builder
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs
COPY wait-for-postgres.sh .

# Create assets directory and non-root user for security
RUN chmod +x wait-for-postgres.sh && mkdir -p /app/assets && adduser -D -s /bin/sh serenibase && chown -R serenibase:serenibase /app

# Switch to non-root user
USER serenibase

# Expose application port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Run the application
CMD ["./main"]
