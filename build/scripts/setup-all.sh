#!/bin/bash
# ========================================================================
#                    SERENIBASE FULL SETUP SCRIPT
#
#  Full automated setup script with default values (no prompts)
# ========================================================================

set -e

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print functions
print_header() {
    echo -e "${BLUE}"
    echo "========================================================================"
    echo "                     SERENIBASE FULL SETUP"
    echo "========================================================================"
    echo -e "${NC}"
}

print_step() {
    echo -e "${GREEN}[OK]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

print_error() {
    echo -e "${RED}[X]${NC} $1"
}

# Print header
print_header
echo ""

# Initialize .env file with defaults if it doesn't exist
echo -e "${BLUE}Setting up environment configuration...${NC}"
if [ ! -f ".env" ]; then
    echo "Creating .env file from template..."
    cp build/config/.env.example .env
    
    # Set PUBLIC_HOST to localhost default (case-insensitive)
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS uses different sed syntax
        sed -i '' 's/${PUBLIC_HOST}/localhost/g' .env
        sed -i '' 's/${public_host}/localhost/g' .env
    else
        # Linux
        sed -i 's/${PUBLIC_HOST}/localhost/g' .env
        sed -i 's/${public_host}/localhost/g' .env
    fi
    print_step ".env file created with defaults"
else
    print_step ".env file already exists"
fi

echo ""

# Clone main services
echo -e "${BLUE}Cloning all service repos...${NC}"
bash "$SCRIPT_DIR/clone-services.sh"

echo ""

# Clone go-postgres-rest
echo -e "${BLUE}Cloning go-postgres-rest...${NC}"
bash "$SCRIPT_DIR/clone-go-postgres-rest.sh"

echo ""

# Start all services with Docker Compose
echo -e "${BLUE}Starting all services with Docker Compose...${NC}"
docker compose -f docker-compose.all.yaml up --build -d

echo ""
echo -e "${BLUE}========================================================================"
echo "                      SETUP COMPLETE!"
echo "========================================================================${NC}"
echo ""
echo -e "${GREEN}Access your application at:${NC}"
echo "  - Frontend: http://localhost:5050"
echo "  - Backend:  http://localhost:8080"
echo ""
