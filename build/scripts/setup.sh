#!/bin/bash
# ========================================================================
#                    SERENIBASE SETUP SCRIPT
#
#  Interactive setup script to configure and deploy SereniBase
# ========================================================================

# Don't use 'set -e' to allow proper Ctrl+C handling
# set -e  # Commented out to handle Ctrl+C gracefully

# Get script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

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

# Setup environment
setup_environment() {
    echo -e "\n${BLUE}Setting up environment...${NC}\n"
    
    # Create a temporary file with all default environment variables
    cat > .env.template << 'EOF'
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║                         SERENIBASE CONFIGURATION                              ║
# ║                  Generated by Interactive Setup Script                        ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🌐 NETWORK CONFIGURATION                            │
# └──────────────────────────────────────────────────────────────────────────────┘

PUBLIC_HOST=localhost

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🖥️  SERVER CONFIGURATION                            │
# └──────────────────────────────────────────────────────────────────────────────┘

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_READ_TIMEOUT=30
SERVER_WRITE_TIMEOUT=30
SERVER_ENV=dev
SERVER_SCHEME=http

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🗄️  DATABASE CONFIGURATION                          │
# └──────────────────────────────────────────────────────────────────────────────┘

DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=serenibase
DATABASE_SSL_MODE=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=1h

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🔐 AUTHENTICATION CONFIGURATION                     │
# └──────────────────────────────────────────────────────────────────────────────┘

AUTH_URL=http://jwt-provider:8081
AUTH_RESET_PASSWORD_URL=http://localhost:5050/reset-password?token=%s
AUTH_JWT_SECRET=change-this-to-a-secure-random-string-min32chars
AUTH_PORT=8081
AUTH_HOST=0.0.0.0
AUTH_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050
AUTH_ENV=development
AUTH_LOG_LEVEL=info

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           👤 ADMIN ACCOUNT                                    │
# └──────────────────────────────────────────────────────────────────────────────┘

OWNER_FIRST_NAME=Admin
OWNER_LAST_NAME=User
OWNER_EMAIL=admin@example.com
OWNER_PASSWORD=Admin@123
TEMPORARY_USER_PASSWORD=FC4i;<S8q?~0

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           📧 EMAIL CONFIGURATION                              │
# └──────────────────────────────────────────────────────────────────────────────┘

EMAIL_URL=http://email-service:8082/api/v1/email
EMAIL_HOST=0.0.0.0
EMAIL_PORT=8082
EMAIL_ALLOWED_ORIGIN=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050
EMAIL_SMTP_HOST=your_email_host
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=
EMAIL_SMTP_PASSWORD=
EMAIL_FROM_EMAIL=

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           📁 STORAGE CONFIGURATION                            │
# └──────────────────────────────────────────────────────────────────────────────┘

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

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🦠 ANTIVIRUS CONFIGURATION                          │
# └──────────────────────────────────────────────────────────────────────────────┘

ANTIVIRUS_URL=http://antivirus-service:8084
ANTIVIRUS_HOST=0.0.0.0
ANTIVIRUS_PORT=8084
ANTIVIRUS_BASE_URL=http://antivirus-service:8084
ANTIVIRUS_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050,http://serenibase:8080,http://base-ui:5050
ANTIVIRUS_DRIVER=clamav
ANTIVIRUS_CLAMAV_ADDRESS=clamav:3310
ANTIVIRUS_CLAMAV_TIMEOUT_SECONDS=30
ANTIVIRUS_MAX_UPLOAD_SIZE_MB=32

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🎨 FRONTEND CONFIGURATION                           │
# └──────────────────────────────────────────────────────────────────────────────┘

BASEUI_VITE_API_BASE_URL=http://localhost:8080

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🔒 CORS CONFIGURATION                               │
# └──────────────────────────────────────────────────────────────────────────────┘

CORS_ALLOWED_ORIGINS=http://localhost:5050,http://127.0.0.1:5050,http://base-ui:5050,http://serenibase:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
CORS_ALLOWED_HEADERS=Content-Type,Content-Length,Accept-Encoding,X-CSRF-Token,Authorization,accept,origin,Cache-Control,X-Requested-With,schema,workspace,base
CORS_ALLOW_CREDENTIALS=true

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           📝 LOGGING CONFIGURATION                            │
# └──────────────────────────────────────────────────────────────────────────────┘

LOG_LEVEL=info
LOG_FILE=app.log
LOG_MAX_SIZE=50
LOG_MAX_BACKUPS=10
LOG_MAX_AGE=30
LOG_COMPRESS=true

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           📦 ASSET CONFIGURATION                              │
# └──────────────────────────────────────────────────────────────────────────────┘

ASSET_MAX_SIZE=5242880
EOF
    
    if [ ! -f ".env" ]; then
        # If .env doesn't exist, create it from template
        cp .env.template .env
        print_step "Created .env with default environment variables"
    else
        # If .env exists, append missing variables
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
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      DATABASE CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    
    echo "Choose database setup:"
    echo "  1. Use default PostgreSQL (Docker container)"
    echo "  2. Use custom database credentials"
    echo ""
    read -p "Enter choice [1]: " DB_CHOICE
    DB_CHOICE=${DB_CHOICE:-1}
    
    if [ "$DB_CHOICE" = "1" ]; then
        echo ""
        echo "Using default PostgreSQL Docker container"
        echo ""
        
        read -p "Database User [postgres]: " DATABASE_USER
        DATABASE_USER=${DATABASE_USER:-postgres}
        
        read -p "Database Password [postgres]: " DATABASE_PASSWORD
        DATABASE_PASSWORD=${DATABASE_PASSWORD:-postgres}
        
        read -p "Database Name [serenibase]: " DATABASE_NAME
        DATABASE_NAME=${DATABASE_NAME:-serenibase}
        
        DATABASE_HOST="postgres"
        DATABASE_PORT="5432"
        DATABASE_SSL_MODE="disable"
    else
        echo ""
        echo "Enter custom database configuration:"
        echo ""
        
        read -p "Database Host: " DATABASE_HOST
        if [ -z "$DATABASE_HOST" ]; then
            print_error "Database host is required"
            exit 1
        fi
        
        read -p "Database Port [5432]: " DATABASE_PORT
        DATABASE_PORT=${DATABASE_PORT:-5432}
        
        read -p "Database User: " DATABASE_USER
        if [ -z "$DATABASE_USER" ]; then
            print_error "Database user is required"
            exit 1
        fi
        
        read -s -p "Database Password: " DATABASE_PASSWORD
        echo ""
        if [ -z "$DATABASE_PASSWORD" ]; then
            print_error "Database password is required"
            exit 1
        fi
        
        read -p "Database Name: " DATABASE_NAME
        if [ -z "$DATABASE_NAME" ]; then
            print_error "Database name is required"
            exit 1
        fi
        
        read -p "SSL Mode [disable]: " DATABASE_SSL_MODE
        DATABASE_SSL_MODE=${DATABASE_SSL_MODE:-disable}
    fi
    
    # Update database configuration in .env
    sed -i.bak "s/^DATABASE_HOST=.*/DATABASE_HOST=$DATABASE_HOST/" .env
    sed -i.bak "s/^DATABASE_PORT=.*/DATABASE_PORT=$DATABASE_PORT/" .env
    sed -i.bak "s/^DATABASE_USER=.*/DATABASE_USER=$DATABASE_USER/" .env
    sed -i.bak "s/^DATABASE_PASSWORD=.*/DATABASE_PASSWORD=$DATABASE_PASSWORD/" .env
    sed -i.bak "s/^DATABASE_NAME=.*/DATABASE_NAME=$DATABASE_NAME/" .env
    sed -i.bak "s/^DATABASE_SSL_MODE=.*/DATABASE_SSL_MODE=$DATABASE_SSL_MODE/" .env
    rm -f .env.bak
    
    print_step "Database configuration updated"
}

