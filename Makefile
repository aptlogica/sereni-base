# Cross-platform Makefile for SereniBase
# Works on Linux, macOS, and Windows
#
# All build scripts are located in: build/scripts/
# Configuration templates are in:   build/config/

# Stop on first error - Ctrl+C will stop the entire Make process
.ONESHELL:
.SHELLFLAGS := -e

.PHONY: setup setup-all setup-all-y clone-all clone-go-postgres-rest up-all down-all logs clean rebuild help status check-env setup-owner

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SHELL := cmd.exe
    SETUP_SCRIPT := build\scripts\setup.bat
    SETUP_PS := powershell -ExecutionPolicy Bypass -File build\scripts\setup-all.ps1
    CLONE_SERVICES := powershell -ExecutionPolicy Bypass -File build\scripts\clone-services.ps1
    CLONE_GO_POSTGRES := powershell -ExecutionPolicy Bypass -File build\scripts\clone-go-postgres-rest.ps1
    COPY_CMD = if not exist ".env" copy "build\config\.env.example" ".env"
    RM_CMD = if exist services rmdir /s /q services & if exist go-postgres-rest rmdir /s /q go-postgres-rest
    SLEEP_CMD = timeout /t 5 /nobreak >nul
else
    DETECTED_OS := $(shell uname -s)
    SHELL := /bin/bash
    SETUP_SCRIPT := bash build/scripts/setup.sh
    SETUP_PS := bash build/scripts/setup.sh
    CLONE_SERVICES := bash build/scripts/clone-services.sh
    CLONE_GO_POSTGRES := bash build/scripts/clone-go-postgres-rest.sh
    COPY_CMD = test -f .env || cp build/config/.env.example .env
    RM_CMD = rm -rf services go-postgres-rest
    SLEEP_CMD = sleep 5
endif

# Default target
help:
	@echo.
	@echo ========================================================================
	@echo                     SERENIBASE - COMMANDS
	@echo ========================================================================
	@echo.
	@echo   Quick Start:
	@echo     make setup              - Interactive setup wizard (recommended)
	@echo     make setup-all          - Full automated setup with owner config
	@echo     make setup-all-y        - Full setup with default values (no prompts)
	@echo.
	@echo   Development:
	@echo     make clone-all          - Clone all microservices
	@echo     make up-all             - Start all services
	@echo     make down-all           - Stop all services
	@echo     make rebuild            - Rebuild and restart all services
	@echo     make logs               - View logs from all services
	@echo.
	@echo   Configuration:
	@echo     make setup-owner        - Configure owner registration details
	@echo.
	@echo   Maintenance:
	@echo     make clean              - Remove containers, volumes, and repos
	@echo     make status             - Show status of all services
	@echo.
	@echo   Documentation:
	@echo     See docs/ENV_CONFIGURATION.md for environment setup
	@echo.

# Interactive setup wizard
setup:
	@echo Starting SereniBase Setup Wizard...
	@$(SETUP_SCRIPT)

# Full automated setup
setup-all: check-env
	@echo Starting full setup process...
	@$(MAKE) clone-all || exit 1
	@$(MAKE) clone-go-postgres-rest || exit 1
	@$(MAKE) setup-owner || exit 1
	@$(MAKE) up-all || exit 1
	@echo.
	@echo ========================================================================
	@echo                     SETUP COMPLETE!
	@echo ========================================================================
	@echo.
	@echo Access your application at:
	@echo   - Frontend: http://localhost:5050
	@echo   - Backend:  http://localhost:8080
	@echo.

# Full automated setup with defaults (no prompts)
setup-all-y: check-env
	@echo Starting full setup process with defaults...
	@$(MAKE) clone-all || exit 1
	@$(MAKE) clone-go-postgres-rest || exit 1
	@$(MAKE) setup-owner-y || exit 1
	@$(MAKE) up-all || exit 1
	@echo.
	@echo ========================================================================
	@echo                     SETUP COMPLETE!
	@echo ========================================================================
	@echo.
	@echo Owner Configuration (using defaults):
	@echo   - First Name: Admin
	@echo   - Last Name: User
	@echo   - Email: admin@example.com
	@echo   - Password: Admin@123
	@echo.
	@echo Access your application at:
	@echo   - Frontend: http://localhost:5050
	@echo   - Backend:  http://localhost:8080
	@echo.

