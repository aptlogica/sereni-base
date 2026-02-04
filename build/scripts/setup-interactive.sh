#!/bin/bash
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║                     SereniBase Interactive Setup (Bash)                       ║
# ║                         Cross-Platform Configuration                          ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Detect if we have color support
if [ -t 1 ]; then
    SUPPORTS_COLOR=true
else
    SUPPORTS_COLOR=false
fi

color_print() {
    local color="$1"
    local message="$2"
    if [ "$SUPPORTS_COLOR" = true ]; then
        echo -e "${color}${message}${NC}"
    else
        echo "$message"
    fi
}

show_help() {
    color_print "$CYAN" "╔══════════════════════════════════════════════════════════════╗"
    color_print "$CYAN" "║          SereniBase Interactive Setup Script                ║"
    color_print "$CYAN" "╚══════════════════════════════════════════════════════════════╝"
    echo ""
    echo "USAGE:"
    echo "  ./setup-interactive.sh [OPTIONS]"
    echo ""
    echo "OPTIONS:"
    echo "  --skip-docker    Skip Docker availability check"
    echo "  --help           Show this help message"
    echo ""
    echo "DESCRIPTION:"
    echo "  This script will guide you through configuring SereniBase."
    echo "  It will prompt for required values and generate a .env file."
    echo ""
    echo "EXAMPLES:"
    echo "  ./setup-interactive.sh"
    echo "  ./setup-interactive.sh --skip-docker"
    echo ""
    exit 0
}

# Parse arguments
SKIP_DOCKER=false
while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-docker)
            SKIP_DOCKER=true
            shift
            ;;
        --help)
            show_help
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            ;;
    esac
done

read_input() {
    local prompt="$1"
    local default="$2"
    local required="$3"
    local is_password="$4"
    local value=""
    
    if [ -n "$default" ]; then
        display_prompt="$prompt [$default]: "
    else
        display_prompt="$prompt: "
    fi
    
    if [ "$is_password" = "true" ]; then
        read -s -p "$display_prompt" value
        echo ""
    else
        read -p "$display_prompt" value
    fi
    
    # Use default if empty
    if [ -z "$value" ]; then
        if [ "$required" = "true" ] && [ -z "$default" ]; then
            color_print "$YELLOW" "⚠️  This value is required!"
            read_input "$prompt" "$default" "$required" "$is_password"
            return
        fi
        value="$default"
    fi
    
    echo "$value"
}

validate_email() {
    local email="$1"
    if [[ "$email" =~ ^[^@]+@[^@]+\.[^@]+$ ]]; then
        return 0
    else
        return 1
    fi
}

get_local_ip() {
    local ip=""
    
    # Try different methods based on OS
    if command -v ip &> /dev/null; then
        # Linux with ip command
        ip=$(ip route get 1 2>/dev/null | awk '{print $7; exit}')
    elif command -v ifconfig &> /dev/null; then
        # macOS or Linux with ifconfig
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            ip=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | head -1)
        else
            # Linux
            ip=$(ifconfig | grep "inet " | grep -v 127.0.0.1 | awk '{print $2}' | cut -d':' -f2 | head -1)
        fi
    fi
    
    # Fallback to localhost
    if [ -z "$ip" ]; then
        ip="localhost"
    fi
    
    echo "$ip"
}

# Clear screen and show banner
clear
color_print "$CYAN" "╔══════════════════════════════════════════════════════════════════════════════╗"
color_print "$CYAN" "║                         🚀 SERENIBASE SETUP                                   ║"
color_print "$CYAN" "║                     Interactive Configuration Wizard                          ║"
color_print "$CYAN" "╚══════════════════════════════════════════════════════════════════════════════╝"
echo ""

# Detect system information
LOCAL_IP=$(get_local_ip)
color_print "$BLUE" "📡 Detected System Information:"
echo "   OS: $(uname -s) $(uname -r)"
echo "   Local IP: $LOCAL_IP"
echo ""

# Check for Docker (unless skipped)
if [ "$SKIP_DOCKER" = false ]; then
    color_print "$BLUE" "🐳 Checking Docker..."
    if command -v docker &> /dev/null; then
        DOCKER_VERSION=$(docker --version)
        color_print "$GREEN" "   ✓ Docker is available: $DOCKER_VERSION"
    else
        color_print "$YELLOW" "   ✗ Docker is not available"
        color_print "$YELLOW" "   Note: Docker is required to run SereniBase with all services"
    fi
    echo ""
