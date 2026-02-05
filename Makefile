# Cross-platform Makefile for SereniBase
# Works on Linux, macOS, and Windows
#
# All build scripts are located in: build/scripts/
# Configuration templates are in:   build/config/

.PHONY: setup setup-y help

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    SETUP_SCRIPT := build\scripts\setup.bat
    SETUP_SCRIPT_Y := build\scripts\setup-y.bat
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
	@echo "    make setup-y            - Full setup with default values (no prompts)"
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
	@$(SETUP_SCRIPT)
else
	@chmod +x $(SETUP_SCRIPT)
	@bash $(SETUP_SCRIPT)
endif

# Full automated setup with defaults (no prompts)
setup-y:
	@echo "Starting SereniBase Setup with defaults (no prompts)..."
ifeq ($(OS),Windows_NT)
	@$(SETUP_SCRIPT_Y)
else
	@chmod +x $(SETUP_SCRIPT_Y)
	@bash $(SETUP_SCRIPT_Y)
endif