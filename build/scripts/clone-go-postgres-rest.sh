#!/bin/bash
set -e

# Cleanup function to handle Ctrl+C
cleanup() {
    echo ""
    echo "[!] Clone interrupted by user."
    exit 1
}

# Trap Ctrl+C (SIGINT) and other termination signals
trap cleanup SIGINT SIGTERM

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

# Load .env if present for GIT_TOKEN - using safer method
if [ -f ".env" ]; then
    # Only load specific variables we need, avoiding issues with special characters
    GIT_TOKEN=$(grep "^GIT_TOKEN=" .env | cut -d'=' -f2- | tr -d '\r')
fi

# Debug: Check if GIT_TOKEN is loaded
if [ -z "$GIT_TOKEN" ]; then
    echo "[INFO] GIT_TOKEN not set, cloning without authentication"
else
    echo "[INFO] GIT_TOKEN is set"
fi

REPO_URL="https://github.com/aptlogica/go-postgres-rest.git"
TARGET_DIR="go-postgres-rest"

# Always remove and re-clone for a clean state
if [ -d "$TARGET_DIR" ]; then
    echo "Removing existing $TARGET_DIR..."
    rm -rf "$TARGET_DIR"
fi

# Inject GIT_TOKEN if available
if [ -n "$GIT_TOKEN" ]; then
    # Use bash string substitution instead of sed to avoid issues with special characters
    REPO_URL="${REPO_URL/https:\/\//https://${GIT_TOKEN}@}"
fi

echo "Cloning $REPO_URL into $TARGET_DIR..."
git clone "$REPO_URL" "$TARGET_DIR"

# Clean Go module cache (if go is available)
if command -v go &> /dev/null; then
    echo "Cleaning Go module cache..."
    go clean -modcache
fi

echo "go-postgres-rest cloned successfully!"
