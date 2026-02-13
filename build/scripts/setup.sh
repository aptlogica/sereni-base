#!/bin/bash
# ========================================================================
#                    SERENIBASE SETUP SCRIPT
#
#  Interactive setup script to configure and deploy SereniBase
#
#  Priority for environment variables:
#    1. Script command-line arguments (highest priority)
#    2. Existing values from .env file (if exists)
#    3. Default variable values (lowest priority)
#
#  Usage:
#    ./setup.sh                    # Interactive mode with priority defaults
#    ./setup.sh --auto-yes         # Non-interactive with all defaults
#    ./setup.sh --smtp-host="..." --smtp-port="..." [other args]
# ========================================================================

# Don't use 'set -e' to allow proper Ctrl+C handling
# set -e  # Commented out to handle Ctrl+C gracefully

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

# ========================================================================
# PARAMETER PARSING - Store all script arguments (bash 3.2 compatible)
# ========================================================================
SCRIPT_ARGS=""
AUTO_YES=false

# Helper function to store script argument (bash 3.2 compatible)
set_script_arg() {
    local key="$1"
    local value="$2"
    SCRIPT_ARGS="${SCRIPT_ARGS}${key}:::${value}|||"
}

# Helper function to get script argument (bash 3.2 compatible)
get_script_arg() {
    local key="$1"
    local pattern="${key}:::"
    
    if [[ "$SCRIPT_ARGS" == *"$pattern"* ]]; then
        # Extract the value between key::: and |||
        local temp="${SCRIPT_ARGS#*$pattern}"
        local value="${temp%%|||*}"
        echo "$value"
        return 0
    fi
    return 1
}