# Setup owner with defaults (no prompts)
setup-owner-y:
	@echo.
	@echo ========================================================================
	@echo              OWNER REGISTRATION CONFIGURATION (DEFAULTS)
	@echo ========================================================================
	@echo.
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -File build/scripts/setup-owner.ps1 -UseDefaults
else
	@bash build/scripts/setup-owner.sh -y
endif

# Clone all microservices
clone-all:
	@echo Cloning all microservices...
	@$(CLONE_SERVICES)

# Clone go-postgres-rest
clone-go-postgres-rest:
	@echo Cloning go-postgres-rest...
	@$(CLONE_GO_POSTGRES)

# Start all services
up-all:
	@echo Starting all services...
	@docker compose -f docker-compose.all.yaml up --build -d
	@echo.
	@echo Services started! Waiting for health checks...
	@$(SLEEP_CMD)
	@docker compose -f docker-compose.all.yaml ps

# Stop all services
down-all:
	@echo Stopping all services...
	@docker compose -f docker-compose.all.yaml down

# View logs
logs:
	@docker compose -f docker-compose.all.yaml logs -f

# Show status
status:
	@docker compose -f docker-compose.all.yaml ps

# Rebuild and restart
rebuild:
	@echo Rebuilding all services...
	@docker compose -f docker-compose.all.yaml down
	@docker compose -f docker-compose.all.yaml up --build -d
	@docker compose -f docker-compose.all.yaml ps

# Clean everything
clean:
	@echo Cleaning up...
	-@docker compose -f docker-compose.all.yaml down -v --remove-orphans
	@$(RM_CMD)
	@echo Cleanup complete!

# Interactive owner configuration setup
setup-owner:
	@echo.
	@echo ========================================================================
	@echo              OWNER REGISTRATION CONFIGURATION SETUP
	@echo ========================================================================
	@echo.
	@echo Current values in .env:
	@echo.
ifeq ($(OS),Windows_NT)
	@powershell -ExecutionPolicy Bypass -Command \
		"$$OwnerFirstName = (Select-String -Path '.env' -Pattern '^OWNER_FIRST_NAME=' -ErrorAction SilentlyContinue).Line; \
		 $$OwnerLastName = (Select-String -Path '.env' -Pattern '^OWNER_LAST_NAME=' -ErrorAction SilentlyContinue).Line; \
		 $$OwnerEmail = (Select-String -Path '.env' -Pattern '^OWNER_EMAIL=' -ErrorAction SilentlyContinue).Line; \
		 $$OwnerPassword = (Select-String -Path '.env' -Pattern '^OWNER_PASSWORD=' -ErrorAction SilentlyContinue).Line; \
		 if ($$OwnerFirstName) { Write-Host $$OwnerFirstName } else { Write-Host 'OWNER_FIRST_NAME=Admin' }; \
		 if ($$OwnerLastName) { Write-Host $$OwnerLastName } else { Write-Host 'OWNER_LAST_NAME=User' }; \
		 if ($$OwnerEmail) { Write-Host $$OwnerEmail } else { Write-Host 'OWNER_EMAIL=admin@example.com' }; \
		 if ($$OwnerPassword) { Write-Host $$OwnerPassword } else { Write-Host 'OWNER_PASSWORD=Admin@123' }"
	@echo.
	@echo Enter new values (press Enter to keep current value):
	@echo.
	@powershell -ExecutionPolicy Bypass -File build/scripts/setup-owner.ps1
else
	@grep "^OWNER_FIRST_NAME\|^OWNER_LAST_NAME\|^OWNER_EMAIL\|^OWNER_PASSWORD" .env || echo "Owner configuration not found"
	@echo.
	@echo Enter new values (press Enter to keep current value):
	@echo.
	@bash build/scripts/setup-owner.sh
endif