fi

# Start configuration
color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
color_print "$CYAN" "                    📋 CONFIGURATION                            "
color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
echo ""
color_print "$YELLOW" "Press Enter to accept default values shown in [brackets]"
echo ""

# Network Configuration
color_print "$BLUE" "┌─────────────────────────────────────────────────────────┐"
color_print "$BLUE" "│           🌐 NETWORK CONFIGURATION                      │"
color_print "$BLUE" "└─────────────────────────────────────────────────────────┘"
echo ""
echo "This is how users will access your application."
echo "Examples:"
echo "  - localhost (for testing on this machine)"
echo "  - $LOCAL_IP (for LAN access)"
echo "  - yourdomain.com (for production)"
echo ""

PUBLIC_HOST=$(read_input "IP Address or Domain" "localhost" "true" "false")
echo ""

# Admin Account Configuration
color_print "$BLUE" "┌─────────────────────────────────────────────────────────┐"
color_print "$BLUE" "│           👤 ADMIN ACCOUNT SETUP                        │"
color_print "$BLUE" "└─────────────────────────────────────────────────────────┘"
echo ""
echo "Create the first administrator account."
echo ""

OWNER_FIRST_NAME=$(read_input "Admin First Name" "Admin" "false" "false")
OWNER_LAST_NAME=$(read_input "Admin Last Name" "User" "false" "false")

# Email validation loop
while true; do
    OWNER_EMAIL=$(read_input "Admin Email" "admin@example.com" "true" "false")
    if validate_email "$OWNER_EMAIL"; then
        break
    else
        color_print "$YELLOW" "⚠️  Please enter a valid email address"
    fi
done

# Password with confirmation
while true; do
    OWNER_PASSWORD=$(read_input "Admin Password" "Admin@123" "true" "false")
    CONFIRM_PASSWORD=$(read_input "Confirm Password" "" "true" "false")
    if [ "$OWNER_PASSWORD" = "$CONFIRM_PASSWORD" ]; then
        break
    else
        color_print "$YELLOW" "⚠️  Passwords do not match! Please try again."
    fi
done
echo ""

# Security Configuration
color_print "$BLUE" "┌─────────────────────────────────────────────────────────┐"
color_print "$BLUE" "│           🔐 SECURITY CONFIGURATION                     │"
color_print "$BLUE" "└─────────────────────────────────────────────────────────┘"
echo ""
echo "JWT secret is used to sign authentication tokens."
echo "⚠️  Use a strong random string (32+ characters) for production!"
echo ""

AUTH_JWT_SECRET=$(read_input "JWT Secret Key" "change-this-to-a-secure-random-string-min32chars" "true" "false")
echo ""

# Database Configuration
color_print "$BLUE" "┌─────────────────────────────────────────────────────────┐"
color_print "$BLUE" "│           🗄️  DATABASE CONFIGURATION                    │"
color_print "$BLUE" "└─────────────────────────────────────────────────────────┘"
echo ""
echo "For Docker deployment, use default values."
echo "For external database, specify custom host and credentials."
echo ""

USE_DOCKER_DB=$(read_input "Use Docker PostgreSQL? (y/n)" "y" "false" "false")
if [[ "$USE_DOCKER_DB" =~ ^[Yy]$ ]] || [ -z "$USE_DOCKER_DB" ]; then
    DATABASE_HOST="postgres"
    DATABASE_USER="postgres"
    DATABASE_PASSWORD=$(read_input "Database Password" "postgres" "false" "false")
    DATABASE_NAME="serenibase"
    color_print "$GREEN" "   Using Docker database configuration"
else
    DATABASE_HOST=$(read_input "Database Host" "localhost" "true" "false")
    DATABASE_USER=$(read_input "Database User" "postgres" "true" "false")
    DATABASE_PASSWORD=$(read_input "Database Password" "postgres" "true" "true")
    DATABASE_NAME=$(read_input "Database Name" "serenibase" "true" "false")
fi
echo ""

# Email Configuration (Optional)
color_print "$BLUE" "┌─────────────────────────────────────────────────────────┐"
color_print "$BLUE" "│           📧 EMAIL CONFIGURATION (Optional)             │"
color_print "$BLUE" "└─────────────────────────────────────────────────────────┘"
echo ""
echo "Email is required for:"
echo "  - Password reset functionality"
echo "  - User notifications"
echo ""
echo "You can skip this and configure later."
echo ""

