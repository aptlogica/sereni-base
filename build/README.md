# Build Scripts

This directory contains setup scripts, configuration templates, and deployment guides for SereniBase.

---

## Deployment Modes

SereniBase supports two deployment configurations:

| Mode | Compose File | Description | Best For |
|------|--------------|-------------|----------|
| **Backend Only** | `docker-compose.yaml` | REST API + PostgreSQL | API development, testing, microservice integration |
| **Full Application** | `docker-compose.all.yaml` | Complete stack with all services | Production, full-stack development, demos |

---

## Documentation

| Guide | Description |
|-------|-------------|
| [SETUP_COMPLETE_GUIDE.md](SETUP_COMPLETE_GUIDE.md) | Comprehensive beginner guide (recommended start) |
| [SETUP.md](SETUP.md) | Quick reference setup guide |

---

## Setup Scripts

### Full Application Setup (Interactive)

| Platform | Script | Description |
|----------|--------|-------------|
| Linux/macOS | `build/scripts/setup.sh` | Interactive wizard with prompts |
| Windows | `build/scripts/setup.ps1` | Interactive wizard with prompts |

### Full Application Setup (Non-Interactive)

| Platform | Script | Description |
|----------|--------|-------------|
| Linux/macOS | `build/scripts/setup-y.sh` | Auto setup with default values |
| Windows | `build/scripts/setup-y.ps1` | Auto setup with default values |

### Service Cloning

| Platform | Script | Description |
|----------|--------|-------------|
| Linux/macOS | `build/scripts/clone-services.sh` | Clone microservice repositories |
| Windows | `build/scripts/clone-services.ps1` | Clone microservice repositories |

---

## Usage

### Full Application Setup

#### Linux/macOS

```bash
# Interactive setup
./build/scripts/setup.sh

# Non-interactive (default values)
./build/scripts/setup-y.sh

# Using Make
make setup        # Interactive
make setup-y      # Non-interactive
```

#### Windows PowerShell

```powershell
# Interactive setup
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup.ps1

# Non-interactive (default values)
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup-y.ps1

# Using Make
make setup        # Interactive
make setup-y      # Non-interactive
```

### Backend Only Setup

```bash
# Copy environment template
cp build/config/.env.example .env

# Start backend services
docker compose -f docker-compose.yaml up -d

# Verify services
docker compose -f docker-compose.yaml ps
```

---

## Directory Structure

```
build/
├── README.md                 # This file
├── SETUP.md                  # Quick reference setup guide
├── SETUP_COMPLETE_GUIDE.md   # Comprehensive beginner guide
├── config/
│   └── .env.example          # Environment template
└── scripts/
    ├── setup.sh              # Interactive setup (Linux/macOS)
    ├── setup.ps1             # Interactive setup (Windows)
    ├── setup-y.sh            # Auto setup (Linux/macOS)
    ├── setup-y.ps1           # Auto setup (Windows)
    ├── clone-services.sh     # Clone services (Linux/macOS)
    └── clone-services.ps1    # Clone services (Windows)
```
