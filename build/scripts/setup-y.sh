#!/bin/bash
# ========================================================================
#                    SERENIBASE SETUP SCRIPT (NO PROMPTS)
#
#  Full automated setup with default values - same as interactive setup
#  but without prompting the user
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
    echo "                     SERENIBASE SETUP WIZARD"
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

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"
echo ""

# Check Docker
if command -v docker &> /dev/null; then
    print_step "Docker is installed: $(docker --version)"
else
    print_error "Docker is not installed. Please install Docker first."
    exit 1
fi

# Check Docker Compose
if docker compose version &> /dev/null; then
    print_step "Docker Compose is installed: $(docker compose version)"
else
    print_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Check Git
if command -v git &> /dev/null; then
    print_step "Git is installed: $(git --version)"
else
    print_error "Git is not installed. Please install Git first."
    exit 1
fi

echo ""
echo -e "${GREEN}All prerequisites satisfied!${NC}"
echo ""

# Setup environment
echo -e "${BLUE}Setting up environment...${NC}"
if [ ! -f ".env" ]; then
    if [ -f "build/config/.env.example" ]; then
        cp build/config/.env.example .env
        print_step "Created .env from build/config/.env.example"
    else
        print_error "build/config/.env.example not found!"
        exit 1
    fi
else
    print_warning ".env already exists. Skipping creation."
fi

echo ""
echo -e "${BLUE}========================================================================"
echo "                      NETWORK CONFIGURATION"
echo "========================================================================${NC}"
echo ""

PUBLIC_HOST="localhost"
echo "Using default IP/domain: $PUBLIC_HOST"
echo ""

# Update .env file with PUBLIC_HOST
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/^PUBLIC_HOST=.*/PUBLIC_HOST=$PUBLIC_HOST/" .env
else
    sed -i "s/^PUBLIC_HOST=.*/PUBLIC_HOST=$PUBLIC_HOST/" .env
fi
print_step "Configured PUBLIC_HOST=$PUBLIC_HOST"

echo ""
echo -e "${BLUE}========================================================================"
echo "                   OWNER REGISTRATION CONFIGURATION"
echo "========================================================================${NC}"
echo ""
echo "Using default values:"
echo ""

OWNER_FIRST_NAME="Admin"
OWNER_LAST_NAME="User"
OWNER_EMAIL="admin@example.com"
OWNER_PASSWORD="Admin@123"

echo "   First Name: $OWNER_FIRST_NAME"
echo "   Last Name:  $OWNER_LAST_NAME"
echo "   Email:      $OWNER_EMAIL"
echo "   Password:   $OWNER_PASSWORD"
echo ""

# Update .env file with owner configuration
if grep -q "^OWNER_FIRST_NAME=" .env; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^OWNER_FIRST_NAME=.*/OWNER_FIRST_NAME=$OWNER_FIRST_NAME/" .env
    else
        sed -i "s/^OWNER_FIRST_NAME=.*/OWNER_FIRST_NAME=$OWNER_FIRST_NAME/" .env
    fi
else
    echo "OWNER_FIRST_NAME=$OWNER_FIRST_NAME" >> .env
fi

if grep -q "^OWNER_LAST_NAME=" .env; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^OWNER_LAST_NAME=.*/OWNER_LAST_NAME=$OWNER_LAST_NAME/" .env
    else
        sed -i "s/^OWNER_LAST_NAME=.*/OWNER_LAST_NAME=$OWNER_LAST_NAME/" .env
    fi
else
    echo "OWNER_LAST_NAME=$OWNER_LAST_NAME" >> .env
fi

if grep -q "^OWNER_EMAIL=" .env; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^OWNER_EMAIL=.*/OWNER_EMAIL=$OWNER_EMAIL/" .env
    else
        sed -i "s/^OWNER_EMAIL=.*/OWNER_EMAIL=$OWNER_EMAIL/" .env
    fi
else
    echo "OWNER_EMAIL=$OWNER_EMAIL" >> .env
fi

if grep -q "^OWNER_PASSWORD=" .env; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^OWNER_PASSWORD=.*/OWNER_PASSWORD=$OWNER_PASSWORD/" .env
    else
        sed -i "s/^OWNER_PASSWORD=.*/OWNER_PASSWORD=$OWNER_PASSWORD/" .env
    fi
else
    echo "OWNER_PASSWORD=$OWNER_PASSWORD" >> .env
fi

print_step "Owner configuration set"

echo ""
echo -e "${BLUE}========================================================================"
echo "                      CLONING REPOSITORIES"
echo "========================================================================${NC}"
echo ""

if [ -f "build/scripts/clone-services.sh" ]; then
    echo "Cloning microservices..."
    bash "$SCRIPT_DIR/clone-services.sh"
    print_step "Cloned microservices"
fi

if [ -f "build/scripts/clone-go-postgres-rest.sh" ]; then
    echo "Cloning go-postgres-rest..."
    bash "$SCRIPT_DIR/clone-go-postgres-rest.sh"
    print_step "Cloned go-postgres-rest"
fi

echo ""
echo -e "${BLUE}========================================================================"
echo "                      STARTING SERVICES"
echo "========================================================================${NC}"
echo ""

docker compose -f docker-compose.all.yaml up --build -d

echo ""
echo "Waiting for services to start..."
sleep 10

docker compose -f docker-compose.all.yaml ps

echo ""
echo -e "${BLUE}========================================================================"
echo "                      SETUP COMPLETE!"
echo "========================================================================${NC}"
echo ""
echo -e "${GREEN}Access your application at:${NC}"
echo "  Frontend:  http://$PUBLIC_HOST:5050"
echo "  Backend:   http://$PUBLIC_HOST:8080"
echo "  MinIO:     http://$PUBLIC_HOST:9001"
echo ""
echo -e "${GREEN}Default admin credentials:${NC}"
echo "  Email:    $OWNER_EMAIL"
echo "  Password: $OWNER_PASSWORD"
echo ""
echo -e "${YELLOW}WARNING: Remember to change default passwords in production!${NC}"
echo ""
echo -e "${GREEN}Useful commands:${NC}"
echo "  make logs      - View service logs"
echo "  make down-all  - Stop all services"
echo "  make clean     - Remove all data"
echo ""