CONFIGURE_EMAIL=$(read_input "Configure email now? (y/n)" "n" "false" "false")
if [[ "$CONFIGURE_EMAIL" =~ ^[Yy]$ ]]; then
    echo ""
    echo "Common SMTP configurations:"
    echo "  Gmail:   smtp.gmail.com:587"
    echo "  Outlook: smtp-mail.outlook.com:587"
    echo ""
    
    EMAIL_SMTP_HOST=$(read_input "SMTP Host" "smtp.gmail.com" "false" "false")
    EMAIL_SMTP_PORT=$(read_input "SMTP Port" "587" "false" "false")
    EMAIL_SMTP_USERNAME=$(read_input "SMTP Username" "" "true" "false")
    EMAIL_SMTP_PASSWORD=$(read_input "SMTP Password" "" "true" "true")
    EMAIL_FROM_EMAIL=$(read_input "From Email" "$EMAIL_SMTP_USERNAME" "false" "false")
else
    EMAIL_SMTP_HOST="smtp.gmail.com"
    EMAIL_SMTP_PORT="587"
    EMAIL_SMTP_USERNAME="your_email@gmail.com"
    EMAIL_SMTP_PASSWORD="your_app_password"
    EMAIL_FROM_EMAIL="your_email@gmail.com"
fi
echo ""

# Storage Configuration
color_print "$BLUE" "┌─────────────────────────────────────────────────────────┐"
color_print "$BLUE" "│           📁 STORAGE CONFIGURATION                      │"
color_print "$BLUE" "└─────────────────────────────────────────────────────────┘"
echo ""
echo "Storage options:"
echo "  1. local  - Store files on disk (simple, default)"
echo "  2. minio  - Use MinIO S3-compatible storage (Docker)"
echo "  3. aws    - Use AWS S3 (production)"
echo ""

STORAGE_DRIVER=$(read_input "Storage driver (local/minio/aws)" "local" "false" "false")
echo ""

# Generate .env file
color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
color_print "$CYAN" "                 💾 GENERATING CONFIGURATION                    "
color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
echo ""

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"

cat > "$ENV_FILE" << EOF
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║                         SERENIBASE CONFIGURATION                              ║
# ║                  Generated by Interactive Setup Script                        ║
# ║                     $(date +"%Y-%m-%d %H:%M:%S")                                       ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🌐 NETWORK CONFIGURATION                            │
# └──────────────────────────────────────────────────────────────────────────────┘

PUBLIC_HOST=$PUBLIC_HOST

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

DATABASE_HOST=$DATABASE_HOST
DATABASE_PORT=5432
DATABASE_USER=$DATABASE_USER
DATABASE_PASSWORD=$DATABASE_PASSWORD
DATABASE_NAME=$DATABASE_NAME
DATABASE_SSL_MODE=disable
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=1h

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🔐 AUTHENTICATION CONFIGURATION                     │
# └──────────────────────────────────────────────────────────────────────────────┘

AUTH_URL=http://jwt-provider:8081
AUTH_RESET_PASSWORD_URL=http://${PUBLIC_HOST}:5050/reset-password?token=%s
AUTH_JWT_SECRET=$AUTH_JWT_SECRET
AUTH_PORT=8081
AUTH_HOST=0.0.0.0
AUTH_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,http://serenibase:8080,http://base-ui:5050
AUTH_ENV=development
AUTH_LOG_LEVEL=info

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           👤 ADMIN ACCOUNT                                    │
# └──────────────────────────────────────────────────────────────────────────────┘

OWNER_FIRST_NAME=$OWNER_FIRST_NAME
OWNER_LAST_NAME=$OWNER_LAST_NAME
OWNER_EMAIL=$OWNER_EMAIL
OWNER_PASSWORD=$OWNER_PASSWORD
TEMPORARY_USER_PASSWORD=FC4i;<S8q?~0

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           📧 EMAIL CONFIGURATION                              │
# └──────────────────────────────────────────────────────────────────────────────┘

