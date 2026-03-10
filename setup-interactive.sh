#!/usr/bin/env bash
set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TARGET="$SCRIPT_DIR/build/scripts/setup.sh"

if [ ! -f "$TARGET" ]; then
  echo "Setup script not found at $TARGET"
  exit 1
fi

bash "$TARGET" "$@"
