# SereniBase Complete Setup Guide

**Comprehensive Setup Documentation for Windows, macOS, and Linux**

This guide provides step-by-step instructions for first-time users setting up SereniBase from scratch.

---

## Table of Contents

1. [Deployment Modes](#deployment-modes)
2. [Minimum Requirements](#1-minimum-requirements)
3. [Clone Repository](#2-clone-repository)
4. [Backend Only Setup](#3a-backend-only-setup)
5. [Full Application Setup](#3b-full-application-setup)
6. [Access URLs](#4-access-urls)
7. [Verify Installation](#5-verify-containers)
8. [Day-2 Operations](#6-day-2-commands)
9. [Troubleshooting](#7-troubleshooting)
10. [Clean Reinstall](#8-clean-from-scratch-reinstall)

---

## Deployment Modes

SereniBase supports **two deployment configurations**:

| Mode | Description | Use Case |
|------|-------------|----------|
| **Backend Only** | Core REST API + PostgreSQL database | API development, microservice integration, lightweight testing |
| **Full Application** | Complete stack with UI, authentication, email, storage, and antivirus | Production deployments, full-stack development, demos |

### Services Comparison

| Service | Backend Only | Full Application |
|---------|:------------:|:----------------:|
| SereniBase REST API | ✓ | ✓ |
| PostgreSQL Database | ✓ | ✓ |
| JWT Authentication | ✗ | ✓ |
| Email Service | ✗ | ✓ |
| Storage Service | ✗ | ✓ |
| MinIO Object Storage | ✗ | ✓ |
| Antivirus (ClamAV) | ✗ | ✓ |
| Frontend UI | ✗ | ✓ |

---

## 1. Minimum Requirements

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Docker Desktop | Latest | Latest |
| RAM (available to Docker) | 4 GB | 8 GB+ |
| Free Disk Space | 10 GB | 20 GB+ |
| Git | Any | Latest |
| Make | Optional | Recommended |

### Verify Tools

```bash
docker --version
docker compose version
git --version
```

> **Note**: If `docker compose` fails, install/enable the Docker Compose plugin first.

---

## 2. Clone Repository

```bash
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base
```

---

## 3a. Backend Only Setup

For API development and lightweight deployments without the full UI stack.

### Quick Start

```bash
# Copy environment template
cp build/config/.env.example .env

# Start backend services
docker compose -f docker-compose.yaml up -d

# Verify services
docker compose -f docker-compose.yaml ps
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

# Stop services (preserve data)
docker compose -f docker-compose.yaml down

# Stop and remove all data
docker compose -f docker-compose.yaml down -v
```

---

## 3b. Full Application Setup

For complete deployments with all microservices and the frontend UI.

### Interactive Setup (Recommended)

#### Windows (PowerShell)

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup.ps1
```

#### macOS / Linux

```bash
chmod +x build/scripts/setup.sh build/scripts/setup-y.sh
./build/scripts/setup.sh
```

### Non-Interactive Setup (Default Values)

#### Windows

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup-y.ps1
```

#### macOS/Linux

```bash
./build/scripts/setup-y.sh
```

### Using Make (All Platforms)

```bash
# Interactive setup
make setup

# Non-interactive with defaults
make setup-y
```

---

## 4. Access URLs

### Full Application Mode

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | `http://localhost:5050` | Web application interface |
| Backend API | `http://localhost:8080` | REST API endpoint |
| Health Check | `http://localhost:8080/api/v1/health` | API health status |
| MinIO Console | `http://localhost:9001` | Object storage administration |

### Backend Only Mode

| Service | URL | Description |
|---------|-----|-------------|
| Backend API | `http://localhost:8080` | REST API endpoint |
| Health Check | `http://localhost:8080/api/v1/health` | API health status |

---

## 5. Verify Containers

### Full Application

```bash
docker compose -f docker-compose.all.yaml ps
```

### Backend Only

```bash
docker compose -f docker-compose.yaml ps
```

**Expected**: All services should show `Up` status (some may display `healthy` after initialization).

### Troubleshooting Unhealthy Services

```bash
docker compose -f docker-compose.all.yaml logs --tail=200 <service-name>
```

## 6. Day-2 Commands

### Full Application Mode

| Action | Command |
|--------|---------|
| Start services | `docker compose -f docker-compose.all.yaml up -d` |
| Stop services | `docker compose -f docker-compose.all.yaml down` |
| View logs | `docker compose -f docker-compose.all.yaml logs -f` |
| Rebuild after changes | `docker compose -f docker-compose.all.yaml up --build -d` |
| Hard reset (delete data) | `docker compose -f docker-compose.all.yaml down -v` |

### Backend Only Mode

| Action | Command |
|--------|---------|
| Start services | `docker compose -f docker-compose.yaml up -d` |
| Stop services | `docker compose -f docker-compose.yaml down` |
| View logs | `docker compose -f docker-compose.yaml logs -f` |
| Rebuild after changes | `docker compose -f docker-compose.yaml up --build -d` |
| Hard reset (delete data) | `docker compose -f docker-compose.yaml down -v` |

### Using Make (Full Application Only)

| Action | Command |
|--------|---------|
| Start | `make up` |
| Stop | `make down` |
| Stop & remove data | `make down-all` |
| View logs | `make logs` |
| Service status | `make ps` |
| Detailed status | `make status` |

## 7. Troubleshooting

### A) `dockerfile parse error ... unknown instruction: server`

Cause:
- Old Docker parser does not support heredoc syntax used in some Dockerfiles.

Status in this repo:
- Fixed by using `services/base-ui/nginx.default.conf` + `COPY` in `services/base-ui/Dockerfile`.

What to do:
1. Pull latest repo changes.
2. Rebuild:
   ```bash
   docker compose -f docker-compose.all.yaml build --no-cache base-ui
   docker compose -f docker-compose.all.yaml up -d
   ```

### B) Login fails with JWT/auth errors

Common causes:
- `AUTH_JWT_SECRET` changed between runs.
- Auth container not healthy.
- App is using stale token from old setup.

Fix:
1. Check auth container:
   ```bash
   docker compose -f docker-compose.all.yaml ps jwt-provider
   docker compose -f docker-compose.all.yaml logs --tail=200 jwt-provider
   ```
2. Check `.env` has one stable value for `AUTH_JWT_SECRET`.
3. If you changed secret, stop containers and restart:
   ```bash
   docker compose -f docker-compose.all.yaml down
   docker compose -f docker-compose.all.yaml up -d
   ```
4. Clear browser local storage/session for base-ui and login again.

### C) Some containers start, some don’t

Common causes:
- Port conflicts (5050, 8080, 8081, 8082, 8083, 8084, 5432, 9000, 9001, 3310).
- Low Docker memory/CPU.
- Stale volumes or broken previous state.

Fix order:
1. Check container states:
   ```bash
   docker compose -f docker-compose.all.yaml ps
   ```
2. Check logs for failing service:
   ```bash
   docker compose -f docker-compose.all.yaml logs --tail=200 <service-name>
   ```
3. Resolve port conflicts, then restart.
4. If still broken:
   ```bash
   docker compose -f docker-compose.all.yaml down -v
   docker compose -f docker-compose.all.yaml up --build -d
   ```

### D) Windows-specific `.env` weird behavior

Cause:
- UTF-8 BOM in `.env` can break first variable parsing on some tools.

Status in this repo:
- Setup scripts now write `.env` as UTF-8 **without BOM**.

If you already have a bad `.env`, regenerate:

```powershell
Remove-Item .env -Force
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup.ps1
```

### E) macOS/Linux script not executable

```bash
chmod +x build/scripts/*.sh
```

## 8. Clean From-Scratch Reinstall

Use this when migrating between machines or resolving unknown state issues.

### Full Application

```bash
docker compose -f docker-compose.all.yaml down -v
docker system prune -f
git pull
make setup
```

### Backend Only

```bash
docker compose -f docker-compose.yaml down -v
docker system prune -f
git pull
docker compose -f docker-compose.yaml up -d
```

---

## 9. Configuration Files Reference

| File | Purpose |
|------|---------|
| `build/scripts/setup.sh` | Interactive setup (Linux/macOS) |
| `build/scripts/setup.ps1` | Interactive setup (Windows) |
| `build/scripts/setup-y.sh` | Auto setup (Linux/macOS) |
| `build/scripts/setup-y.ps1` | Auto setup (Windows) |
| `docker-compose.yaml` | Backend only deployment |
| `docker-compose.all.yaml` | Full application deployment |
| `build/config/.env.example` | Environment template |
| `.env` | Active configuration (generated) |

---

## 10. Related Documentation

- [Setup Guide](SETUP.md) - Quick reference setup guide
- [Interactive Setup](../INTERACTIVE_SETUP_README.md) - Detailed wizard documentation
- [Environment Variables](../docs/ENVIRONMENT_VARIABLES.md) - Complete variable reference
- [Quick Reference Card](../docs/ENV_QUICK_REFERENCE_CARD.md) - Common configurations

