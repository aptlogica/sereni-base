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

SERVICES_DIR="services"
SERVICES_FILE="build/scripts/services.list"

# Load .env if present - using safer method
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

# Create services directory if missing
mkdir -p "$SERVICES_DIR"

# Process services
while IFS= read -r line || [ -n "$line" ]; do
    # Skip empty lines
    [ -z "$line" ] && continue
    
    # Parse line: name repo [branch]
    name=$(echo "$line" | awk '{print $1}')
    repo=$(echo "$line" | awk '{print $2}')
    branch=$(echo "$line" | awk '{print $3}')
    target="$SERVICES_DIR/$name"
    
    # Repo exists -> check remote and branch
    if [ -d "$target/.git" ]; then
        need_reclone=false
        current_url=$(git -C "$target" remote get-url origin 2>/dev/null || echo "")
        current_branch=$(git -C "$target" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")
        
        if [ "$current_url" != "$repo" ]; then
            echo "REMOTE URL mismatch for $name. Re-cloning..."
            need_reclone=true
        elif [ -n "$branch" ] && [ "$current_branch" != "$branch" ]; then
            echo "BRANCH mismatch for $name. Re-cloning..."
            need_reclone=true
        fi
        
        if [ "$need_reclone" = true ]; then
            rm -rf "$target"
        else
            echo "PULLING: $name"
            # Inject GIT_TOKEN for pull operation if available
            if [ -n "$GIT_TOKEN" ]; then
                # Construct authenticated URL for pull
                auth_repo="${repo/https:\/\//https://${GIT_TOKEN}@}"
                # Temporarily set remote URL with token
                git -C "$target" remote set-url origin "$auth_repo"
                git -C "$target" pull
                # Restore original URL without token (for security)
                git -C "$target" remote set-url origin "$repo"
            else
                git -C "$target" pull
            fi
            continue
        fi
    fi
    
    # Inject GIT_TOKEN if available
    if [ -n "$GIT_TOKEN" ]; then
        # Use a function to safely append token to URL without sed complications
        # This method avoids issues with special characters in the token
        auth_repo="${repo/https:\/\//https://${GIT_TOKEN}@}"
    else
        auth_repo="$repo"
    fi
    
    # Clone
    echo "CLONING: $name"
    if [ -n "$branch" ]; then
        git clone --branch "$branch" "$auth_repo" "$target"
    else
        git clone "$auth_repo" "$target"
    fi
    
    # Remove token from remote URL (for security)
    if [ -n "$GIT_TOKEN" ]; then
        git -C "$target" remote set-url origin "$repo"
    fi
done < "$SERVICES_FILE"

echo "All services cloned/updated successfully!"