configure_jwt_secret() {
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
    
    # Update JWT secret in .env
    sed -i.bak "s/^AUTH_JWT_SECRET=.*/AUTH_JWT_SECRET=$AUTH_JWT_SECRET/" .env
    rm -f .env.bak
    
    print_step "JWT Secret configured"
}

configure_email() {
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      EMAIL CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""
    echo "Enter SMTP email configuration (REQUIRED):"
    echo ""
    
    read -p "SMTP Host [your_email_host]: " EMAIL_SMTP_HOST
    EMAIL_SMTP_HOST=${EMAIL_SMTP_HOST:-your_email_host}
    
    read -p "SMTP Port [587]: " EMAIL_SMTP_PORT
    EMAIL_SMTP_PORT=${EMAIL_SMTP_PORT:-587}
    
    read -p "SMTP Username (email): " EMAIL_SMTP_USERNAME
    if [ -z "$EMAIL_SMTP_USERNAME" ]; then
        print_error "SMTP username is required"
        exit 1
    fi
    
    read -s -p "SMTP Password (app password): " EMAIL_SMTP_PASSWORD
    echo ""
    if [ -z "$EMAIL_SMTP_PASSWORD" ]; then
        print_error "SMTP password is required"
        exit 1
    fi
    
    read -p "From Email [$EMAIL_SMTP_USERNAME]: " EMAIL_FROM_EMAIL
    EMAIL_FROM_EMAIL=${EMAIL_FROM_EMAIL:-$EMAIL_SMTP_USERNAME}
    
    # Update email configuration in .env
    sed -i.bak "s/^EMAIL_SMTP_HOST=.*/EMAIL_SMTP_HOST=$EMAIL_SMTP_HOST/" .env
    sed -i.bak "s/^EMAIL_SMTP_PORT=.*/EMAIL_SMTP_PORT=$EMAIL_SMTP_PORT/" .env
    sed -i.bak "s|^EMAIL_SMTP_USERNAME=.*|EMAIL_SMTP_USERNAME=$EMAIL_SMTP_USERNAME|" .env
    sed -i.bak "s|^EMAIL_SMTP_PASSWORD=.*|EMAIL_SMTP_PASSWORD=$EMAIL_SMTP_PASSWORD|" .env
    sed -i.bak "s|^EMAIL_FROM_EMAIL=.*|EMAIL_FROM_EMAIL=$EMAIL_FROM_EMAIL|" .env
    rm -f .env.bak
    
    print_step "Email configuration updated"
}

