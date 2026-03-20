# Cross-platform Makefile for SereniBase
# Works on Linux, macOS, and Windows
#
# All build scripts are located in: build/scripts/
# Configuration templates are in:   build/config/

.PHONY: setup setup-y help up down down-all restart logs clean clean-all ps status test test-coverage

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SETUP_SCRIPT := build\scripts\setup.ps1
    SETUP_SCRIPT_Y := build\scripts\setup-y.ps1
    COMPOSE_FILE := docker-compose.all.yaml
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        DETECTED_OS := Linux
    else ifeq ($(UNAME_S),Darwin)
        DETECTED_OS := macOS
    else
        DETECTED_OS := Unix
    endif
    SETUP_SCRIPT := build/scripts/setup.sh
    SETUP_SCRIPT_Y := build/scripts/setup-y.sh
    COMPOSE_FILE := docker-compose.all.yaml
endif

# Default target
help:
	@echo ""
	@echo "========================================================================"
	@echo "                     SERENIBASE - COMMANDS"
	@echo "========================================================================"
	@echo ""
	@echo "  Quick Start:"
	@echo "    make setup              - Interactive setup wizard (recommended)"
	@echo ""
	@echo "  Testing:"
	@echo "    make test               - Run all tests"
	@echo "    make test-coverage      - Run tests with coverage report"
	@echo ""
	@echo "  Docker Management:"
	@echo "    make up                 - Start all services"
	@echo "    make down               - Stop all services (keep data)"
	@echo "    make down-all           - Stop all services and remove volumes"
	@echo "    make restart            - Restart all services"
	@echo "    make logs               - View logs from all services"
	@echo "    make ps                 - Show running services"
	@echo "    make status             - Show detailed service status"
	@echo ""
	@echo "  Cleanup:"
	@echo "    make clean              - Stop services and remove containers"
	@echo "    make clean-all          - Full cleanup (containers + volumes + images)"
	@echo ""
	@echo "  Detected OS: $(DETECTED_OS)"
	@echo ""
	@echo "  Documentation:"
	@echo "    See docs/ENV_CONFIGURATION.md for environment setup"
	@echo ""

# Interactive setup wizard
setup:
	@echo "Starting SereniBase Setup Wizard..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -File $(SETUP_SCRIPT)
else
	@chmod +x $(SETUP_SCRIPT)
	@bash $(SETUP_SCRIPT)
endif

# Full automated setup with defaults (no prompts)
setup-y:
	@echo "Starting SereniBase Setup with defaults (no prompts)..."
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -File $(SETUP_SCRIPT_Y)
else
	@chmod +x $(SETUP_SCRIPT_Y)
	@bash $(SETUP_SCRIPT_Y)
endif

# ============================================================================
# Testing Commands
# ============================================================================

# Run all tests
test:
	go test ./...

# Run tests with coverage report
test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out -covermode=atomic -coverpkg=./... ./tests/...
	@echo "Coverage report generated at coverage.out"

# ============================================================================
# Docker Management Commands
# ============================================================================

# Start all services
up:
	@echo "Starting all SereniBase services..."
	@docker compose -f $(COMPOSE_FILE) up -d
	@echo ""
	@echo "Services started! Access:"
	@echo "  Frontend:  http://localhost:5050"
	@echo "  Backend:   http://localhost:8080"
	@echo "  MinIO:     http://localhost:9001"
	@echo "  MailHog:   http://localhost:8025"
	@echo ""

# Stop services (keep data)
down:
	@echo "Stopping all services (preserving data)..."
	@docker compose -f $(COMPOSE_FILE) down
	@echo "Services stopped. Data volumes preserved."

# Stop services and remove volumes (clean slate)
down-all:
	@echo "Stopping all services and removing volumes..."
	@docker compose -f $(COMPOSE_FILE) down -v
	@echo "Services stopped and all data removed."

# Restart all services
restart:
	@echo "Restarting all services..."
	@docker compose -f $(COMPOSE_FILE) restart
	@echo "All services restarted."

# View logs
logs:
	@docker compose -f $(COMPOSE_FILE) logs -f

# Show running services
ps:
	@docker compose -f $(COMPOSE_FILE) ps

# Show detailed service status
status:
	@echo "========================================================================"
	@echo "                    SERENIBASE SERVICE STATUS"
	@echo "========================================================================"
	@echo ""
	@docker compose -f $(COMPOSE_FILE) ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}"
	@echo ""
	@echo "To view logs: make logs"
	@echo ""

# ============================================================================
# Cleanup Commands
# ============================================================================

# Stop and remove containers (keep volumes)
clean:
	@echo "Cleaning up containers..."
	@docker compose -f $(COMPOSE_FILE) down
	@echo "Containers removed. Volumes preserved."

# Full cleanup: containers + volumes + images
clean-all:
	@echo "========================================================================"
	@echo "                    FULL CLEANUP WARNING"
	@echo "========================================================================"
	@echo ""
	@echo "This will remove:"
	@echo "  - All containers"
	@echo "  - All volumes (DATABASE DATA WILL BE LOST)"
	@echo "  - All built images"
	@echo ""
ifeq ($(OS),Windows_NT)
	@powershell -Command "$$response = Read-Host 'Are you sure? Type YES to continue'; if ($$response -ne 'YES') { Write-Host 'Cleanup cancelled.'; exit 1 }"
else
	@read -p "Are you sure? Type YES to continue: " response; \
	if [ "$$response" != "YES" ]; then \
		echo "Cleanup cancelled."; \
		exit 1; \
	fi
endif
	@echo ""
	@echo "Stopping all services..."
	@docker compose -f $(COMPOSE_FILE) down -v
	@echo ""
	@echo "Removing built images..."
	@docker compose -f $(COMPOSE_FILE) down --rmi local
	@echo ""
	@echo "Full cleanup complete!"
	@echo ""
	@echo "To start fresh, run: make setup"
	@echo ""
