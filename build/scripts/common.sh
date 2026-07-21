#!/bin/bash
# ========================================================================
#                    SERENIBASE COMMON FUNCTIONS
#                    Shared utilities for setup scripts
# ========================================================================

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ========================================================================
#                           PRINT FUNCTIONS
# ========================================================================

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

print_info() {
    echo -e "${BLUE}[i]${NC} $1"
}

# ========================================================================
#                      PREREQUISITES CHECKING
# ========================================================================

check_prerequisites() {
    echo -e "\n${BLUE}Checking prerequisites...${NC}\n"
    
    local all_satisfied=true
    
    # Check Docker
    if command -v docker &> /dev/null; then
        print_step "Docker is installed: $(docker --version)"
    else
        print_error "Docker is not installed. Please install Docker first."
        all_satisfied=false
    fi
    
    # Check Docker Compose
    if docker compose version &> /dev/null; then
        print_step "Docker Compose is installed: $(docker compose version)"
    else
        print_error "Docker Compose is not installed. Please install Docker Compose first."
        all_satisfied=false
    fi
    
    # Check Git
    if command -v git &> /dev/null; then
        print_step "Git is installed: $(git --version)"
    else
        print_error "Git is not installed. Please install Git first."
        all_satisfied=false
    fi
    
    # Check Make (optional)
    if command -v make &> /dev/null; then
        print_step "Make is installed"
    else
        print_warning "Make is not installed. You can still use docker compose directly."
    fi
    
    if [ "$all_satisfied" = false ]; then
        exit 1
    fi
}

# ========================================================================
#                   ENVIRONMENT SETUP FUNCTIONS
# ========================================================================

# Update a single environment variable in .env file
update_env_var() {
    local var_name="$1"
    local var_value="$2"
    local env_file="${3:-.env}"
    
    # Escape special characters for sed
    local escaped_value=$(printf '%s\n' "$var_value" | sed -e 's/[&/\]/\\&/g')
    
    if grep -q "^${var_name}=" "$env_file" 2>/dev/null; then
        # Update existing variable
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/^${var_name}=.*/${var_name}=${escaped_value}/" "$env_file"
        else
            sed -i "s/^${var_name}=.*/${var_name}=${escaped_value}/" "$env_file"
        fi
    else
        # Append new variable
        echo "${var_name}=${var_value}" >> "$env_file"
    fi
}

# Convert file to Unix line endings
convert_to_unix_line_endings() {
    local file="${1:-.env}"
    
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' 's/\r$//' "$file" 2>/dev/null || true
    else
        sed -i 's/\r$//' "$file" 2>/dev/null || true
    fi
}

# ========================================================================
#                      REPOSITORY CLONING
# ========================================================================

clone_repositories() {
    echo -e "\n${BLUE}Cloning repositories...${NC}\n"
    
    chmod +x build/scripts/*.sh 2>/dev/null || true
    
    if [ -f "build/scripts/clone-services.sh" ]; then
        print_info "Cloning microservices..."
        bash build/scripts/clone-services.sh
        print_step "Cloned microservices"
    fi
    
    if [ -f "build/scripts/clone-go-postgres-rest.sh" ]; then
        print_info "Cloning go-postgres-rest..."
        bash build/scripts/clone-go-postgres-rest.sh
        print_step "Cloned go-postgres-rest"
    fi
}

# ========================================================================
#                      DOCKER OPERATIONS
# ========================================================================

start_docker_services() {
    echo -e "\n${BLUE}Starting services...${NC}\n"
    
    docker compose -f docker-compose.all.yaml up --build -d
    
    print_step "Services started!"
    
    echo -e "\n${BLUE}Waiting for services to be ready...${NC}"
    sleep 10
    
    docker compose -f docker-compose.all.yaml ps
}

stop_docker_services() {
    if docker compose -f docker-compose.all.yaml ps -q 2>/dev/null | grep -q .; then
        echo -e "${YELLOW}[!] Stopping Docker containers...${NC}"
        docker compose -f docker-compose.all.yaml down 2>/dev/null || true
    fi
}

# ========================================================================
#                      COMPLETION MESSAGE
# ========================================================================

print_completion() {
    local public_host="${1:-localhost}"
    local owner_email="${2:-admin@example.com}"
    local owner_password="${3:-Admin@123}"
    
    echo -e "\n${GREEN}"
    echo "========================================================================"
    echo "                      SETUP COMPLETE!"
    echo "========================================================================"
    echo -e "${NC}"
    
    echo -e "Access your application at:"
    echo -e "  ${GREEN}Frontend:${NC}  http://$public_host:5050"
    echo -e "  ${GREEN}Backend:${NC}   http://$public_host:8080"
    echo -e "  ${GREEN}RustFS:${NC}     http://$public_host:9001"
    echo ""
    echo -e "Default admin credentials:"
    echo -e "  ${YELLOW}Email:${NC}    $owner_email"
    echo -e "  ${YELLOW}Password:${NC} $owner_password"
    echo ""
    echo -e "${YELLOW}WARNING: Remember to change default passwords in production!${NC}"
    echo ""
    echo "Useful commands:"
    echo "  make logs      - View service logs"
    echo "  make down-all  - Stop all services"
    echo "  make clean     - Remove all data"
}

# ========================================================================
#                      CLEANUP HANDLER
# ========================================================================

setup_cleanup_handler() {
    cleanup() {
        echo ""
        echo -e "${YELLOW}[!] Setup interrupted by user. Cleaning up...${NC}"
        
        stop_docker_services
        
        # Kill any background processes started by this script
        jobs -p 2>/dev/null | xargs -r kill 2>/dev/null || true
        
        echo -e "${RED}[X] Setup cancelled.${NC}"
        exit 1
    }
    
    # Trap Ctrl+C (SIGINT) and other termination signals
    trap cleanup SIGINT SIGTERM
}