configure_storage() {
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
    read -p "Enter choice [2]: " STORAGE_CHOICE
    STORAGE_CHOICE=${STORAGE_CHOICE:-2}
    
    if [ "$STORAGE_CHOICE" = "1" ]; then
        echo ""
        echo "Using local filesystem storage"
        
        read -p "Storage path [./uploads]: " STORAGE_DEV_PATH
        STORAGE_DEV_PATH=${STORAGE_DEV_PATH:-./uploads}
        
        sed -i.bak "s|^STORAGE_DRIVER=.*|STORAGE_DRIVER=local|" .env
        sed -i.bak "s|^STORAGE_DEV_PATH=.*|STORAGE_DEV_PATH=$STORAGE_DEV_PATH|" .env
        rm -f .env.bak
        
        print_step "Local filesystem storage configured"
        
    elif [ "$STORAGE_CHOICE" = "2" ]; then
        echo ""
        echo "Using default MinIO Docker container"
        
        read -p "MinIO Access Key [minioadmin]: " STORAGE_MINIO_ACCESS_KEY
        STORAGE_MINIO_ACCESS_KEY=${STORAGE_MINIO_ACCESS_KEY:-minioadmin}
        
        read -p "MinIO Secret Key [minioadmin]: " STORAGE_MINIO_SECRET_KEY
        STORAGE_MINIO_SECRET_KEY=${STORAGE_MINIO_SECRET_KEY:-minioadmin}
        
        read -p "Bucket Name [serenibase]: " STORAGE_MINIO_BUCKET
        STORAGE_MINIO_BUCKET=${STORAGE_MINIO_BUCKET:-serenibase}
        
        sed -i.bak "s|^STORAGE_DRIVER=.*|STORAGE_DRIVER=minio|" .env
        sed -i.bak "s|^STORAGE_MINIO_ENDPOINT=.*|STORAGE_MINIO_ENDPOINT=minio:9000|" .env
        sed -i.bak "s|^STORAGE_MINIO_ACCESS_KEY=.*|STORAGE_MINIO_ACCESS_KEY=$STORAGE_MINIO_ACCESS_KEY|" .env
        sed -i.bak "s|^STORAGE_MINIO_SECRET_KEY=.*|STORAGE_MINIO_SECRET_KEY=$STORAGE_MINIO_SECRET_KEY|" .env
        sed -i.bak "s|^STORAGE_MINIO_BUCKET=.*|STORAGE_MINIO_BUCKET=$STORAGE_MINIO_BUCKET|" .env
        sed -i.bak "s|^STORAGE_MINIO_USE_SSL=.*|STORAGE_MINIO_USE_SSL=false|" .env
        rm -f .env.bak
        
        print_step "MinIO Docker storage configured"
        
    elif [ "$STORAGE_CHOICE" = "3" ]; then
        echo ""
        echo "Enter custom MinIO configuration:"
        echo ""
        
        read -p "MinIO Endpoint (host:port): " STORAGE_MINIO_ENDPOINT
        if [ -z "$STORAGE_MINIO_ENDPOINT" ]; then
            print_error "MinIO endpoint is required"
            exit 1
        fi
        
        read -p "MinIO Access Key: " STORAGE_MINIO_ACCESS_KEY
        if [ -z "$STORAGE_MINIO_ACCESS_KEY" ]; then
            print_error "MinIO access key is required"
            exit 1
        fi
        
        read -s -p "MinIO Secret Key: " STORAGE_MINIO_SECRET_KEY
        echo ""
        if [ -z "$STORAGE_MINIO_SECRET_KEY" ]; then
            print_error "MinIO secret key is required"
            exit 1
        fi
        
        read -p "Bucket Name [serenibase]: " STORAGE_MINIO_BUCKET
        STORAGE_MINIO_BUCKET=${STORAGE_MINIO_BUCKET:-serenibase}
        
        read -p "Use SSL (true/false) [false]: " STORAGE_MINIO_USE_SSL
        STORAGE_MINIO_USE_SSL=${STORAGE_MINIO_USE_SSL:-false}
        
        sed -i.bak "s|^STORAGE_DRIVER=.*|STORAGE_DRIVER=minio|" .env
        sed -i.bak "s|^STORAGE_MINIO_ENDPOINT=.*|STORAGE_MINIO_ENDPOINT=$STORAGE_MINIO_ENDPOINT|" .env
        sed -i.bak "s|^STORAGE_MINIO_ACCESS_KEY=.*|STORAGE_MINIO_ACCESS_KEY=$STORAGE_MINIO_ACCESS_KEY|" .env
        sed -i.bak "s|^STORAGE_MINIO_SECRET_KEY=.*|STORAGE_MINIO_SECRET_KEY=$STORAGE_MINIO_SECRET_KEY|" .env
        sed -i.bak "s|^STORAGE_MINIO_BUCKET=.*|STORAGE_MINIO_BUCKET=$STORAGE_MINIO_BUCKET|" .env
        sed -i.bak "s|^STORAGE_MINIO_USE_SSL=.*|STORAGE_MINIO_USE_SSL=$STORAGE_MINIO_USE_SSL|" .env
        rm -f .env.bak
        
        print_step "Custom MinIO storage configured"
        
    elif [ "$STORAGE_CHOICE" = "4" ]; then
        echo ""
        echo "Enter AWS S3 configuration:"
        echo ""
        
        read -p "AWS Region [us-east-1]: " STORAGE_AWS_REGION
        STORAGE_AWS_REGION=${STORAGE_AWS_REGION:-us-east-1}
        
        read -p "S3 Bucket Name: " STORAGE_AWS_BUCKET
        if [ -z "$STORAGE_AWS_BUCKET" ]; then
            print_error "S3 bucket name is required"
            exit 1
        fi
        
        read -p "AWS Access Key: " STORAGE_AWS_ACCESS_KEY
        if [ -z "$STORAGE_AWS_ACCESS_KEY" ]; then
            print_error "AWS access key is required"
            exit 1
        fi
        
        read -s -p "AWS Secret Key: " STORAGE_AWS_SECRET_KEY
        echo ""
        if [ -z "$STORAGE_AWS_SECRET_KEY" ]; then
            print_error "AWS secret key is required"
            exit 1
        fi
        
        sed -i.bak "s|^STORAGE_DRIVER=.*|STORAGE_DRIVER=s3|" .env
        sed -i.bak "s|^STORAGE_AWS_REGION=.*|STORAGE_AWS_REGION=$STORAGE_AWS_REGION|" .env
        sed -i.bak "s|^STORAGE_AWS_BUCKET=.*|STORAGE_AWS_BUCKET=$STORAGE_AWS_BUCKET|" .env
        sed -i.bak "s|^STORAGE_AWS_ACCESS_KEY=.*|STORAGE_AWS_ACCESS_KEY=$STORAGE_AWS_ACCESS_KEY|" .env
        sed -i.bak "s|^STORAGE_AWS_SECRET_KEY=.*|STORAGE_AWS_SECRET_KEY=$STORAGE_AWS_SECRET_KEY|" .env
        rm -f .env.bak
        
        print_step "AWS S3 storage configured"
        
    else
        print_error "Invalid choice"
        exit 1
    fi
}

