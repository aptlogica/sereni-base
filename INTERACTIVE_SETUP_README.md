# Interactive Setup Guide

This guide explains the interactive setup scripts used to configure SereniBase.

## Quick Start

## Prerequisites

| Requirement | Notes |
|-------------|-------|
| Docker + Docker Compose | Required to run the stack |
| Git | Required to clone the repo |
| Chocolatey (Windows) | Install: https://chocolatey.org/install |
| Make (GNU Make) | Windows: `choco install make` |
| SMTP access | Optional, required for email notifications |

### Windows (PowerShell)

```powershell
.\setup-interactive.ps1
```

### Linux/macOS (Bash)

```bash
chmod +x setup-interactive.sh
./setup-interactive.sh
```

## What the Setup Does

- Collects required configuration values
- Generates or updates `.env`
- Prepares Docker volumes
- Starts the full stack via Docker Compose

## Options

### Windows (PowerShell)

- `-AutoYes` : Run non-interactively using defaults
- `-SkipDocker` : Skip Docker and Docker Compose checks
- `-Help` : Show help

Examples:

```powershell
.\setup-interactive.ps1 -AutoYes
.\setup-interactive.ps1 -SkipDocker
.\setup-interactive.ps1 -Help
```

### Linux/macOS (Bash)

- `--auto-yes` : Run non-interactively using defaults
- `--skip-docker` : Skip Docker and Docker Compose checks
- `--help` : Show help

Examples:

```bash
./setup-interactive.sh --auto-yes
./setup-interactive.sh --skip-docker
./setup-interactive.sh --help
```

## Configuration Template

The setup uses `build/config/.env.example` as the base template.

## Troubleshooting

- If Docker is not running, start Docker Desktop (Windows/macOS) or the Docker daemon (Linux)
- If ports are in use, stop the conflicting services or change ports in `.env`
- If the setup fails, review logs with `make logs`

## Related Docs

- [Environment Setup Guide](docs/ENVIRONMENT_SETUP_GUIDE.md)
- [Environment Variables Reference](docs/ENVIRONMENT_VARIABLES.md)
- [Quick Reference Card](docs/ENV_QUICK_REFERENCE_CARD.md)
