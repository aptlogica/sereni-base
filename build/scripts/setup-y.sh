#!/bin/bash
# ========================================================================
#                    SERENIBASE SETUP SCRIPT (NO PROMPTS)
#
#  Full automated setup with default values - same as interactive setup
#  but without prompting the user
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
DATABASE_HOST_PARAM=postgres
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
EMAIL_SMTP_HOST=mailhog
EMAIL_SMTP_PORT=1025
EMAIL_SMTP_USERNAME=
EMAIL_SMTP_PASSWORD=
EMAIL_FROM_EMAIL=test@example.com

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

echo ""
echo -e "${BLUE}========================================================================"
echo "                      DATABASE CONFIGURATION"
echo "========================================================================${NC}"
echo ""

echo "Using default PostgreSQL Docker container"

# Check if database credentials are default or empty, update them
if grep -q "^DATABASE_USER=" .env; then
    current_user=$(grep "^DATABASE_USER=" .env | cut -d'=' -f2)
    if [ -z "$current_user" ] || [ "$current_user" = "postgres" ]; then
        sed -i.bak "s/^DATABASE_USER=.*/DATABASE_USER=postgres/" .env
    fi
else
    sed -i.bak "s/^DATABASE_USER=.*/DATABASE_USER=postgres/" .env
fi

if grep -q "^DATABASE_PASSWORD=" .env; then
    current_pass=$(grep "^DATABASE_PASSWORD=" .env | cut -d'=' -f2)
    if [ -z "$current_pass" ] || [ "$current_pass" = "postgres" ]; then
        sed -i.bak "s/^DATABASE_PASSWORD=.*/DATABASE_PASSWORD=postgres/" .env
    fi
else
    sed -i.bak "s/^DATABASE_PASSWORD=.*/DATABASE_PASSWORD=postgres/" .env
fi

if grep -q "^DATABASE_NAME=" .env; then
    current_name=$(grep "^DATABASE_NAME=" .env | cut -d'=' -f2)
    if [ -z "$current_name" ] || [ "$current_name" = "serenibase" ]; then
        sed -i.bak "s/^DATABASE_NAME=.*/DATABASE_NAME=serenibase/" .env
    fi
else
    sed -i.bak "s/^DATABASE_NAME=.*/DATABASE_NAME=serenibase/" .env
fi

rm -f .env.bak
print_step "Database configuration set to defaults"

echo ""
echo -e "${BLUE}========================================================================"
echo "                      AUTHENTICATION CONFIGURATION"
echo "========================================================================${NC}"
echo ""

# Check if JWT secret exists and is default, generate new one if needed
if grep -q "^AUTH_JWT_SECRET=" .env; then
    current_secret=$(grep "^AUTH_JWT_SECRET=" .env | cut -d'=' -f2)
    if [[ "$current_secret" =~ "change-this" ]] || [ -z "$current_secret" ]; then
        new_secret=$(openssl rand -base64 32 | tr -d '\n' | head -c 32)
        sed -i.bak "s/^AUTH_JWT_SECRET=.*/AUTH_JWT_SECRET=$new_secret/" .env
        rm -f .env.bak
        echo "[OK] Generated new JWT Secret: $new_secret"
    else
        echo "[OK] Using existing JWT Secret"
    fi
else
    new_secret=$(openssl rand -base64 32 | tr -d '\n' | head -c 32)
    sed -i.bak "s/^AUTH_JWT_SECRET=.*/AUTH_JWT_SECRET=$new_secret/" .env
    rm -f .env.bak
    echo "[OK] Generated new JWT Secret: $new_secret"
fi

echo ""
echo -e "${BLUE}========================================================================"
echo "                      EMAIL CONFIGURATION"
echo "========================================================================${NC}"
echo ""

# Check if email is configured
if grep -q "^EMAIL_SMTP_USERNAME=" .env; then
    current_email=$(grep "^EMAIL_SMTP_USERNAME=" .env | cut -d'=' -f2)
    if [[ "$current_email" =~ "@" ]] && [ "$current_email" != "your_email@gmail.com" ]; then
        echo "[OK] Using existing email configuration"
    else
        print_warning "Email not configured. Please update .env with your email credentials."
    fi
