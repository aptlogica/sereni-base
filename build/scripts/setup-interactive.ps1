#!/usr/bin/env pwsh
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║                   SereniBase Interactive Setup (PowerShell)                   ║
# ║                         Cross-Platform Configuration                          ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

param(
    [switch]$SkipDocker,
    [switch]$Help
)

# Handle Ctrl+C gracefully - Exit immediately without prompts
$null = [Console]::TreatControlCAsInput = $false

# Set error action to stop on errors
$ErrorActionPreference = "Stop"

# Global flag to track cancellation
$script:SetupCancelled = $false

# Trap handler for Ctrl+C and other interruptions
trap {
    $script:SetupCancelled = $true
    Write-Host ""
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host "  Setup cancelled by user." -ForegroundColor Yellow
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host ""
    exit 130  # Standard exit code for Ctrl+C
}

# Color support
$script:SupportsColor = $Host.UI.SupportsVirtualTerminal

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    if ($script:SupportsColor) {
        $colorCodes = @{
            "Red" = "`e[31m"
            "Green" = "`e[32m"
            "Yellow" = "`e[33m"
            "Blue" = "`e[34m"
            "Cyan" = "`e[36m"
            "White" = "`e[37m"
            "Reset" = "`e[0m"
        }
        Write-Host "$($colorCodes[$Color])$Message$($colorCodes['Reset'])"
    } else {
        Write-Host $Message
    }
}

function Show-Help {
    Write-ColorOutput "╔══════════════════════════════════════════════════════════════╗" "Cyan"
    Write-ColorOutput "║          SereniBase Interactive Setup Script                ║" "Cyan"
    Write-ColorOutput "╚══════════════════════════════════════════════════════════════╝" "Cyan"
    Write-Host ""
    Write-Host "USAGE:"
    Write-Host "  .\setup-interactive.ps1 [OPTIONS]"
    Write-Host ""
    Write-Host "OPTIONS:"
    Write-Host "  -SkipDocker    Skip Docker availability check"
    Write-Host "  -Help          Show this help message"
    Write-Host ""
    Write-Host "DESCRIPTION:"
    Write-Host "  This script will guide you through configuring SereniBase."
    Write-Host "  It will prompt for required values and generate a .env file."
    Write-Host ""
    Write-Host "EXAMPLES:"
    Write-Host "  .\setup-interactive.ps1"
    Write-Host "  .\setup-interactive.ps1 -SkipDocker"
    Write-Host ""
    exit 0
}

if ($Help) {
    Show-Help
}

function Read-UserInput {
    param(
        [string]$Prompt,
        [string]$Default = "",
        [bool]$Required = $false,
        [bool]$IsPassword = $false,
        [bool]$ShowDefault = $false
    )
    
    # Check if setup was cancelled
    if ($script:SetupCancelled) {
        exit 130
    }
    
    try {
        if ($IsPassword) {
            if ($Default) {
                $displayPrompt = "$Prompt [$Default]" + ": "
            } else {
                $displayPrompt = "$Prompt" + ": "
            }
            Write-Host $displayPrompt -NoNewline
            
            try {
                $secureInput = Read-Host -AsSecureString
            }
            catch {
                # Ctrl+C pressed during password input
                throw
            }
            
            # Check if cancelled after Read-Host
            if ($script:SetupCancelled) {
                exit 130
            }
            
            $BSTR = [System.Runtime.InteropServices.Marshal]::SecureStringToBSTR($secureInput)
            $value = [System.Runtime.InteropServices.Marshal]::PtrToStringAuto($BSTR)
        } elseif ($ShowDefault -and $Default) {
            # Show default value that can be edited
            Write-Host "$Prompt [Press Enter for '$Default' or type to change]: " -NoNewline
            
            try {
                $value = Read-Host
            }
            catch {
                # Ctrl+C pressed during input
                throw
            }
            
            # Check if cancelled after Read-Host
            if ($script:SetupCancelled) {
                exit 130
            }
        } else {
            if ($Default) {
                $displayPrompt = "$Prompt [$Default]" + ": "
            } else {
                $displayPrompt = "$Prompt" + ": "
            }
            
            try {
                $value = Read-Host $displayPrompt
            }
            catch {
                # Ctrl+C pressed during input
                throw
            }
            
            # Check if cancelled after Read-Host
            if ($script:SetupCancelled) {
                exit 130
            }
        }
        
        if ([string]::IsNullOrWhiteSpace($value)) {
            if ($Required -and [string]::IsNullOrWhiteSpace($Default)) {
                Write-ColorOutput "⚠️  This value is required!" "Yellow"
                return Read-UserInput -Prompt $Prompt -Default $Default -Required $Required -IsPassword $IsPassword -ShowDefault $ShowDefault
            }
            return $Default
        }
        
        return $value.Trim()
    }
    catch {
        # Handle Ctrl+C or any interruption
        $script:SetupCancelled = $true
        Write-Host ""
        Write-Host ""
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host "  Setup cancelled by user." -ForegroundColor Yellow
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host ""
        exit 130  # Standard exit code for Ctrl+C
    }
}

