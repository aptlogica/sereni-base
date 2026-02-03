#!/bin/bash
# ========================================================================
#                    SERENIBASE SETUP SCRIPT
#
#  Interactive setup script to configure and deploy SereniBase
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

# Check prerequisites
check_prerequisites() {
    echo -e "\n${BLUE}Checking prerequisites...${NC}\n"
    
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
    
    # Check Make
    if command -v make &> /dev/null; then
        print_step "Make is installed"
    else
        print_warning "Make is not installed. You can still use docker compose directly."
    fi
}

# Setup environment
setup_environment() {
    echo -e "\n${BLUE}Setting up environment...${NC}\n"
    
    if [ ! -f ".env" ]; then
        if [ -f "build/config/.env.example" ]; then
            cp build/config/.env.example .env
            # Convert Windows CRLF to Unix LF line endings
            sed -i 's/\r$//' .env 2>/dev/null || sed -i '' 's/\r$//' .env
            print_step "Created .env from build/config/.env.example"
        else
            print_error "build/config/.env.example not found!"
            exit 1
        fi
    else
        # Also convert existing .env file to Unix line endings
        sed -i 's/\r$//' .env 2>/dev/null || sed -i '' 's/\r$//' .env
        print_warning ".env already exists. Skipping creation."
    fi
}

# Configure public host
configure_host() {
    echo -e "\n${BLUE}Network Configuration${NC}\n"
    
    # Detect local IP
    local_ip=$(hostname -I 2>/dev/null | awk '{print $1}' || ipconfig getifaddr en0 2>/dev/null || echo "localhost")
    
    echo "Your detected local IP: $local_ip"
    echo ""
    echo "How would you like to access SereniBase?"
    echo "  1) localhost (local development only)"
    echo "  2) $local_ip (LAN access)"
    echo "  3) Custom IP/domain"
    echo ""
    read -p "Enter choice [1-3]: " choice
    
    case $choice in
        1)
            PUBLIC_HOST="localhost"
            ;;
        2)
            PUBLIC_HOST="$local_ip"
            ;;
        3)
            read -p "Enter your IP or domain: " PUBLIC_HOST
            ;;
        *)
            PUBLIC_HOST="localhost"
            ;;
    esac
    
    # Update .env file
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^PUBLIC_HOST=.*/PUBLIC_HOST=$PUBLIC_HOST/" .env
    else
        sed -i "s/^PUBLIC_HOST=.*/PUBLIC_HOST=$PUBLIC_HOST/" .env
    fi
    
    print_step "Configured PUBLIC_HOST=$PUBLIC_HOST"
}

# Configure owner registration
configure_owner() {
    echo -e "\n${BLUE}Owner Registration Configuration${NC}\n"
    
    echo "Enter owner registration details (press Enter to use defaults):"
    echo ""
    
    read -p "First Name [Admin]: " OWNER_FIRST_NAME
    if [ -z "$OWNER_FIRST_NAME" ]; then
        OWNER_FIRST_NAME="Admin"
    fi
    
    read -p "Last Name [User]: " OWNER_LAST_NAME
    if [ -z "$OWNER_LAST_NAME" ]; then
        OWNER_LAST_NAME="User"
    fi
    
    read -p "Email [admin@example.com]: " OWNER_EMAIL
    if [ -z "$OWNER_EMAIL" ]; then
        OWNER_EMAIL="admin@example.com"
    fi
    
    read -p "Password [Admin@123]: " OWNER_PASSWORD
    if [ -z "$OWNER_PASSWORD" ]; then
        OWNER_PASSWORD="Admin@123"
    fi
    
    # Update .env file
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/^OWNER_FIRST_NAME=.*/OWNER_FIRST_NAME=$OWNER_FIRST_NAME/" .env
        sed -i '' "s/^OWNER_LAST_NAME=.*/OWNER_LAST_NAME=$OWNER_LAST_NAME/" .env
        sed -i '' "s/^OWNER_EMAIL=.*/OWNER_EMAIL=$OWNER_EMAIL/" .env
        sed -i '' "s/^OWNER_PASSWORD=.*/OWNER_PASSWORD=$OWNER_PASSWORD/" .env
    else
        sed -i "s/^OWNER_FIRST_NAME=.*/OWNER_FIRST_NAME=$OWNER_FIRST_NAME/" .env
        sed -i "s/^OWNER_LAST_NAME=.*/OWNER_LAST_NAME=$OWNER_LAST_NAME/" .env
        sed -i "s/^OWNER_EMAIL=.*/OWNER_EMAIL=$OWNER_EMAIL/" .env
        sed -i "s/^OWNER_PASSWORD=.*/OWNER_PASSWORD=$OWNER_PASSWORD/" .env
    fi
    
    print_step "Owner configuration set"
}

# Clone repositories
clone_repos() {
    echo -e "\n${BLUE}Cloning repositories...${NC}\n"
    
    chmod +x build/scripts/*.sh 2>/dev/null || true
    
    if [ -f "build/scripts/clone-services.sh" ]; then
        bash build/scripts/clone-services.sh
        print_step "Cloned microservices"
    fi
    
    if [ -f "build/scripts/clone-go-postgres-rest.sh" ]; then
        bash build/scripts/clone-go-postgres-rest.sh
        print_step "Cloned go-postgres-rest"
    fi
}

# Start services
start_services() {
    echo -e "\n${BLUE}Starting services...${NC}\n"
    
    docker compose -f docker-compose.all.yaml up --build -d
    
    print_step "Services started!"
    
    echo -e "\n${BLUE}Waiting for services to be ready...${NC}"
    sleep 10
    
    docker compose -f docker-compose.all.yaml ps
}

# Print completion message
print_completion() {
    echo -e "\n${GREEN}"
    echo "========================================================================"
    echo "                      SETUP COMPLETE!"
    echo "========================================================================"
    echo -e "${NC}"
    
    echo -e "Access your application at:"
    echo -e "  ${GREEN}Frontend:${NC}  http://$PUBLIC_HOST:5050"
    echo -e "  ${GREEN}Backend:${NC}   http://$PUBLIC_HOST:8080"
    echo -e "  ${GREEN}MinIO:${NC}     http://$PUBLIC_HOST:9001"
    echo ""
    echo -e "Default admin credentials:"
    echo -e "  ${YELLOW}Email:${NC}    admin@example.com"
    echo -e "  ${YELLOW}Password:${NC} Admin@123"
    echo ""
    echo -e "${YELLOW}WARNING: Remember to change default passwords in production!${NC}"
    echo ""
    echo "Useful commands:"
    echo "  make logs      - View service logs"
    echo "  make down-all  - Stop all services"
    echo "  make clean     - Remove all data"
}

# Main execution
main() {
    print_header
    check_prerequisites
    setup_environment
    configure_host
    configure_owner
    clone_repos
    start_services
    print_completion
}

# Run main function
main