else
    print_warning "Email not configured. Please update .env with your email credentials."
fi

echo ""
echo -e "${BLUE}========================================================================"
echo "                      STORAGE CONFIGURATION"
echo "========================================================================${NC}"
echo ""

echo "Using default MinIO Docker container"

# Set default MinIO storage if not already configured
if grep -q "^STORAGE_DRIVER=" .env; then
    current_driver=$(grep "^STORAGE_DRIVER=" .env | cut -d'=' -f2)
    if [ -z "$current_driver" ] || [ "$current_driver" = "minio" ]; then
        sed -i.bak "s|^STORAGE_DRIVER=.*|STORAGE_DRIVER=minio|" .env
    fi
else
    sed -i.bak "s|^STORAGE_DRIVER=.*|STORAGE_DRIVER=minio|" .env
fi

sed -i.bak "s|^STORAGE_MINIO_ENDPOINT=.*|STORAGE_MINIO_ENDPOINT=minio:9000|" .env
sed -i.bak "s|^STORAGE_MINIO_ACCESS_KEY=.*|STORAGE_MINIO_ACCESS_KEY=minioadmin|" .env
sed -i.bak "s|^STORAGE_MINIO_SECRET_KEY=.*|STORAGE_MINIO_SECRET_KEY=minioadmin|" .env
sed -i.bak "s|^STORAGE_MINIO_BUCKET=.*|STORAGE_MINIO_BUCKET=serenibase|" .env
sed -i.bak "s|^STORAGE_MINIO_USE_SSL=.*|STORAGE_MINIO_USE_SSL=false|" .env

rm -f .env.bak
print_step "Storage configuration set to defaults"

echo ""
echo -e "${BLUE}========================================================================"
echo "                      NETWORK CONFIGURATION"
echo "========================================================================${NC}"
echo ""

PUBLIC_HOST="localhost"
echo "Using default IP/domain: $PUBLIC_HOST"
echo ""

# Update .env file - only add if variables don't exist yet
if [[ "$OSTYPE" == "darwin"* ]]; then
    grep -q "^PUBLIC_HOST=" .env || echo "PUBLIC_HOST=$PUBLIC_HOST" >> .env
    grep -q "^SERVER_IP=" .env || echo "SERVER_IP=$PUBLIC_HOST" >> .env
    grep -q "^STORAGE_SERVER_IP=" .env || echo "STORAGE_SERVER_IP=$PUBLIC_HOST" >> .env
else
    grep -q "^PUBLIC_HOST=" .env || echo "PUBLIC_HOST=$PUBLIC_HOST" >> .env
    grep -q "^SERVER_IP=" .env || echo "SERVER_IP=$PUBLIC_HOST" >> .env
    grep -q "^STORAGE_SERVER_IP=" .env || echo "STORAGE_SERVER_IP=$PUBLIC_HOST" >> .env
fi
print_step "Configured PUBLIC_HOST (added if missing)"
print_step "Configured SERVER_IP (added if missing)"
print_step "Configured BASEUI_VITE_API_BASE_URL (added if missing)"

# Always ensure Base UI API URL matches public host
ensure_baseui_api_base_url "$PUBLIC_HOST"

# Always ensure CORS includes the public host
ensure_cors_origin "$PUBLIC_HOST"

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
# Update .env file with owner configuration - only add if not already present
grep -q "^OWNER_FIRST_NAME=" .env || echo "OWNER_FIRST_NAME=$OWNER_FIRST_NAME" >> .env
grep -q "^OWNER_LAST_NAME=" .env || echo "OWNER_LAST_NAME=$OWNER_LAST_NAME" >> .env
grep -q "^OWNER_EMAIL=" .env || echo "OWNER_EMAIL=$OWNER_EMAIL" >> .env
grep -q "^OWNER_PASSWORD=" .env || echo "OWNER_PASSWORD=$OWNER_PASSWORD" >> .env

print_step "Owner configuration set (only added if missing)"

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
