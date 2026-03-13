# sereni-base - Foundation for Cloud-Native Microservices

> Enterprise-grade open source database platform and no-code application builder. A comprehensive self-hosted backend solution and database management tool providing microservice orchestration, database integration, authentication, and observability for production-grade applications. Free open source database builder and no-code platform serving as an Airtable alternative.

[![Version](https://img.shields.io/badge/Version-1.0.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat-square&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Quality Gate Status](https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_1234abcd&metric=alert_status&token=sqb_152d71a0f9a3621514372a3e4c87460e3059bbc2)](https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_1234abcd)

## Overview

**sereni-base** is an enterprise-grade open source database platform and self-hosted backend platform that serves as a comprehensive alternative to Airtable and NocoDB. This no-code open source database & application builder and database management tool provides essential building blocks for secure, scalable microservices including service orchestration, database integration, authentication, and comprehensive observability capabilities.

## Key Features

- **Service Orchestration**: Advanced microservice coordination and configuration management
- **Database Integration**: Comprehensive PostgreSQL integration with connection pooling
- **Authentication & Authorization**: Enterprise-grade security with role-based access control
- **Observability & Metrics**: Comprehensive monitoring, logging, and performance analytics
- **Cloud-Native Architecture**: Container-ready with Kubernetes support

## Architecture
- Go 1.23+, idiomatic design
- Modular, testable codebase

## Installation
```sh
go get github.com/aptlogica/sereni-base
```

## Configuration
See `.env.example` for environment variables and configuration options.

## Quick Start

```go
package main

import (
    "context"
    "log"
    
    "github.com/aptlogica/sereni-base/pkg/server"
    "github.com/aptlogica/sereni-base/pkg/config"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal("Failed to load config:", err)
    }
    
    // Initialize server
    srv := server.New(cfg)
    
    // Start server
    ctx := context.Background()
    if err := srv.Start(ctx); err != nil {
        log.Fatal("Server failed to start:", err)
    }
}
```

### Docker Deployment

```bash
# Clone repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Configure environment
cp .env.example .env
# Edit .env with your settings

# Start with Docker Compose
docker-compose up -d
```

## Development

### Local Setup
```bash
# Clone the repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Configure your database and other settings in .env

# Run migrations
go run ./cmd/migrate up

# Start development server
go run ./cmd/server
```

### Environment Configuration
```bash
DATABASE_URL=postgres://user:password@localhost:5432/serenibase
PORT=8080
JWT_SECRET=your-secret-key
LOG_LEVEL=debug
REDIS_URL=redis://localhost:6379
```

## Testing
- Run `go test ./...` to execute unit tests

## Security
See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## License
MIT License. Copyright (c) 2026 Aptlogica Technologies.

