# Cross-platform Makefile for SereniBase
# Works on Linux, macOS, and Windows
#
# All build scripts are located in: build/scripts/
# Configuration templates are in:   build/config/

# Stop on first error - Ctrl+C will stop the entire Make process
.ONESHELL:
.SHELLFLAGS := -e

.PHONY: setup setup-y

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SHELL := cmd.exe
    SETUP_SCRIPT := build\scripts\setup.bat
    SETUP_SCRIPT_Y := build\scripts\setup-y.bat
else
    DETECTED_OS := $(shell uname -s)
    SHELL := /bin/bash
    SETUP_SCRIPT := bash build/scripts/setup.sh
    SETUP_SCRIPT_Y := bash build/scripts/setup-y.sh
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
	@echo     make setup-y            - Full setup with default values (no prompts)
	@echo.
	@echo   Documentation:
	@echo     See docs/ENV_CONFIGURATION.md for environment setup
	@echo.

# Interactive setup wizard
setup:
	@echo Starting SereniBase Setup Wizard...
	@$(SETUP_SCRIPT)

# Full automated setup with defaults (no prompts)
setup-y:
	@echo Starting SereniBase Setup with defaults (no prompts)...
	@$(SETUP_SCRIPT_Y)