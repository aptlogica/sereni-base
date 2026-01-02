# --------------------------------------------
# 🏗️ Build stage
# --------------------------------------------
    FROM golang:1.24-alpine AS builder

    # Install build dependencies
    RUN apk add --no-cache git ca-certificates
    
    WORKDIR /app
    
    # Copy go.mod and go.sum first for dependency caching
    COPY go.mod go.sum ./
    
    # Copy the local module required by 'replace' directive in go.mod
    # Ensure go-db-rest exists inside your build context
    COPY go-db-rest ./go-db-rest
    
    # Download dependencies
    RUN go mod download
    
    # Install swag tool for generating Swagger docs
    RUN go install github.com/swaggo/swag/cmd/swag@latest
    
    # Copy the rest of your application source code
    COPY . .
    
    # Generate Swagger documentation (adjust the path if needed)
    RUN swag init -g internal/app/app.go -o docs
        
    # Build the Go binary
    RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
    
    # --------------------------------------------
    # 🪶 Production stage
    # --------------------------------------------
    FROM alpine:latest
    
    # Install runtime dependencies
    RUN apk --no-cache add ca-certificates tzdata curl
    
    WORKDIR /root/
    
    # Copy binary and required files from builder
    COPY --from=builder /app/main .
    COPY --from=builder /app/docs ./docs
    
    # Create assets directory for uploads or static files
    RUN mkdir -p /root/assets
    
    # Create a non-root user for security
    RUN adduser -D -s /bin/sh serenibase
    
    # Change ownership
    RUN chown -R serenibase:serenibase /root
    
    # Switch to non-root user
    USER serenibase
    
    # Expose application port
    EXPOSE 8080
    
    # Health check endpoint
    HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
        CMD curl -f http://localhost:8080/api/v1/health || exit 1
    
    # Run the application
    CMD ["./main"]
    