configure_public_host() {
    echo ""
    echo -e "${BLUE}========================================================================"
    echo "                      NETWORK CONFIGURATION"
    echo "========================================================================${NC}"
    echo ""

    echo "Enter your public IP address or domain name:"
    echo "(Examples: 192.168.1.100, myapp.example.com, or localhost for local development)"
    echo ""
    read -p "IP/Domain [localhost]: " PUBLIC_HOST

    # Use localhost as default if nothing entered
    if [ -z "$PUBLIC_HOST" ]; then
        PUBLIC_HOST="localhost"
    fi

    # Escape special characters for sed
    ESCAPED_HOST=$(printf '%s\n' "$PUBLIC_HOST" | sed -e 's/[&/\\]/\\&/g')

    # Update .env file - only add if variable doesn't exist yet
    if [[ "$OSTYPE" == "darwin"* ]]; then
        grep -q "^PUBLIC_HOST=" .env || echo "PUBLIC_HOST=$ESCAPED_HOST" >> .env
        grep -q "^SERVER_IP=" .env || echo "SERVER_IP=$ESCAPED_HOST" >> .env
        grep -q "^STORAGE_SERVER_IP=" .env || echo "STORAGE_SERVER_IP=$ESCAPED_HOST" >> .env
    else
        grep -q "^PUBLIC_HOST=" .env || echo "PUBLIC_HOST=$ESCAPED_HOST" >> .env
        grep -q "^SERVER_IP=" .env || echo "SERVER_IP=$ESCAPED_HOST" >> .env
        grep -q "^STORAGE_SERVER_IP=" .env || echo "STORAGE_SERVER_IP=$ESCAPED_HOST" >> .env
    fi

    # Always ensure Base UI API URL matches public host
    ensure_baseui_api_base_url "$PUBLIC_HOST"

    # Always ensure CORS includes the public host
    ensure_cors_origin "$PUBLIC_HOST"
    
    # Always ensure reset-password URL matches public host
    grep -q "^AUTH_RESET_PASSWORD_URL=" .env || echo "AUTH_RESET_PASSWORD_URL=http://$PUBLIC_HOST:5050/reset-password?token=%s" >> .env
    
    print_step "Configured PUBLIC_HOST (added if missing)"
    print_step "Configured SERVER_IP (added if missing)"
    print_step "Configured BASEUI_VITE_API_BASE_URL (added if missing)"
    print_step "Configured AUTH_RESET_PASSWORD_URL (added if missing)"
}

