# Interactive Setup Guide

This guide explains the interactive setup scripts used to configure SereniBase.

---

## Deployment Modes

SereniBase supports two deployment configurations:

| Mode | Compose File | Description |
|------|--------------|-------------|
| **Backend Only** | `docker-compose.yaml` | Core REST API + PostgreSQL |
| **Full Application** | `docker-compose.all.yaml` | Complete stack with all microservices |

> **Note**: The interactive setup wizard configures the **Full Application** mode. For Backend Only mode, see [Backend Only Setup](#backend-only-setup).

---

## Quick Start

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| Docker + Docker Compose | Required to run the stack |
| Git | Required to clone the repo |
| Chocolatey (Windows) | Install: https://chocolatey.org/install |
| Make (GNU Make) | Windows: `choco install make` |
| SMTP access | Optional, required for email notifications |

---

## Full Application Setup (Interactive Wizard)

### Windows (PowerShell)

```powershell
.\setup-interactive.ps1
```

### Linux/macOS (Bash)

```bash
chmod +x setup-interactive.sh
./setup-interactive.sh
```

---

## Backend Only Setup

For lightweight API-only deployments without the full UI and supporting services:

### Quick Start

```bash
# Copy environment template
cp build/config/.env.example .env

# Edit .env with your database credentials (optional)
# Default values work for local development

# Start backend services
docker compose -f docker-compose.yaml up -d
```

### Backend Services

| Service | Port | Description |
|---------|------|-------------|
| serenibase-rest | 8080 | REST API server |
| postgres | 5432 | PostgreSQL database |

### Backend Commands

```bash
# Start services
docker compose -f docker-compose.yaml up -d

# View logs
docker compose -f docker-compose.yaml logs -f

# Stop services
docker compose -f docker-compose.yaml down

# Stop and remove all data
docker compose -f docker-compose.yaml down -v
```

---

## What the Setup Does

- Collects required configuration values
- Generates or updates `.env`
- Prepares Docker volumes
- Starts the full stack via Docker Compose

## Options

### Windows (PowerShell) - Full Application

| Option | Description |
|--------|-------------|
| `-AutoYes` | Run non-interactively using defaults |
| `-SkipDocker` | Skip Docker and Docker Compose checks |
| `-Help` | Show help |

Examples:

```powershell
.\setup-interactive.ps1 -AutoYes
.\setup-interactive.ps1 -SkipDocker
.\setup-interactive.ps1 -Help
```

### Linux/macOS (Bash) - Full Application

| Option | Description |
|--------|-------------|
| `--auto-yes` | Run non-interactively using defaults |
| `--skip-docker` | Skip Docker and Docker Compose checks |
| `--help` | Show help |

Examples:

```bash
./setup-interactive.sh --auto-yes
./setup-interactive.sh --skip-docker
./setup-interactive.sh --help
```

---

## Configuration Template

The setup uses `build/config/.env.example` as the base template.

---

## Access URLs

### Full Application Mode

| Service | URL |
|---------|-----|
| Frontend | `http://localhost:5050` |
| Backend API | `http://localhost:8080` |
| Health Check | `http://localhost:8080/api/v1/health` |
| MinIO Console | `http://localhost:9001` |

### Backend Only Mode

| Service | URL |
|---------|-----|
| Backend API | `http://localhost:8080` |
| Health Check | `http://localhost:8080/api/v1/health` |

---

## Troubleshooting

- If Docker is not running, start Docker Desktop (Windows/macOS) or the Docker daemon (Linux)
- If ports are in use, stop the conflicting services or change ports in `.env`
- If the setup fails, review logs with `make logs` or `docker compose logs`

---

## Related Documentation

- [Complete Setup Guide](build/SETUP_COMPLETE_GUIDE.md) - Comprehensive beginner guide
- [Setup Reference](build/SETUP.md) - Quick reference setup guide
- [Environment Setup Guide](docs/ENVIRONMENT_SETUP_GUIDE.md) - Environment configuration
- [Environment Variables Reference](docs/ENVIRONMENT_VARIABLES.md) - Complete variable list
- [Quick Reference Card](docs/ENV_QUICK_REFERENCE_CARD.md) - Common configurations