function Test-Email {
    param([string]$Email)
    return $Email -match '^[^@]+@[^@]+\.[^@]+$'
}

function Get-LocalIP {
    try {
        $ip = (Get-NetIPAddress -AddressFamily IPv4 | Where-Object {
            $_.InterfaceAlias -notlike "*Loopback*" -and 
            $_.IPAddress -notlike "169.254.*"
        } | Select-Object -First 1).IPAddress
        return $ip
    } catch {
        return "localhost"
    }
}

# Clear screen and show banner
Clear-Host
Write-ColorOutput "╔══════════════════════════════════════════════════════════════════════════════╗" "Cyan"
Write-ColorOutput "║                         🚀 SERENIBASE SETUP                                   ║" "Cyan"
Write-ColorOutput "║                     Interactive Configuration Wizard                          ║" "Cyan"
Write-ColorOutput "╚══════════════════════════════════════════════════════════════════════════════╝" "Cyan"
Write-Host ""

# Detect system information
$localIP = Get-LocalIP
Write-ColorOutput "📡 Detected System Information:" "Blue"
Write-Host "   OS: $($PSVersionTable.OS)"
Write-Host "   Local IP: $localIP"
Write-Host ""

# Check for Docker (unless skipped)
if (-not $SkipDocker) {
    Write-ColorOutput "🐳 Checking Docker..." "Blue"
    $dockerAvailable = $false
    try {
        $dockerVersion = docker --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            $dockerAvailable = $true
            Write-ColorOutput "   ✓ Docker is available: $dockerVersion" "Green"
        }
    } catch {
        Write-ColorOutput "   ✗ Docker is not available" "Yellow"
        Write-ColorOutput "   Note: Docker is required to run SereniBase with all services" "Yellow"
    }
    Write-Host ""
}

# Start configuration
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput "                    📋 CONFIGURATION                            " "Cyan"
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-Host ""
Write-ColorOutput "Press Enter to accept default values shown in [brackets]" "Yellow"
Write-ColorOutput "Press Ctrl+C at any time to cancel setup" "Yellow"
Write-Host ""

# Check for cancellation before starting
if ($script:SetupCancelled) {
    exit 130
}

# Network Configuration
Write-ColorOutput "========================================================================" "Cyan"
Write-ColorOutput "                     NETWORK CONFIGURATION                              " "Cyan"
Write-ColorOutput "========================================================================" "Cyan"
Write-Host ""
Write-ColorOutput "💡 Examples: localhost (local dev), $localIP (LAN access), yourdomain.com (production)" "Cyan"
Write-Host ""

$PUBLIC_HOST = Read-UserInput -Prompt "Custom IP/domain (for LAN or production access" -Default "localhost" -Required $true
Write-Host ""

# Admin Account Configuration
Write-ColorOutput "========================================================================" "Cyan"
Write-ColorOutput "                  OWNER REGISTRATION CONFIGURATION                      " "Cyan"
Write-ColorOutput "========================================================================" "Cyan"
Write-Host ""
Write-Host "Enter owner registration details (press Enter to use defaults):"
Write-Host ""

