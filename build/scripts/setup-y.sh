#!/bin/bash
# ========================================================================
#                    SERENIBASE SETUP SCRIPT (AUTO-YES MODE)
#
#  Full automated setup with default values
#  Supports all parameters that setup.sh supports
#
#  Priority for environment variables:
#    1. Script command-line arguments (highest priority)
#    2. Existing values from .env file (protected - never overridden)
#    3. Default variable values (lowest priority)
#
#  Usage Examples:
#    ./setup-y.sh
#    ./setup-y.sh --public-host "192.168.1.100"
#    ./setup-y.sh --database-port "5433" --smtp-host "smtp.gmail.com"
#
# ========================================================================

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Pass all arguments to setup.sh with --auto-yes flag prepended
exec "$SCRIPT_DIR/setup.sh" --auto-yes "$@"