configure_owner() {
    echo "\n${BLUE}Owner Registration Configuration${NC}\n"
    
    echo "Enter owner registration details (press Enter to use defaults):"
    echo ""
    
    read -p "First Name [Admin]: " OWNER_FIRST_NAME
    read -p "Last Name [User]: " OWNER_LAST_NAME
    read -p "Email [admin@example.com]: " OWNER_EMAIL
    read -s -p "Password [Admin@123]: " OWNER_PASSWORD
    echo ""

    OWNER_FIRST_NAME=${OWNER_FIRST_NAME:-Admin}
    OWNER_LAST_NAME=${OWNER_LAST_NAME:-User}
    OWNER_EMAIL=${OWNER_EMAIL:-admin@example.com}
    OWNER_PASSWORD=${OWNER_PASSWORD:-Admin@123}

    # Update .env file with owner configuration - only add if not already present
    grep -q "^OWNER_FIRST_NAME=" .env || echo "OWNER_FIRST_NAME=$OWNER_FIRST_NAME" >> .env
    grep -q "^OWNER_LAST_NAME=" .env || echo "OWNER_LAST_NAME=$OWNER_LAST_NAME" >> .env
    grep -q "^OWNER_EMAIL=" .env || echo "OWNER_EMAIL=$OWNER_EMAIL" >> .env
    grep -q "^OWNER_PASSWORD=" .env || echo "OWNER_PASSWORD=$OWNER_PASSWORD" >> .env

    print_step "Owner configuration set (only added if missing)"
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
clone_repositories
start_services


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