$OWNER_FIRST_NAME = Read-UserInput -Prompt "First Name" -Default "Admin"
$OWNER_LAST_NAME = Read-UserInput -Prompt "Last Name" -Default "User"

# Email validation loop
do {
    if ($script:SetupCancelled) { exit 130 }
    $OWNER_EMAIL = Read-UserInput -Prompt "Email" -Default "admin@example.com" -Required $true
    if (-not (Test-Email $OWNER_EMAIL)) {
        Write-ColorOutput "⚠️  Please enter a valid email address" "Yellow"
    }
} while (-not (Test-Email $OWNER_EMAIL))

# Password with confirmation
do {
    if ($script:SetupCancelled) { exit 130 }
    $OWNER_PASSWORD = Read-UserInput -Prompt "Password" -Default "Admin@123" -IsPassword $false
    $confirmPassword = Read-UserInput -Prompt "Confirm Password" -IsPassword $false
    if ($OWNER_PASSWORD -ne $confirmPassword) {
        Write-ColorOutput "⚠️  Passwords do not match! Please try again." "Yellow"
    }
} while ($OWNER_PASSWORD -ne $confirmPassword)
Write-Host ""

# Security Configuration
Write-ColorOutput "========================================================================" "Cyan"
Write-ColorOutput "                    SECURITY CONFIGURATION                              " "Cyan"
Write-ColorOutput "========================================================================" "Cyan"
Write-Host ""
Write-Host "JWT secret is used to sign authentication tokens."
Write-Host "⚠️  Use a strong random string (at least 32 characters) for production!"
Write-Host ""

$AUTH_JWT_SECRET = Read-UserInput -Prompt "JWT Secret Key" -Default "change-this-to-a-secure-random-string-min32chars" -Required $true
Write-Host ""

# Database Configuration
# Database Configuration
Write-ColorOutput "========================================================================" "Cyan"
Write-ColorOutput "                    DATABASE CONFIGURATION                              " "Cyan"
Write-ColorOutput "========================================================================" "Cyan"
Write-Host ""
Write-Host "For Docker deployment, use default values."
Write-Host "For external database, specify custom host and credentials."
Write-Host ""

$useDockerDB = Read-UserInput -Prompt "Use Docker PostgreSQL? (y/n)" -Default "y"
if ($useDockerDB -eq "y" -or $useDockerDB -eq "Y" -or $useDockerDB -eq "") {
    $DATABASE_HOST = "postgres"
    $DATABASE_USER = "postgres"
    $DATABASE_PASSWORD = Read-UserInput -Prompt "Database Password" -Default "postgres"
    $DATABASE_NAME = "serenibase"
    Write-ColorOutput "   Using Docker database configuration" "Green"
} else {
    $DATABASE_HOST = Read-UserInput -Prompt "Database Host" -Default "localhost" -Required $true
    $DATABASE_USER = Read-UserInput -Prompt "Database User" -Default "postgres" -Required $true
    $DATABASE_PASSWORD = Read-UserInput -Prompt "Database Password" -Default "postgres" -Required $true -IsPassword $true
    $DATABASE_NAME = Read-UserInput -Prompt "Database Name" -Default "serenibase" -Required $true
}
Write-Host ""

# Email Configuration (Optional)
Write-ColorOutput "========================================================================" "Cyan"
Write-ColorOutput "                EMAIL CONFIGURATION (Optional)                          " "Cyan"
Write-ColorOutput "========================================================================" "Cyan"
Write-Host ""
Write-Host "Email is required for:"
Write-Host "  - Password reset functionality"
Write-Host "  - User notifications"
Write-Host ""
Write-Host "You can skip this and configure later."
Write-Host ""

