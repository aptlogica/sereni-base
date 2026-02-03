#!/bin/bash
set -e

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

SERVICES_DIR="services"
SERVICES_FILE="services.list"

# Load .env if present - using safer method
if [ -f ".env" ]; then
    # Convert Windows line endings (CRLF) to Unix (LF) if needed, then source
    set -a
    source <(sed 's/\r$//' .env)
    set +a
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
            git -C "$target" pull
            continue
        fi
    fi
    
    # Inject GIT_TOKEN if available
    if [ -n "$GIT_TOKEN" ]; then
        # Use a function to safely append token to URL without sed complications
        # This method avoids issues with special characters in the token
        repo="${repo/https:\/\//https://${GIT_TOKEN}@}"
    fi
    
    # Clone
    echo "CLONING: $name"
    if [ -n "$branch" ]; then
        git clone --branch "$branch" "$repo" "$target"
    else
        git clone "$repo" "$target"
    fi
done < "$SERVICES_FILE"

echo "All services cloned/updated successfully!"
