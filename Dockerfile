# ==============================================================================
# Build Stage
# ==============================================================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Copy dependency files for better layer caching
COPY go.mod go.sum ./

# Copy the local module required by 'replace' directive in go.mod
# Ensure go-postgres-rest exists inside your build context
COPY go-postgres-rest ./go-postgres-rest


# Install Swagger CLI tool
RUN go install github.com/swaggo/swag/cmd/swag@latest

# Copy the rest of application source code
COPY . .

# Generate Swagger documentation for antivirus-service
WORKDIR /app/services/antivirus-service
RUN swag init -g internal/handlers/scan_handler.go -o docs
WORKDIR /app

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main ./cmd/server/main.go

# ==============================================================================
# Production Stage
# ==============================================================================
FROM alpine:3.20

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

WORKDIR /app

# Copy binary and required files from builder
COPY --from=builder /app/main .
COPY --from=builder /app/docs ./docs

# Copy .env file if it exists (optional for production)

# Create assets directory for uploads or static files
RUN mkdir -p /app/assets

# Create non-root user for security
RUN adduser -D -s /bin/sh serenibase && \
    chown -R serenibase:serenibase /app

# Switch to non-root user
USER serenibase

# Expose application port
EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/api/v1/health || exit 1

# Run the application
CMD ["./main"]