$configureEmail = Read-UserInput -Prompt "Configure email now? (y/n)" -Default "n"
if ($configureEmail -eq "y" -or $configureEmail -eq "Y") {
    Write-Host ""
    Write-Host "Common SMTP configurations:"
    Write-Host "  Gmail:   smtp.gmail.com:587"
    Write-Host "  Outlook: smtp-mail.outlook.com:587"
    Write-Host ""
    
    $EMAIL_SMTP_HOST = Read-UserInput -Prompt "SMTP Host" -Default "smtp.gmail.com"
    $EMAIL_SMTP_PORT = Read-UserInput -Prompt "SMTP Port" -Default "587"
    $EMAIL_SMTP_USERNAME = Read-UserInput -Prompt "SMTP Username" -Required $true
    $EMAIL_SMTP_PASSWORD = Read-UserInput -Prompt "SMTP Password" -Required $true -IsPassword $true
    $EMAIL_FROM_EMAIL = Read-UserInput -Prompt "From Email" -Default $EMAIL_SMTP_USERNAME
} else {
    $EMAIL_SMTP_HOST = "smtp.gmail.com"
    $EMAIL_SMTP_PORT = "587"
    $EMAIL_SMTP_USERNAME = "your_email@gmail.com"
    $EMAIL_SMTP_PASSWORD = "your_app_password"
    $EMAIL_FROM_EMAIL = "your_email@gmail.com"
}
Write-Host ""

# Storage Configuration
Write-ColorOutput "========================================================================" "Cyan"
Write-ColorOutput "                    STORAGE CONFIGURATION                               " "Cyan"
Write-ColorOutput "========================================================================" "Cyan"
Write-Host ""
Write-Host "Storage options:"
Write-Host "  1. local  - Store files on disk [simple, default]"
Write-Host "  2. minio  - Use MinIO S3-compatible storage [Docker]"
Write-Host "  3. aws    - Use AWS S3 [production]"
Write-Host ""

$STORAGE_DRIVER = Read-UserInput -Prompt "Storage driver (local/minio/aws)" -Default "local"
Write-Host ""

# Generate .env file
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput "                 💾 GENERATING CONFIGURATION                    " "Cyan"
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-Host ""

$envContent = @"
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║                         SERENIBASE CONFIGURATION                              ║
# ║                  Generated by Interactive Setup Script                        ║
# ║                     $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")                                       ║
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
"@

# Write .env file
$envPath = Join-Path $PSScriptRoot ".env"
try {
    $envContent | Out-File -FilePath $envPath -Encoding UTF8 -NoNewline
    Write-ColorOutput "✓ Configuration file created: .env" "Green"
} catch {
    Write-ColorOutput "✗ Failed to create .env file: $_" "Red"
    exit 1
}

Write-Host ""

# Show summary
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-ColorOutput "                    ✅ SETUP COMPLETE!                          " "Cyan"
Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-Host ""

Write-ColorOutput "📝 Configuration Summary:" "Blue"
Write-Host "   • Access URL: http://$PUBLIC_HOST:8080"
Write-Host "   • Admin Email: $OWNER_EMAIL"
Write-Host "   • Database Host: $DATABASE_HOST"
Write-Host "   • Storage Driver: $STORAGE_DRIVER"
Write-Host "   • Config File: .env"
Write-Host ""

Write-ColorOutput "🚀 Next Steps:" "Blue"
Write-Host ""
Write-Host "1. Start the application:"
Write-ColorOutput "   docker-compose up -d" "Yellow"
Write-Host ""
Write-Host "2. Access the application:"
Write-ColorOutput "   http://$PUBLIC_HOST:8080" "Yellow"
Write-Host ""
Write-Host "3. Login with your admin credentials:"
Write-Host "   Email: $OWNER_EMAIL"
Write-Host "   Password: [the password you entered]"
Write-Host ""

if ($configureEmail -ne "y" -and $configureEmail -ne "Y") {
    Write-ColorOutput "⚠️  Note: Email is not configured. Password reset will not work." "Yellow"
    Write-Host "   To configure later, edit .env and set EMAIL_SMTP_* variables"
    Write-Host ""
}

Write-ColorOutput "📚 Documentation:" "Blue"
Write-Host "   • Environment Variables: docs/ENVIRONMENT_VARIABLES.md"
Write-Host "   • API Response Codes: docs/API_RESPONSE_CODES.md"
Write-Host "   • Setup Guide: README.md"
Write-Host ""

Write-ColorOutput "═══════════════════════════════════════════════════════════════" "Cyan"
Write-Host ""