while [[ $# -gt 0 ]]; do
    case $1 in
        --auto-yes)
            AUTO_YES=true
            shift
            ;;
        --*=*)
            # Handle --key=value format
            key="${1#--}"
            key="${key%=*}"
            value="${1#*=}"
            set_script_arg "$key" "$value"
            shift
            ;;
        --*)
            # Handle --key value format
            key="${1#--}"
            if [[ $# -lt 2 ]]; then
                echo "Error: $1 requires a value"
                exit 1
            fi
            set_script_arg "$key" "$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--auto-yes] [--key=value ...]"
            exit 1
            ;;
    esac
done

# Cleanup function to stop all processes on Ctrl+C
cleanup() {
    local exit_code=$?
    
    # Only cleanup if interrupted (exit code 130 = Ctrl+C or SIGINT)
    # Don't cleanup on successful completion
    if [ $exit_code -eq 0 ]; then
        return 0
    fi
    
    echo ""
    echo -e "${YELLOW}[!] Setup interrupted by user (Ctrl+C). Cleaning up...${NC}"
    
    # Stop any running docker containers started by this script
    if docker compose -f docker-compose.all.yaml ps -q 2>/dev/null | grep -q .; then
        echo -e "${YELLOW}[!] Stopping Docker containers...${NC}"
        docker compose -f docker-compose.all.yaml down 2>/dev/null || true
    fi
    
    # Kill any background processes started by this script
    local jobs_list=$(jobs -p 2>/dev/null)
    if [ -n "$jobs_list" ]; then
        echo -e "${YELLOW}[!] Killing background processes...${NC}"
        echo "$jobs_list" | xargs kill -9 2>/dev/null || true
    fi
    
    echo -e "${RED}[X] Setup cancelled. All processes stopped.${NC}"
    exit 130  # Standard exit code for Ctrl+C
}

# Trap Ctrl+C (SIGINT) and other termination signals
trap cleanup SIGINT SIGTERM SIGHUP

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

# Update a single environment variable in .env (overwrite if exists)
update_env_var() {
    local var_name="$1"
    local var_value="$2"

    # Escape special characters for sed replacement
    local escaped_value
    escaped_value=$(printf '%s\n' "$var_value" | sed -e 's/[\/&|\\]/\\&/g')

    if grep -q "^${var_name}=" .env 2>/dev/null; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|^${var_name}=.*|${var_name}=${escaped_value}|" .env
        else
            sed -i "s|^${var_name}=.*|${var_name}=${escaped_value}|" .env
        fi
    else
        echo "${var_name}=${var_value}" >> .env
    fi
}

# Get existing environment variable value from .env file
get_env_var() {
    local var_name="$1"
    if [ ! -f ".env" ]; then
        echo ""
        return
    fi
    local value=$(grep -E "^${var_name}=" .env 2>/dev/null | tail -n 1 | cut -d'=' -f2- | tr -d '\r')
    echo "$value"
}

# Resolve environment variable value with priority system
# Priority 1: Script argument (highest)
# Priority 2: Existing .env value
# Priority 3: Default value (lowest)
resolve_env_var() {
    local var_name="$1"
    local default_value="$2"
    
    # Priority 1: Check script arguments
    local script_value
    script_value=$(get_script_arg "$var_name" 2>/dev/null)
    if [ -n "$script_value" ]; then
        echo "$script_value"
        return
    fi
    
    # Priority 2: Check existing .env file
    local existing_value=$(get_env_var "$var_name")
    if [ -n "$existing_value" ]; then
        echo "$existing_value"
        return
    fi
    
    # Priority 3: Use default value
    echo "$default_value"
}

# Prompt for value with priority system
# Priority 1: Script argument (highest - can override anything)
# Priority 2: Existing .env value (protected - never prompted, never overridden)
# Priority 3: Default value (lowest - only used if no .env value)
prompt_env_var() {
    local var_name="$1"
    local default_value="$2"
    local prompt_text="$3"
    local is_password="${4:-false}"
    
    # Priority 1: If script argument provided, use it (can override .env)
    local script_value
    script_value=$(get_script_arg "$var_name" 2>/dev/null)
    if [ -n "$script_value" ]; then
        echo "$script_value"
        return
    fi
    
    # Priority 2: If value exists in .env, use it SILENTLY (never override)
    local existing_value=$(get_env_var "$var_name")
    if [ -n "$existing_value" ]; then
        echo "$existing_value"
        return
    fi
    
    # Priority 3: Value doesn't exist in .env, prompt or use default
    
    # If in auto-yes mode, use default without prompting
    if [ "$AUTO_YES" = "true" ]; then
        echo "$default_value"
        return
    fi
    
    # Interactive mode: prompt user for new value
    local prompt_msg="$prompt_text [$default_value]: "
    
    if [ "$is_password" = "true" ]; then
        read -s -p "$prompt_msg" user_input
        echo ""
    else
        read -p "$prompt_msg" user_input
    fi
    
    # Return user input if provided, otherwise return the default
    if [ -n "$user_input" ]; then
        echo "$user_input"
    else
        echo "$default_value"
    fi
}

# Update env var only if the new value is different from existing
update_env_var_if_changed() {
    local var_name="$1"
    local new_value="$2"
    
    local existing_value=$(get_env_var "$var_name")
    
    # Only update if the value is different or doesn't exist yet
    if [ "$existing_value" != "$new_value" ]; then
        update_env_var "$var_name" "$new_value"
    fi
}

# Check if a variable already exists in .env (NEVER override if exists)
var_exists_in_env() {
    local var_name="$1"
    local existing_value=$(get_env_var "$var_name")
    if [ -n "$existing_value" ]; then
        return 0  # True - variable exists
    fi
    return 1  # False - variable doesn't exist
}

# Check if ALL variables in the list exist in .env
all_vars_exist_in_env() {
    for var_name in "$@"; do
        if ! var_exists_in_env "$var_name"; then
            return 1  # At least one variable missing
        fi
    done
    return 0  # All variables exist
}

# Ensure CORS_ALLOWED_ORIGINS includes PUBLIC_HOST:5050
ensure_cors_origin() {
    local host="$1"
    local required_origin="http://${host}:5050"
    local current_origins

    current_origins=$(grep -E '^CORS_ALLOWED_ORIGINS=' .env 2>/dev/null | tail -n 1 | cut -d'=' -f2-)

    if [ -z "$current_origins" ]; then
        update_env_var "CORS_ALLOWED_ORIGINS" "http://localhost:5050,http://127.0.0.1:5050,http://${host}:5050,http://base-ui:5050,http://serenibase:8080"
        return
    fi

    if ! echo "$current_origins" | tr ',' '\n' | grep -Fxq "$required_origin"; then
        update_env_var "CORS_ALLOWED_ORIGINS" "${current_origins},${required_origin}"
    fi
}

# Ensure BASEUI_VITE_API_BASE_URL matches PUBLIC_HOST
ensure_baseui_api_base_url() {
    local host="$1"
    local desired_url="http://${host}:8080"
    local current_url

    current_url=$(grep -E '^BASEUI_VITE_API_BASE_URL=' .env 2>/dev/null | tail -n 1 | cut -d'=' -f2-)

    if [ -z "$current_url" ] || [ "$current_url" != "$desired_url" ]; then
        update_env_var "BASEUI_VITE_API_BASE_URL" "$desired_url"
    fi
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

# Setup environment - Create .env if not exists, merge with template
setup_environment() {
    echo -e "\n${BLUE}Setting up environment...${NC}\n"
    
    # Create .env template with all default variables (using ASCII for compatibility)
    cat > .env.template << 'EOF'
# ==============================================================================
#                         SERENIBASE CONFIGURATION
#                  Generated by Interactive Setup Script
# ==============================================================================

# ------------------------------------------------------------------------------
#                           NETWORK CONFIGURATION
# ------------------------------------------------------------------------------

PUBLIC_HOST=localhost

# ------------------------------------------------------------------------------
#                           SERVER CONFIGURATION
# ------------------------------------------------------------------------------

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30
SERVER_ENV=dev
SERVER_SCHEME=http

# ------------------------------------------------------------------------------
#                           DATABASE CONFIGURATION
# ------------------------------------------------------------------------------

DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=serenibase
DATABASE_SSL_MODE=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=1h

# ------------------------------------------------------------------------------
#                           AUTHENTICATION CONFIGURATION
# ------------------------------------------------------------------------------

AUTH_URL=http://jwt-provider:8081
AUTH_RESET_PASSWORD_URL=http://localhost:5050/reset-password?token=%s
AUTH_JWT_SECRET=change-this-to-a-secure-random-string-min32chars
AUTH_PORT=8081
AUTH_HOST=0.0.0.0
AUTH_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050
AUTH_ENV=development
AUTH_LOG_LEVEL=info

# ------------------------------------------------------------------------------
#                           ADMIN ACCOUNT
# ------------------------------------------------------------------------------

OWNER_FIRST_NAME=Admin
OWNER_LAST_NAME=User
OWNER_EMAIL=admin@example.com
OWNER_PASSWORD=Admin@123
TEMPORARY_USER_PASSWORD=FC4i;<S8q?~0

# ------------------------------------------------------------------------------
#                           EMAIL CONFIGURATION
# ------------------------------------------------------------------------------

EMAIL_URL=http://email-service:8082/api/v1/email
EMAIL_HOST=0.0.0.0
EMAIL_PORT=8082
EMAIL_ALLOWED_ORIGIN=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050
EMAIL_SMTP_HOST=
EMAIL_SMTP_PORT=
EMAIL_SMTP_USERNAME=
EMAIL_SMTP_PASSWORD=
EMAIL_FROM_EMAIL=

# ------------------------------------------------------------------------------
#                           STORAGE CONFIGURATION
# ------------------------------------------------------------------------------

STORAGE_URL=http://sereni-storage-provider:8083/api/v1
STORAGE_SERVER_PORT=8083
STORAGE_SERVER_HOST=0.0.0.0
STORAGE_SERVER_SCHEME=http
STORAGE_DRIVER=minio
STORAGE_DEV_PATH=./uploads
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=serenibase
STORAGE_AWS_ACCESS_KEY=your-access-key
STORAGE_AWS_SECRET_KEY=your-secret-key
STORAGE_MINIO_ENDPOINT=minio:9000
STORAGE_MINIO_ACCESS_KEY=minioadmin
STORAGE_MINIO_SECRET_KEY=minioadmin
STORAGE_MINIO_BUCKET=serenibase
STORAGE_MINIO_USE_SSL=false
STORAGE_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050

# ------------------------------------------------------------------------------
#                           ANTIVIRUS CONFIGURATION
# ------------------------------------------------------------------------------

ANTIVIRUS_URL=http://antivirus-service:8084
ANTIVIRUS_HOST=0.0.0.0
ANTIVIRUS_PORT=8084
ANTIVIRUS_BASE_URL=http://antivirus-service:8084
ANTIVIRUS_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050
ANTIVIRUS_DRIVER=clamav
ANTIVIRUS_CLAMAV_ADDRESS=clamav:3310
ANTIVIRUS_CLAMAV_TIMEOUT_SECONDS=30
ANTIVIRUS_MAX_UPLOAD_SIZE_MB=32

# ------------------------------------------------------------------------------
#                           FRONTEND CONFIGURATION
# ------------------------------------------------------------------------------

BASEUI_VITE_API_BASE_URL=http://localhost:8080

# ------------------------------------------------------------------------------
#                           CORS CONFIGURATION
# ------------------------------------------------------------------------------

CORS_ALLOWED_ORIGINS=http://localhost:5050,http://127.0.0.1:5050,http://base-ui:5050,http://serenibase:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
CORS_ALLOWED_HEADERS=Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization,accept,origin,Cache-Control,X-Requested-With,schema,workspace,base
CORS_ALLOW_CREDENTIALS=true

# ------------------------------------------------------------------------------
#                           LOGGING CONFIGURATION
# ------------------------------------------------------------------------------

LOG_LEVEL=info
LOG_FILE=app.log
LOG_MAX_SIZE=50
LOG_MAX_BACKUPS=10
LOG_MAX_AGE=30
LOG_COMPRESS=true

# ------------------------------------------------------------------------------
#                           ASSET CONFIGURATION
# ------------------------------------------------------------------------------

ASSET_MAX_SIZE=5242880
EOF
    
    if [ ! -f ".env" ]; then
        # .env doesn't exist, create it from template
        cp .env.template .env
        print_step "Created .env with environment variables"
    else
        # .env exists, merge with template (add missing variables)
        print_warning ".env already exists. Checking for missing variables..."
        
        # Read existing .env content
        existing_vars=$(grep -E '^[A-Z_]+=.*' .env 2>/dev/null | cut -d'=' -f1 | sort)
        
        # Read template variables
        template_vars=$(grep -E '^[A-Z_]+=.*' .env.template | cut -d'=' -f1 | sort)
        
        # Find missing variables
        missing_count=0
        
        # Create temporary file for missing vars
        temp_missing=$(mktemp)
        
        while IFS= read -r var_name; do
            if ! echo "$existing_vars" | grep -q "^${var_name}$"; then
                grep "^${var_name}=" .env.template >> "$temp_missing"
                missing_count=$((missing_count + 1))
            fi
        done <<< "$template_vars"
        
        if [ $missing_count -gt 0 ]; then
            echo "" >> .env
            echo "# Added by setup script on $(date '+%Y-%m-%d %H:%M:%S')" >> .env
            cat "$temp_missing" >> .env
            print_step "Added $missing_count missing variable(s) to .env"
        else
            print_step "All variables already exist in .env"
        fi
        
        # Clean up temp file
        rm -f "$temp_missing"
    fi

    # Clean up template file
    rm -f .env.template
}

configure_database() {
    # Check if ALL database variables already exist in .env
    # If they do, skip this entire section (NEVER override)
    if all_vars_exist_in_env "DATABASE_HOST" "DATABASE_PORT" "DATABASE_USER" "DATABASE_PASSWORD" "DATABASE_NAME" "DATABASE_SSL_MODE"; then
        print_step "Database configuration already set in .env (skipping)"
        return
    fi
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      DATABASE CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    
    echo "Choose database setup:"
    echo "  1. Use default PostgreSQL (Docker container)"
    echo "  2. Use custom database credentials"
    echo ""
    
    if [ "$AUTO_YES" = "true" ]; then
        DB_CHOICE=1
    else
        read -p "Enter choice [1]: " DB_CHOICE
        DB_CHOICE=${DB_CHOICE:-1}
    fi
    
    if [ "$DB_CHOICE" = "1" ]; then
        echo ""
        echo "Using default PostgreSQL Docker container"
        echo ""
        
        DATABASE_USER=$(prompt_env_var "DATABASE_USER" "postgres" "Database User")
        DATABASE_PASSWORD=$(prompt_env_var "DATABASE_PASSWORD" "postgres" "Database Password" "true")
        DATABASE_NAME=$(prompt_env_var "DATABASE_NAME" "serenibase" "Database Name")
        
        # DATABASE_PORT: Used for external host connections (e.g., localhost:5432)
        # For Docker-to-Docker internal connections, always use 5432
        if var_exists_in_env "DATABASE_PORT"; then
            DATABASE_PORT=$(get_env_var "DATABASE_PORT")
        else
            DATABASE_PORT="5432"
        fi
        
        # DATABASE_HOST: For external connections (localhost or IP address)
        # For Docker-to-Docker internal connections, always use 'postgres' (service name)
        if var_exists_in_env "DATABASE_HOST"; then
            DATABASE_HOST=$(get_env_var "DATABASE_HOST")
        else
            DATABASE_HOST="postgres"
        fi
        
        # DATABASE_INTERNAL_HOST and DATABASE_INTERNAL_PORT are used by applications
        # running INSIDE docker-compose network to connect to PostgreSQL container
        # These should always be: postgres:5432 (service name and internal port)
        DATABASE_INTERNAL_HOST="postgres"
        DATABASE_INTERNAL_PORT="5432"
        
        # Use disable only if DATABASE_SSL_MODE doesn't exist in .env
        if var_exists_in_env "DATABASE_SSL_MODE"; then
            DATABASE_SSL_MODE=$(get_env_var "DATABASE_SSL_MODE")
        else
            DATABASE_SSL_MODE="disable"
        fi
    else
        echo ""
        echo "Enter custom database configuration:"
        echo ""
        
        DATABASE_HOST=$(prompt_env_var "DATABASE_HOST" "" "Database Host")
        if [ -z "$DATABASE_HOST" ]; then
            print_error "Database host is required"
            exit 1
        fi
        
        DATABASE_PORT=$(prompt_env_var "DATABASE_PORT" "5432" "Database Port")
        
        DATABASE_USER=$(prompt_env_var "DATABASE_USER" "" "Database User")
        if [ -z "$DATABASE_USER" ]; then
            print_error "Database user is required"
            exit 1
        fi
        
        DATABASE_PASSWORD=$(prompt_env_var "DATABASE_PASSWORD" "" "Database Password" "true")
        if [ -z "$DATABASE_PASSWORD" ]; then
            print_error "Database password is required"
            exit 1
        fi
        
        DATABASE_NAME=$(prompt_env_var "DATABASE_NAME" "" "Database Name")
        if [ -z "$DATABASE_NAME" ]; then
            print_error "Database name is required"
            exit 1
        fi
        
        DATABASE_SSL_MODE=$(prompt_env_var "DATABASE_SSL_MODE" "disable" "SSL Mode")
    fi
    
    # Update database configuration in .env (only if changed)
    update_env_var_if_changed "DATABASE_HOST" "$DATABASE_HOST"
    update_env_var_if_changed "DATABASE_PORT" "$DATABASE_PORT"
    update_env_var_if_changed "DATABASE_USER" "$DATABASE_USER"
    update_env_var_if_changed "DATABASE_PASSWORD" "$DATABASE_PASSWORD"
    update_env_var_if_changed "DATABASE_NAME" "$DATABASE_NAME"
    update_env_var_if_changed "DATABASE_SSL_MODE" "$DATABASE_SSL_MODE"
    
    # Set internal Docker connection variables (used by services inside docker-compose)
    # These are always postgres:5432 for container-to-container communication
    if [ "$DB_CHOICE" = "1" ]; then
        update_env_var_if_changed "DATABASE_INTERNAL_HOST" "postgres"
        update_env_var_if_changed "DATABASE_INTERNAL_PORT" "5432"
    fi
    
    print_step "Database configuration updated"
}

configure_jwt_secret() {
    # Check if JWT secret already exists in .env AND is not the default placeholder
    local current_jwt_secret=$(get_env_var "AUTH_JWT_SECRET")
    if [ -n "$current_jwt_secret" ] && [ "$current_jwt_secret" != "change-this-to-a-secure-random-string-min32chars" ]; then
        print_step "JWT Secret already set in .env (skipping)"
        return
    fi
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      AUTHENTICATION CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    
    read -p "JWT Secret (min 32 chars) [press Enter to generate]: " AUTH_JWT_SECRET
    
    if [ -z "$AUTH_JWT_SECRET" ]; then
        # Generate random JWT secret
        AUTH_JWT_SECRET=$(openssl rand -base64 32 | tr -d '\n' | head -c 32)
        echo "Generated JWT Secret: $AUTH_JWT_SECRET"
    fi
    
    # Update JWT secret in .env (only if changed)
    update_env_var_if_changed "AUTH_JWT_SECRET" "$AUTH_JWT_SECRET"
    
    print_step "JWT Secret configured"
}

configure_email() {
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      EMAIL CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    echo "Enter SMTP email configuration (press Enter to keep existing values):"
    echo ""
    
    # Get current values from .env to use as defaults
    local current_smtp_host=$(get_env_var "EMAIL_SMTP_HOST")
    local current_smtp_port=$(get_env_var "EMAIL_SMTP_PORT")
    local current_smtp_username=$(get_env_var "EMAIL_SMTP_USERNAME")
    local current_smtp_password=$(get_env_var "EMAIL_SMTP_PASSWORD")
    local current_from_email=$(get_env_var "EMAIL_FROM_EMAIL")
    
    # Set defaults - use current value if exists, otherwise use reasonable defaults
    local default_smtp_host="${current_smtp_host:=smtp.gmail.com}"
    local default_smtp_port="${current_smtp_port:=587}"
    local default_smtp_username="${current_smtp_username:=}"
    local default_smtp_password="${current_smtp_password:=}"
    local default_from_email="${current_from_email:=}"
    
    # Always prompt for each field, using current value as default
    if [ "$AUTO_YES" = true ]; then
        EMAIL_SMTP_HOST="${default_smtp_host}"
        EMAIL_SMTP_PORT="${default_smtp_port}"
        EMAIL_SMTP_USERNAME="${default_smtp_username}"
        EMAIL_SMTP_PASSWORD="${default_smtp_password}"
        EMAIL_FROM_EMAIL="${default_from_email:-$EMAIL_SMTP_USERNAME}"
    else
        # Interactive prompts
        read -p "SMTP Host [${default_smtp_host}]: " EMAIL_SMTP_HOST
        EMAIL_SMTP_HOST="${EMAIL_SMTP_HOST:-$default_smtp_host}"
        
        read -p "SMTP Port [${default_smtp_port}]: " EMAIL_SMTP_PORT
        EMAIL_SMTP_PORT="${EMAIL_SMTP_PORT:-$default_smtp_port}"
        
        read -p "SMTP Username (email) [${default_smtp_username}]: " EMAIL_SMTP_USERNAME
        EMAIL_SMTP_USERNAME="${EMAIL_SMTP_USERNAME:-$default_smtp_username}"
        if [ -z "$EMAIL_SMTP_USERNAME" ]; then
            print_error "SMTP username is required"
            exit 1
        fi
        
        read -sp "SMTP Password (app password) [${default_smtp_password}]: " EMAIL_SMTP_PASSWORD
        EMAIL_SMTP_PASSWORD="${EMAIL_SMTP_PASSWORD:-$default_smtp_password}"
        echo ""
        if [ -z "$EMAIL_SMTP_PASSWORD" ]; then
            print_error "SMTP password is required"
            exit 1
        fi
        
        read -p "From Email [${default_from_email:-$EMAIL_SMTP_USERNAME}]: " EMAIL_FROM_EMAIL
        EMAIL_FROM_EMAIL="${EMAIL_FROM_EMAIL:-${default_from_email:-$EMAIL_SMTP_USERNAME}}"
    fi
    
    # Update email configuration in .env (only if changed)
    update_env_var_if_changed "EMAIL_SMTP_HOST" "$EMAIL_SMTP_HOST"
    update_env_var_if_changed "EMAIL_SMTP_PORT" "$EMAIL_SMTP_PORT"
    update_env_var_if_changed "EMAIL_SMTP_USERNAME" "$EMAIL_SMTP_USERNAME"
    update_env_var_if_changed "EMAIL_SMTP_PASSWORD" "$EMAIL_SMTP_PASSWORD"
    update_env_var_if_changed "EMAIL_FROM_EMAIL" "$EMAIL_FROM_EMAIL"
    
    print_step "Email configuration updated"
}

configure_storage() {
    # Check if storage driver is already configured in .env
    # If it is, skip this entire section (NEVER override)
    if var_exists_in_env "STORAGE_DRIVER"; then
        print_step "Storage configuration already set in .env (skipping)"
        return
    fi
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      STORAGE CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    
    echo "Choose storage driver:"
    echo "  1. Local filesystem (for development only)"
    echo "  2. MinIO (Docker container - recommended)"
    echo "  3. MinIO Custom (external MinIO server)"
    echo "  4. AWS S3"
    echo ""
    
    if [ "$AUTO_YES" = "true" ]; then
        STORAGE_CHOICE=2
    else
        read -p "Enter choice [2]: " STORAGE_CHOICE
        STORAGE_CHOICE=${STORAGE_CHOICE:-2}
    fi
    
    if [ "$STORAGE_CHOICE" = "1" ]; then
        echo ""
        echo "Using local filesystem storage"
        
        STORAGE_DEV_PATH=$(prompt_env_var "STORAGE_DEV_PATH" "./uploads" "Storage path")
        
        update_env_var_if_changed "STORAGE_DRIVER" "local"
        update_env_var_if_changed "STORAGE_DEV_PATH" "$STORAGE_DEV_PATH"
        
        print_step "Local filesystem storage configured"
        
    elif [ "$STORAGE_CHOICE" = "2" ]; then
        echo ""
        echo "Using default MinIO Docker container"
        
        STORAGE_MINIO_ACCESS_KEY=$(prompt_env_var "STORAGE_MINIO_ACCESS_KEY" "minioadmin" "MinIO Access Key")
        STORAGE_MINIO_SECRET_KEY=$(prompt_env_var "STORAGE_MINIO_SECRET_KEY" "minioadmin" "MinIO Secret Key" "true")
        STORAGE_MINIO_BUCKET=$(prompt_env_var "STORAGE_MINIO_BUCKET" "serenibase" "Bucket Name")
        
        update_env_var_if_changed "STORAGE_DRIVER" "minio"
        update_env_var_if_changed "STORAGE_MINIO_ENDPOINT" "minio:9000"
        update_env_var_if_changed "STORAGE_MINIO_ACCESS_KEY" "$STORAGE_MINIO_ACCESS_KEY"
        update_env_var_if_changed "STORAGE_MINIO_SECRET_KEY" "$STORAGE_MINIO_SECRET_KEY"
        update_env_var_if_changed "STORAGE_MINIO_BUCKET" "$STORAGE_MINIO_BUCKET"
        update_env_var_if_changed "STORAGE_MINIO_USE_SSL" "false"
        
        print_step "MinIO Docker storage configured"
        
    elif [ "$STORAGE_CHOICE" = "3" ]; then
        echo ""
        echo "Enter custom MinIO configuration:"
        echo ""
        
        STORAGE_MINIO_ENDPOINT=$(prompt_env_var "STORAGE_MINIO_ENDPOINT" "" "MinIO Endpoint (host:port)")
        if [ -z "$STORAGE_MINIO_ENDPOINT" ]; then
            print_error "MinIO endpoint is required"
            exit 1
        fi
        
        STORAGE_MINIO_ACCESS_KEY=$(prompt_env_var "STORAGE_MINIO_ACCESS_KEY" "" "MinIO Access Key")
        if [ -z "$STORAGE_MINIO_ACCESS_KEY" ]; then
            print_error "MinIO access key is required"
            exit 1
        fi
        
        STORAGE_MINIO_SECRET_KEY=$(prompt_env_var "STORAGE_MINIO_SECRET_KEY" "" "MinIO Secret Key" "true")
        if [ -z "$STORAGE_MINIO_SECRET_KEY" ]; then
            print_error "MinIO secret key is required"
            exit 1
        fi
        
        STORAGE_MINIO_BUCKET=$(prompt_env_var "STORAGE_MINIO_BUCKET" "serenibase" "Bucket Name")
        STORAGE_MINIO_USE_SSL=$(prompt_env_var "STORAGE_MINIO_USE_SSL" "false" "Use SSL (true/false)")
        
        update_env_var_if_changed "STORAGE_DRIVER" "minio"
        update_env_var_if_changed "STORAGE_MINIO_ENDPOINT" "$STORAGE_MINIO_ENDPOINT"
        update_env_var_if_changed "STORAGE_MINIO_ACCESS_KEY" "$STORAGE_MINIO_ACCESS_KEY"
        update_env_var_if_changed "STORAGE_MINIO_SECRET_KEY" "$STORAGE_MINIO_SECRET_KEY"
        update_env_var_if_changed "STORAGE_MINIO_BUCKET" "$STORAGE_MINIO_BUCKET"
        update_env_var_if_changed "STORAGE_MINIO_USE_SSL" "$STORAGE_MINIO_USE_SSL"
        
        print_step "Custom MinIO storage configured"
        
    elif [ "$STORAGE_CHOICE" = "4" ]; then
        echo ""
        echo "Enter AWS S3 configuration:"
        echo ""
        
        STORAGE_AWS_REGION=$(prompt_env_var "STORAGE_AWS_REGION" "us-east-1" "AWS Region")
        
        STORAGE_AWS_BUCKET=$(prompt_env_var "STORAGE_AWS_BUCKET" "" "S3 Bucket Name")
        if [ -z "$STORAGE_AWS_BUCKET" ]; then
            print_error "S3 bucket name is required"
            exit 1
        fi
        
        STORAGE_AWS_ACCESS_KEY=$(prompt_env_var "STORAGE_AWS_ACCESS_KEY" "" "AWS Access Key")
        if [ -z "$STORAGE_AWS_ACCESS_KEY" ]; then
            print_error "AWS access key is required"
            exit 1
        fi
        
        STORAGE_AWS_SECRET_KEY=$(prompt_env_var "STORAGE_AWS_SECRET_KEY" "" "AWS Secret Key" "true")
        if [ -z "$STORAGE_AWS_SECRET_KEY" ]; then
            print_error "AWS secret key is required"
            exit 1
        fi
        
        update_env_var_if_changed "STORAGE_DRIVER" "s3"
        update_env_var_if_changed "STORAGE_AWS_REGION" "$STORAGE_AWS_REGION"
        update_env_var_if_changed "STORAGE_AWS_BUCKET" "$STORAGE_AWS_BUCKET"
        update_env_var_if_changed "STORAGE_AWS_ACCESS_KEY" "$STORAGE_AWS_ACCESS_KEY"
        update_env_var_if_changed "STORAGE_AWS_SECRET_KEY" "$STORAGE_AWS_SECRET_KEY"
        
        print_step "AWS S3 storage configured"
        
    else
        print_error "Invalid choice"
        exit 1
    fi
}

configure_public_host() {
    # Check if PUBLIC_HOST already exists in .env
    # If it does, skip this entire section (NEVER override)
    if var_exists_in_env "PUBLIC_HOST"; then
        print_step "Network configuration already set in .env (skipping)"
        return
    fi
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      NETWORK CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""

    echo "Enter your public IP address or domain name:"
    echo "(Examples: 192.168.1.100, myapp.example.com, or localhost for local development)"
    echo ""
    
    PUBLIC_HOST=$(prompt_env_var "PUBLIC_HOST" "localhost" "IP/Domain")

    # Use localhost as default if nothing entered
    if [ -z "$PUBLIC_HOST" ]; then
        PUBLIC_HOST="localhost"
    fi

    # Update public host related variables (only if changed)
    update_env_var_if_changed "PUBLIC_HOST" "$PUBLIC_HOST"
    update_env_var_if_changed "SERVER_IP" "$PUBLIC_HOST"
    update_env_var_if_changed "STORAGE_SERVER_IP" "$PUBLIC_HOST"

    # Always ensure Base UI API URL matches public host
    ensure_baseui_api_base_url "$PUBLIC_HOST"

    # Always ensure CORS includes the public host
    ensure_cors_origin "$PUBLIC_HOST"
    
    # Always ensure reset-password URL matches public host (only if changed)
    update_env_var_if_changed "AUTH_RESET_PASSWORD_URL" "http://$PUBLIC_HOST:5050/reset-password?token=%s"
    
    print_step "Configured PUBLIC_HOST"
    print_step "Configured SERVER_IP"
    print_step "Configured STORAGE_SERVER_IP"
    print_step "Configured BASEUI_VITE_API_BASE_URL"
    print_step "Configured AUTH_RESET_PASSWORD_URL"
}

configure_owner() {
    # Check if ALL owner variables already exist in .env
    # If they do, skip this entire section (NEVER override)
    if all_vars_exist_in_env "OWNER_FIRST_NAME" "OWNER_LAST_NAME" "OWNER_EMAIL" "OWNER_PASSWORD"; then
        print_step "Owner configuration already set in .env (skipping)"
        return
    fi
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      OWNER REGISTRATION"
    echo "========================================================================${NC}"
    echo ""
    
    echo "Enter owner registration details (press Enter to keep existing values):"
    echo ""
    
    OWNER_FIRST_NAME=$(prompt_env_var "OWNER_FIRST_NAME" "Admin" "First Name")
    OWNER_LAST_NAME=$(prompt_env_var "OWNER_LAST_NAME" "User" "Last Name")
    OWNER_EMAIL=$(prompt_env_var "OWNER_EMAIL" "admin@example.com" "Email")
    OWNER_PASSWORD=$(prompt_env_var "OWNER_PASSWORD" "Admin@123" "Password" "true")

    # Update .env file with owner configuration (only if changed)
    update_env_var_if_changed "OWNER_FIRST_NAME" "$OWNER_FIRST_NAME"
    update_env_var_if_changed "OWNER_LAST_NAME" "$OWNER_LAST_NAME"
    update_env_var_if_changed "OWNER_EMAIL" "$OWNER_EMAIL"
    update_env_var_if_changed "OWNER_PASSWORD" "$OWNER_PASSWORD"

    print_step "Owner configuration updated"
}

configure_ports() {
    # Check if ALL port variables already exist in .env
    # If they do, skip this entire section (NEVER override)
    if all_vars_exist_in_env "MINIO_API_PORT" "MINIO_CONSOLE_PORT" "BASE_UI_PORT" "ANTIVIRUS_CLAMAV_PORT"; then
        print_step "Port configuration already set in .env (skipping)"
        return
    fi
    
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      PORT CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    echo "Configure container ports (press Enter to use defaults):"
    echo ""
    
    MINIO_API_PORT=$(prompt_env_var "MINIO_API_PORT" "9000" "MinIO API Port")
    MINIO_CONSOLE_PORT=$(prompt_env_var "MINIO_CONSOLE_PORT" "9001" "MinIO Console Port")
    BASE_UI_PORT=$(prompt_env_var "BASE_UI_PORT" "5050" "Base UI Port")
    ANTIVIRUS_CLAMAV_PORT=$(prompt_env_var "ANTIVIRUS_CLAMAV_PORT" "3310" "ClamAV Port")
    
    # Update port configuration in .env (only if changed)
    update_env_var_if_changed "MINIO_API_PORT" "$MINIO_API_PORT"
    update_env_var_if_changed "MINIO_CONSOLE_PORT" "$MINIO_CONSOLE_PORT"
    update_env_var_if_changed "BASE_UI_PORT" "$BASE_UI_PORT"
    update_env_var_if_changed "ANTIVIRUS_CLAMAV_PORT" "$ANTIVIRUS_CLAMAV_PORT"
    
    print_step "Port configuration updated"
}

prepare_docker_volumes() {
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                    PREPARING DOCKER VOLUMES"
    echo "========================================================================${NC}"
    echo ""
    
    # Pre-create required directories with proper permissions
    # This is especially important on macOS Docker Desktop
    local dirs_to_create=(
        "./services/storage-service/uploads"
        "./data"
    )
    
    for dir in "${dirs_to_create[@]}"; do
        if [ ! -d "$dir" ]; then
            echo "Creating directory: $dir"
            mkdir -p "$dir"
            chmod 755 "$dir"
        fi
    done
    
    print_step "Docker volumes prepared (directories created with proper permissions)"
}

clone_repositories() {
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
}

start_services() {
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
}

# Entry point
print_header
check_prerequisites
setup_environment
configure_database
configure_jwt_secret
configure_email
configure_storage
configure_public_host
configure_owner
configure_ports
clone_repositories
prepare_docker_volumes
start_services

# Read final values from .env for display (with fallbacks)
PUBLIC_HOST=$(get_env_var "PUBLIC_HOST")
[ -z "$PUBLIC_HOST" ] && PUBLIC_HOST="localhost"
OWNER_EMAIL=$(get_env_var "OWNER_EMAIL")
[ -z "$OWNER_EMAIL" ] && OWNER_EMAIL="admin@example.com"
OWNER_PASSWORD=$(get_env_var "OWNER_PASSWORD")
[ -z "$OWNER_PASSWORD" ] && OWNER_PASSWORD="Admin@123"

echo ""
echo -e "${BLUE}========================================================================"
echo "                      SETUP COMPLETE!"
echo "========================================================================${NC}"
echo ""
echo -e "${GREEN}Access your application at:${NC}"
echo "  Frontend:  http://${PUBLIC_HOST}:5050"
echo "  Backend:   http://${PUBLIC_HOST}:8080"
echo "  MinIO:     http://${PUBLIC_HOST}:9001"
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