EMAIL_URL=http://email-service:8082/api/v1/email
EMAIL_HOST=0.0.0.0
EMAIL_PORT=8082
EMAIL_ALLOWED_ORIGIN=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,http://serenibase:8080,http://base-ui:5050
EMAIL_SMTP_HOST=$EMAIL_SMTP_HOST
EMAIL_SMTP_PORT=$EMAIL_SMTP_PORT
EMAIL_SMTP_USERNAME=$EMAIL_SMTP_USERNAME
EMAIL_SMTP_PASSWORD=$EMAIL_SMTP_PASSWORD
EMAIL_FROM_EMAIL=$EMAIL_FROM_EMAIL

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           📁 STORAGE CONFIGURATION                            │
# └──────────────────────────────────────────────────────────────────────────────┘

STORAGE_URL=http://sereni-storage-provider:8083/api/v1
STORAGE_SERVER_PORT=8083
STORAGE_SERVER_HOST=0.0.0.0
STORAGE_SERVER_SCHEME=http
STORAGE_DRIVER=$STORAGE_DRIVER
STORAGE_DEV_PATH=./uploads
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=my-bucket
STORAGE_AWS_ACCESS_KEY=your-access-key
STORAGE_AWS_SECRET_KEY=your-secret-key
STORAGE_MINIO_ENDPOINT=minio:9000
STORAGE_MINIO_ACCESS_KEY=minioadmin
STORAGE_MINIO_SECRET_KEY=minioadmin
STORAGE_MINIO_BUCKET=my-bucket
STORAGE_MINIO_USE_SSL=false
STORAGE_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,http://serenibase:8080,http://base-ui:5050

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🦠 ANTIVIRUS CONFIGURATION                          │
# └──────────────────────────────────────────────────────────────────────────────┘

ANTIVIRUS_URL=http://antivirus-service:8084
ANTIVIRUS_HOST=0.0.0.0
ANTIVIRUS_PORT=8084
ANTIVIRUS_BASE_URL=http://antivirus-service:8084
ANTIVIRUS_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,http://serenibase:8080,http://base-ui:5050
ANTIVIRUS_DRIVER=clamav
ANTIVIRUS_CLAMAV_ADDRESS=clamav:3310
ANTIVIRUS_CLAMAV_TIMEOUT_SECONDS=30
ANTIVIRUS_MAX_UPLOAD_SIZE_MB=32

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🎨 FRONTEND CONFIGURATION                           │
# └──────────────────────────────────────────────────────────────────────────────┘

BASEUI_VITE_API_BASE_URL=http://${PUBLIC_HOST}:8080

# ┌──────────────────────────────────────────────────────────────────────────────┐
# │                           🔒 CORS CONFIGURATION                               │
# └──────────────────────────────────────────────────────────────────────────────┘

CORS_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:5050,http://localhost:5050,http://127.0.0.1:5050,http://base-ui:5050,http://serenibase:8080
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

color_print "$GREEN" "✓ Configuration file created: .env"
echo ""

# Show summary
color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
color_print "$CYAN" "                    ✅ SETUP COMPLETE!                          "
color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
echo ""

color_print "$BLUE" "📝 Configuration Summary:"
echo "   • Access URL: http://$PUBLIC_HOST:8080"
echo "   • Admin Email: $OWNER_EMAIL"
echo "   • Database Host: $DATABASE_HOST"
echo "   • Storage Driver: $STORAGE_DRIVER"
echo "   • Config File: .env"
echo ""

color_print "$BLUE" "🚀 Next Steps:"
echo ""
echo "1. Start the application:"
color_print "$YELLOW" "   docker-compose up -d"
echo ""
echo "2. Access the application:"
color_print "$YELLOW" "   http://$PUBLIC_HOST:8080"
echo ""
echo "3. Login with your admin credentials:"
echo "   Email: $OWNER_EMAIL"
echo "   Password: [the password you entered]"
echo ""

if [[ ! "$CONFIGURE_EMAIL" =~ ^[Yy]$ ]]; then
    color_print "$YELLOW" "⚠️  Note: Email is not configured. Password reset will not work."
    echo "   To configure later, edit .env and set EMAIL_SMTP_* variables"
    echo ""
fi

color_print "$BLUE" "📚 Documentation:"
echo "   • Environment Variables: docs/ENVIRONMENT_VARIABLES.md"
echo "   • API Response Codes: docs/API_RESPONSE_CODES.md"
echo "   • Setup Guide: README.md"
echo ""

color_print "$CYAN" "═══════════════════════════════════════════════════════════════"
echo ""
