# ========================================================================
#                    SERENIBASE SETUP SCRIPT
#                    Windows PowerShell Version
# ========================================================================
#
# Priority for environment variables:
#   1. Script parameters (highest priority)
#   2. Existing values from .env file (if exists)
#   3. Default variable values (lowest priority)
#
# Ctrl+C Handling:
#   Press Ctrl+C to immediately terminate the setup
#
# Usage:
#   .\setup.ps1                                      # Interactive mode
#   .\setup.ps1 -AutoYes                             # Non-interactive with defaults
#   .\setup.ps1 -SmtpHost "..." -SmtpPort "..." ... # With custom parameters
#
# ========================================================================

param(
    [switch]$AutoYes,  # For non-interactive setup with defaults
    [switch]$SkipDocker,  # Skip Docker and Docker Compose checks
    [switch]$Help,
    # SMTP Configuration
    [string]$SmtpHost = "",
    [string]$SmtpPort = "",
    [string]$SmtpUsername = "",
    [string]$SmtpPassword = "",
    [string]$SmtpFromEmail = "",
    # Additional environment variables can be passed as parameters
    # Example: .\setup.ps1 -PublicHost "myhost.com" -DatabaseHost "db.example.com"
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]]$UnnamedParameters
)

# Store all parameters in a hashtable for priority resolution
$ParameterValues = @{
    "auto-yes" = $AutoYes
    "smtp-host" = $SmtpHost
    "smtp-port" = $SmtpPort
    "smtp-username" = $SmtpUsername
    "smtp-password" = $SmtpPassword
    "smtp-from-email" = $SmtpFromEmail
}

$ErrorActionPreference = "Stop"

if ($Help) {
    Write-Host ""
    Write-Host "SereniBase Interactive Setup"
    Write-Host ""
    Write-Host "Usage:"
    Write-Host "  .\\setup.ps1                 # Interactive mode"
    Write-Host "  .\\setup.ps1 -AutoYes         # Non-interactive with defaults"
    Write-Host "  .\\setup.ps1 -SkipDocker      # Skip Docker prerequisite checks"
    Write-Host "  .\\setup.ps1 -Help            # Show this help"
    Write-Host ""
    exit 0
}

$SkipDockerCheck = $SkipDocker -or ($env:SETUP_SKIP_DOCKER -eq "1") -or ($env:SETUP_SKIP_DOCKER -eq "true")

# Trap Ctrl+C to exit immediately
$script:cancelled = $false
[Console]::TreatControlCAsInput = $false

trap {
    Write-Host "`n`n[!] Setup cancelled by user." -ForegroundColor Yellow
    exit 130
}

# Write UTF-8 text without BOM so Docker/.env parsers behave consistently across OSes.
function Set-TextFileNoBom {
    param(
        [string]$Path,
        [string]$Content
    )
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

# Function to read input with Ctrl+C support
function Read-HostWithCancel {
    param(
        [string]$Prompt,
        [string]$Default = ""
    )
    
    if ($Default) {
        $displayPrompt = "$Prompt [$Default]"
    } else {
        $displayPrompt = $Prompt
    }
    
    try {
        $userInput = Read-Host -Prompt $displayPrompt
        if ([string]::IsNullOrWhiteSpace($userInput)) {
            return $Default
        }
        return $userInput
    } catch {
        Write-Host "`n`n[!] Setup cancelled by user." -ForegroundColor Yellow
        exit 130
    }
}

# Function to read choice with Ctrl+C support
function Read-Choice {
    param(
        [string]$Prompt,
        [string]$Default = "1"
    )
    
    $displayPrompt = "$Prompt [$Default]"
    
    try {
        $userChoice = Read-Host -Prompt $displayPrompt
        if ([string]::IsNullOrWhiteSpace($userChoice)) {
            return $Default
        }
        return $userChoice
    } catch {
        Write-Host "`n`n[!] Setup cancelled by user." -ForegroundColor Yellow
        exit 130
    }
}

# Get the directory where this script is located
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
# Navigate to project root (two levels up from build/scripts/)
$ProjectRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
Set-Location $ProjectRoot

Write-Host ""
Write-Host "========================================================================"
Write-Host "                     SERENIBASE SETUP WIZARD"
Write-Host "========================================================================"
Write-Host ""

# Check prerequisites
Write-Host "Checking prerequisites..."
Write-Host ""

if ($SkipDockerCheck) {
    Write-Host "[!] Skipping Docker and Docker Compose checks" -ForegroundColor Yellow
} else {
    try {
        docker --version | Out-Null
        Write-Host "[OK] Docker is installed" -ForegroundColor Green
    } catch {
        Write-Host "[X] Docker is not installed. Please install Docker Desktop first." -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }

    try {
        docker compose version | Out-Null
        Write-Host "[OK] Docker Compose is installed" -ForegroundColor Green
    } catch {
        Write-Host "[X] Docker Compose is not installed." -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}

try {
    git --version | Out-Null
    Write-Host "[OK] Git is installed" -ForegroundColor Green
} catch {
    Write-Host "[X] Git is not installed. Please install Git first." -ForegroundColor Red
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host ""
Write-Host "All prerequisites satisfied!" -ForegroundColor Green
Write-Host ""

# Setup environment
Write-Host "Setting up environment..."

# Create a template .env content (using ASCII characters for compatibility)
$envTemplate = @"
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
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=168h
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
TEMPORARY_USER_PASSWORD=CHANGE_THIS      # Default password for new users

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
STORAGE_DRIVER=rustfs
STORAGE_DEV_PATH=./uploads
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=serenibase
STORAGE_AWS_ACCESS_KEY=your-access-key
STORAGE_AWS_SECRET_KEY=your-secret-key
RUSTFS_ENDPOINT=http://rustfs-server:9000
RUSTFS_ACCESS_KEY=rustfsadmin
RUSTFS_SECRET_KEY=rustfsadmin
RUSTFS_BUCKET=serenibase
RUSTFS_USE_SSL=false
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
"@

$envTemplatePath = Join-Path $ProjectRoot ".env.template"
$script:envPath = Join-Path $ProjectRoot ".env"

Set-TextFileNoBom -Path $envTemplatePath -Content $envTemplate

if (-not (Test-Path $script:envPath)) {
    Set-TextFileNoBom -Path $script:envPath -Content $envTemplate
    Write-Host "[OK] Created .env with default environment variables" -ForegroundColor Green
} else {
    Write-Host "[!] .env already exists. Checking for missing variables..." -ForegroundColor Yellow
    $appendScript = Join-Path $ProjectRoot "build\scripts\append-env-vars.ps1"
    if (Test-Path $appendScript) {
        & powershell -NoProfile -ExecutionPolicy Bypass -File $appendScript -TargetEnv $script:envPath -TemplateSource $envTemplatePath
    }
}

# Clean up template file
Remove-Item $envTemplatePath -ErrorAction SilentlyContinue

# Function to update .env file
function Update-EnvVar {
    param(
        [string]$Key,
        [string]$Value
    )
    $content = Get-Content $script:envPath -Raw -Encoding UTF8
    $pattern = "(?m)^$Key=.*$"
    if ($content -match $pattern) {
        $content = $content -replace $pattern, "$Key=$Value"
    } else {
        $content = $content.TrimEnd() + "`n$Key=$Value`n"
    }
    Set-TextFileNoBom -Path $script:envPath -Content $content
}

# Get existing environment variable value from .env file
function Get-EnvVar {
    param(
        [string]$Key
    )
    if (-not (Test-Path $script:envPath)) {
        return ""
    }
    $content = Get-Content $script:envPath -Raw -Encoding UTF8
    if ($content -match "(?m)^$Key=(.*)$") {
        # Remove any trailing carriage return (from Unix line endings)
        return $matches[1] -replace "`r", ''
    }
    return ""
}

# Resolve environment variable value with priority system
# Priority 1: Script parameter (highest)
# Priority 2: Existing .env value
# Priority 3: Default value (lowest)
function Resolve-EnvVar {
    param(
        [string]$Key,
        [string]$DefaultValue
    )
    
    # Priority 1: Check parameter values hashtable
    if ($ParameterValues.ContainsKey($Key) -and -not [string]::IsNullOrWhiteSpace($ParameterValues[$Key])) {
        return $ParameterValues[$Key]
    }
    
    # Convert hyphenated key to underscore for environment variable names
    $envVarKey = $Key -replace "-", "_"
    
    # Priority 2: Check existing .env file
    $existingValue = Get-EnvVar -Key $envVarKey
    if (-not [string]::IsNullOrWhiteSpace($existingValue)) {
        return $existingValue
    }
    
    # Priority 3: Use default value
    return $DefaultValue
}

# Prompt for value with priority system
# Priority 1: Script parameter (highest - can override anything)
# Priority 2: Existing .env value (protected - never prompted, never overridden)
# Priority 3: Default value (lowest - only used if no .env value)
function Read-EnvVar {
    param(
        [string]$Key,
        [string]$DefaultValue,
        [string]$Prompt,
        [bool]$IsPassword = $false
    )
    
    # Priority 1: If script parameter provided, use it (can override .env)
    if ($ParameterValues.ContainsKey($Key) -and -not [string]::IsNullOrWhiteSpace($ParameterValues[$Key])) {
        return $ParameterValues[$Key]
    }
    
    # Convert hyphenated key to underscore for environment variable names
    $envVarKey = $Key -replace "-", "_"
    
    # Priority 2: If value exists in .env, use it SILENTLY (never override)
    $existingValue = Get-EnvVar -Key $envVarKey
    if (-not [string]::IsNullOrWhiteSpace($existingValue)) {
        return $existingValue
    }
    
    # Priority 3: Value doesn't exist in .env, prompt or use default
    
    # If in auto-yes mode, use default without prompting
    if ($AutoYes) {
        return $DefaultValue
    }
    
    # Interactive mode: prompt user for new value
    if ($IsPassword) {
        $displayPrompt = if ($DefaultValue) { "$Prompt [$DefaultValue]" } else { $Prompt }
        $userInput = Read-Host -Prompt $displayPrompt -AsSecureString
        if ($userInput.Length -eq 0) {
            return $DefaultValue
        }
        # Convert SecureString back to plain text for storage
        $ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToCoTaskMemUnicode($userInput)
        return [System.Runtime.InteropServices.Marshal]::PtrToStringUni($ptr)
    } else {
        return Read-HostWithCancel -Prompt $Prompt -Default $DefaultValue
    }
}

# Update env var only if the new value is different from existing
function Update-EnvVarIfChanged {
    param(
        [string]$Key,
        [string]$Value
    )
    $existingValue = Get-EnvVar -Key $Key
    
    # Only update if the value is different or doesn't exist yet
    if ($existingValue -ne $Value) {
        Update-EnvVar -Key $Key -Value $Value
    }
}

# Check if a variable already exists in .env (NEVER override if exists)
function Test-EnvVarExists {
    param([string]$Key)
    $existingValue = Get-EnvVar -Key $Key
    return -not [string]::IsNullOrWhiteSpace($existingValue)
}

# Check if ALL variables in the list exist in .env
function Test-AllEnvVarsExist {
    param([string[]]$Keys)
    foreach ($key in $Keys) {
        if (-not (Test-EnvVarExists -Key $key)) {
            return $false
        }
    }
    return $true
}

# ========================================================================
#                      DATABASE CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      DATABASE CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""

# Check if ALL database variables already exist in .env
# If they do, skip this entire section (NEVER override)
if (Test-AllEnvVarsExist -Keys @("DATABASE_HOST", "DATABASE_PORT", "DATABASE_USER", "DATABASE_PASSWORD", "DATABASE_NAME", "DATABASE_SSL_MODE")) {
    Write-Host "[OK] Database configuration already set in .env (skipping)" -ForegroundColor Green
} else {
    
Write-Host "Choose database setup:"
Write-Host "  1. Use default PostgreSQL (Docker container)"
Write-Host "  2. Use custom database credentials"
Write-Host ""

if ($AutoYes) {
    $DB_CHOICE = "1"
} else {
    $DB_CHOICE = Read-Choice -Prompt "Enter choice" -Default "1"
}

if ($DB_CHOICE -eq "1") {
    Write-Host ""
    Write-Host "Using default PostgreSQL Docker container"
    Write-Host ""
    
    if ($AutoYes) {
        $DATABASE_USER = "postgres"
        $DATABASE_PASSWORD = "postgres"
        $DATABASE_NAME = "serenibase"
    } else {
        $DATABASE_USER = Read-EnvVar -Key "DATABASE_USER" -DefaultValue "postgres" -Prompt "Database User"
        $DATABASE_PASSWORD = Read-EnvVar -Key "DATABASE_PASSWORD" -DefaultValue "postgres" -Prompt "Database Password" -IsPassword $true
        $DATABASE_NAME = Read-EnvVar -Key "DATABASE_NAME" -DefaultValue "serenibase" -Prompt "Database Name"
    }
    
    # DATABASE_PORT and DATABASE_HOST are for external/host connections
    # For Docker-to-Docker internal connections, use DATABASE_INTERNAL_HOST and DATABASE_INTERNAL_PORT
    if (Test-EnvVarExists -Key "DATABASE_PORT") {
        $DATABASE_PORT = Get-EnvVar -Key "DATABASE_PORT"
    } else {
        $DATABASE_PORT = "5432"
    }
    
    if (Test-EnvVarExists -Key "DATABASE_HOST") {
        $DATABASE_HOST = Get-EnvVar -Key "DATABASE_HOST"
    } else {
        $DATABASE_HOST = "postgres"
    }
    
    if (Test-EnvVarExists -Key "DATABASE_SSL_MODE") {
        $DATABASE_SSL_MODE = Get-EnvVar -Key "DATABASE_SSL_MODE"
    } else {
        $DATABASE_SSL_MODE = "disable"
    }
    
    # Set internal Docker connection variables (always use for container-to-container communication)
    $DATABASE_INTERNAL_HOST = "postgres"
    $DATABASE_INTERNAL_PORT = "5432"
} else {
    Write-Host ""
    Write-Host "Enter custom database configuration:"
    Write-Host ""
    
    $DATABASE_HOST = Read-EnvVar -Key "DATABASE_HOST" -DefaultValue "" -Prompt "Database Host"
    if ([string]::IsNullOrWhiteSpace($DATABASE_HOST)) {
        Write-Host "[ERROR] Database host is required" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
    
    $DATABASE_PORT = Read-EnvVar -Key "DATABASE_PORT" -DefaultValue "5432" -Prompt "Database Port"
    
    $DATABASE_USER = Read-EnvVar -Key "DATABASE_USER" -DefaultValue "" -Prompt "Database User"
    if ([string]::IsNullOrWhiteSpace($DATABASE_USER)) {
        Write-Host "[ERROR] Database user is required" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
    
    $DATABASE_PASSWORD = Read-EnvVar -Key "DATABASE_PASSWORD" -DefaultValue "" -Prompt "Database Password" -IsPassword $true
    if ([string]::IsNullOrWhiteSpace($DATABASE_PASSWORD)) {
        Write-Host "[ERROR] Database password is required" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
    
    $DATABASE_NAME = Read-EnvVar -Key "DATABASE_NAME" -DefaultValue "" -Prompt "Database Name"
    if ([string]::IsNullOrWhiteSpace($DATABASE_NAME)) {
        Write-Host "[ERROR] Database name is required" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
    
    $DATABASE_SSL_MODE = Read-EnvVar -Key "DATABASE_SSL_MODE" -DefaultValue "disable" -Prompt "SSL Mode"
}

Update-EnvVarIfChanged -Key "DATABASE_HOST" -Value $DATABASE_HOST
Update-EnvVarIfChanged -Key "DATABASE_PORT" -Value $DATABASE_PORT
Update-EnvVarIfChanged -Key "DATABASE_USER" -Value $DATABASE_USER
Update-EnvVarIfChanged -Key "DATABASE_PASSWORD" -Value $DATABASE_PASSWORD
Update-EnvVarIfChanged -Key "DATABASE_NAME" -Value $DATABASE_NAME
Update-EnvVarIfChanged -Key "DATABASE_SSL_MODE" -Value $DATABASE_SSL_MODE

# Set internal Docker connection variables (used by services inside docker-compose)
# These are always postgres:5432 for container-to-container communication
if ($DB_CHOICE -eq "1") {
    Update-EnvVarIfChanged -Key "DATABASE_INTERNAL_HOST" -Value "postgres"
    Update-EnvVarIfChanged -Key "DATABASE_INTERNAL_PORT" -Value "5432"
}

Write-Host "[OK] Database configuration updated" -ForegroundColor Green

} # End of database configuration else block

# ========================================================================
#                      AUTHENTICATION CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      AUTHENTICATION CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""

# Check if JWT secret already exists in .env AND is not the default placeholder
$currentJwtSecret = Get-EnvVar -Key "AUTH_JWT_SECRET"
$isDefaultJwtSecret = ($currentJwtSecret -eq "change-this-to-a-secure-random-string-min32chars") -or [string]::IsNullOrWhiteSpace($currentJwtSecret)

if (-not $isDefaultJwtSecret) {
    Write-Host "[OK] JWT Secret already set in .env (skipping)" -ForegroundColor Green
} else {

if ($AutoYes) {
    $AUTH_JWT_SECRET = [System.Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes((New-Guid).ToString() + (New-Guid).ToString())).Substring(0,32)
    Write-Host "Generated JWT Secret: $AUTH_JWT_SECRET"
} else {
    $AUTH_JWT_SECRET = Read-HostWithCancel -Prompt "JWT Secret (min 32 chars) [press Enter to generate]"
    if ([string]::IsNullOrWhiteSpace($AUTH_JWT_SECRET)) {
        $AUTH_JWT_SECRET = [System.Convert]::ToBase64String([System.Text.Encoding]::UTF8.GetBytes((New-Guid).ToString() + (New-Guid).ToString())).Substring(0,32)
        Write-Host "Generated JWT Secret: $AUTH_JWT_SECRET"
    }
}

Update-EnvVarIfChanged -Key "AUTH_JWT_SECRET" -Value $AUTH_JWT_SECRET
Write-Host "[OK] JWT Secret configured" -ForegroundColor Green

} # End of JWT configuration else block

# ========================================================================
#                      EMAIL CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      EMAIL CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""

Write-Host "Enter SMTP email configuration (press Enter to keep existing values):"
Write-Host ""

# Get current values from .env to use as defaults
$currentSmtpHost = Get-EnvVar -Key "EMAIL_SMTP_HOST"
$currentSmtpPort = Get-EnvVar -Key "EMAIL_SMTP_PORT"
$currentSmtpUsername = Get-EnvVar -Key "EMAIL_SMTP_USERNAME"
$currentSmtpPassword = Get-EnvVar -Key "EMAIL_SMTP_PASSWORD"
$currentFromEmail = Get-EnvVar -Key "EMAIL_FROM_EMAIL"

# Always prompt for each field, using .env value as default (or use parameter if provided)
if (-not [string]::IsNullOrWhiteSpace($SmtpHost)) {
    $EMAIL_SMTP_HOST = $SmtpHost
} else {
    if ($AutoYes) {
        $EMAIL_SMTP_HOST = if ([string]::IsNullOrWhiteSpace($currentSmtpHost)) { "smtp.gmail.com" } else { $currentSmtpHost }
    } else {
        $defaultHost = if ([string]::IsNullOrWhiteSpace($currentSmtpHost)) { "smtp.gmail.com" } else { $currentSmtpHost }
        $userInput = Read-Host "SMTP Host [$defaultHost]"
        $EMAIL_SMTP_HOST = if ([string]::IsNullOrWhiteSpace($userInput)) { $defaultHost } else { $userInput }
    }
}

if (-not [string]::IsNullOrWhiteSpace($SmtpPort)) {
    $EMAIL_SMTP_PORT = $SmtpPort
} else {
    if ($AutoYes) {
        $EMAIL_SMTP_PORT = if ([string]::IsNullOrWhiteSpace($currentSmtpPort)) { "587" } else { $currentSmtpPort }
    } else {
        $defaultPort = if ([string]::IsNullOrWhiteSpace($currentSmtpPort)) { "587" } else { $currentSmtpPort }
        $userInput = Read-Host "SMTP Port [$defaultPort]"
        $EMAIL_SMTP_PORT = if ([string]::IsNullOrWhiteSpace($userInput)) { $defaultPort } else { $userInput }
    }
}

if (-not [string]::IsNullOrWhiteSpace($SmtpUsername)) {
    $EMAIL_SMTP_USERNAME = $SmtpUsername
} else {
    if ($AutoYes) {
        $EMAIL_SMTP_USERNAME = $currentSmtpUsername
    } else {
        $userInput = Read-Host "SMTP Username (email) [$currentSmtpUsername]"
        $EMAIL_SMTP_USERNAME = if ([string]::IsNullOrWhiteSpace($userInput)) { $currentSmtpUsername } else { $userInput }
    }
    if ([string]::IsNullOrWhiteSpace($EMAIL_SMTP_USERNAME)) {
        Write-Host "[ERROR] SMTP username is required" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}

if (-not [string]::IsNullOrWhiteSpace($SmtpPassword)) {
    $EMAIL_SMTP_PASSWORD = $SmtpPassword
} else {
    if ($AutoYes) {
        $EMAIL_SMTP_PASSWORD = $currentSmtpPassword
    } else {
        $inputPassword = Read-Host "SMTP Password (app password) [$currentSmtpPassword]" -AsSecureString
        if ($inputPassword.Length -eq 0) {
            $EMAIL_SMTP_PASSWORD = $currentSmtpPassword
        } else {
            $ptr = [System.Runtime.InteropServices.Marshal]::SecureStringToCoTaskMemUnicode($inputPassword)
            $EMAIL_SMTP_PASSWORD = [System.Runtime.InteropServices.Marshal]::PtrToStringUni($ptr)
        }
    }
    if ([string]::IsNullOrWhiteSpace($EMAIL_SMTP_PASSWORD)) {
        Write-Host "[ERROR] SMTP password is required" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}

if (-not [string]::IsNullOrWhiteSpace($SmtpFromEmail)) {
    $EMAIL_FROM_EMAIL = $SmtpFromEmail
} else {
    if ($AutoYes) {
        $EMAIL_FROM_EMAIL = if ([string]::IsNullOrWhiteSpace($currentFromEmail)) { $EMAIL_SMTP_USERNAME } else { $currentFromEmail }
    } else {
        $defaultFromEmail = if ([string]::IsNullOrWhiteSpace($currentFromEmail)) { $EMAIL_SMTP_USERNAME } else { $currentFromEmail }
        $userInput = Read-Host "From Email [$defaultFromEmail]"
        $EMAIL_FROM_EMAIL = if ([string]::IsNullOrWhiteSpace($userInput)) { $defaultFromEmail } else { $userInput }
    }
}

Update-EnvVarIfChanged -Key "EMAIL_SMTP_HOST" -Value $EMAIL_SMTP_HOST
Update-EnvVarIfChanged -Key "EMAIL_SMTP_PORT" -Value $EMAIL_SMTP_PORT
Update-EnvVarIfChanged -Key "EMAIL_SMTP_USERNAME" -Value $EMAIL_SMTP_USERNAME
Update-EnvVarIfChanged -Key "EMAIL_SMTP_PASSWORD" -Value $EMAIL_SMTP_PASSWORD
Update-EnvVarIfChanged -Key "EMAIL_FROM_EMAIL" -Value $EMAIL_FROM_EMAIL

Write-Host "[OK] Email configuration updated" -ForegroundColor Green

# ========================================================================
#                      STORAGE CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      STORAGE CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""
Write-Host "Choose storage driver:"
Write-Host "  1. Local filesystem (for development only)"
Write-Host "  2. RustFS (Docker container - recommended)"
Write-Host "  3. RustFS Custom (external RustFS server)"
Write-Host "  4. AWS S3"
Write-Host ""

if ($AutoYes) {
    $STORAGE_CHOICE = "2"
} else {
    $STORAGE_CHOICE = Read-Choice -Prompt "Enter choice" -Default "2"
}

switch ($STORAGE_CHOICE) {
    "1" {
        Write-Host ""
        Write-Host "Using local filesystem storage"
        $STORAGE_DRIVER = "local"
        
        if ($AutoYes) {
            $STORAGE_DEV_PATH = "./uploads"
        } else {
            $STORAGE_DEV_PATH = Read-HostWithCancel -Prompt "Storage path" -Default "./uploads"
        }
        
        Update-EnvVarIfChanged -Key "STORAGE_DRIVER" -Value "local"
        Update-EnvVarIfChanged -Key "STORAGE_DEV_PATH" -Value $STORAGE_DEV_PATH
        Write-Host "[OK] Local filesystem storage configured" -ForegroundColor Green
    }
    "2" {
        Write-Host ""
        Write-Host "Using default RustFS Docker container"
        
        if ($AutoYes) {
            $RUSTFS_ACCESS_KEY = "rustfsadmin"
            $RUSTFS_SECRET_KEY = "rustfsadmin"
            $RUSTFS_BUCKET = "serenibase"
        } else {
            $RUSTFS_ACCESS_KEY = Read-HostWithCancel -Prompt "RustFS Access Key" -Default "rustfsadmin"
            $RUSTFS_SECRET_KEY = Read-HostWithCancel -Prompt "RustFS Secret Key" -Default "rustfsadmin"
            $RUSTFS_BUCKET = Read-HostWithCancel -Prompt "Bucket Name" -Default "serenibase"
        }
        
        Update-EnvVarIfChanged -Key "STORAGE_DRIVER" -Value "rustfs"
        Update-EnvVarIfChanged -Key "RUSTFS_ENDPOINT" -Value "http://rustfs-server:9000"
        Update-EnvVarIfChanged -Key "RUSTFS_ACCESS_KEY" -Value $RUSTFS_ACCESS_KEY
        Update-EnvVarIfChanged -Key "RUSTFS_SECRET_KEY" -Value $RUSTFS_SECRET_KEY
        Update-EnvVarIfChanged -Key "RUSTFS_BUCKET" -Value $RUSTFS_BUCKET
        Update-EnvVarIfChanged -Key "RUSTFS_USE_SSL" -Value "false"
        Write-Host "[OK] RustFS Docker storage configured" -ForegroundColor Green
    }
    "3" {
        Write-Host ""
        Write-Host "Enter custom RustFS configuration:"
        Write-Host ""
        
        $RUSTFS_ENDPOINT = Read-HostWithCancel -Prompt "RustFS Endpoint (host:port)" -Default "http://rustfs-server:9000"
        if ([string]::IsNullOrWhiteSpace($RUSTFS_ENDPOINT)) {
            Write-Host "[ERROR] RustFS endpoint is required" -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
        
        $RUSTFS_ACCESS_KEY = Read-HostWithCancel -Prompt "RustFS Access Key" -Default "rustfsadmin"
        if ([string]::IsNullOrWhiteSpace($RUSTFS_ACCESS_KEY)) {
            Write-Host "[ERROR] RustFS access key is required" -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
        
        $RUSTFS_SECRET_KEY = Read-HostWithCancel -Prompt "RustFS Secret Key" -Default "rustfsadmin"
        if ([string]::IsNullOrWhiteSpace($RUSTFS_SECRET_KEY)) {
            Write-Host "[ERROR] RustFS secret key is required" -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
        
        $RUSTFS_BUCKET = Read-HostWithCancel -Prompt "Bucket Name" -Default "serenibase"
        $RUSTFS_USE_SSL = Read-HostWithCancel -Prompt "Use SSL (true/false)" -Default "false"
        
        Update-EnvVarIfChanged -Key "STORAGE_DRIVER" -Value "rustfs"
        Update-EnvVarIfChanged -Key "RUSTFS_ENDPOINT" -Value $RUSTFS_ENDPOINT
        Update-EnvVarIfChanged -Key "RUSTFS_ACCESS_KEY" -Value $RUSTFS_ACCESS_KEY
        Update-EnvVarIfChanged -Key "RUSTFS_SECRET_KEY" -Value $RUSTFS_SECRET_KEY
        Update-EnvVarIfChanged -Key "RUSTFS_BUCKET" -Value $RUSTFS_BUCKET
        Update-EnvVarIfChanged -Key "RUSTFS_USE_SSL" -Value $RUSTFS_USE_SSL
        Write-Host "[OK] Custom RustFS storage configured" -ForegroundColor Green
    }
    "4" {
        Write-Host ""
        Write-Host "Enter AWS S3 configuration:"
        Write-Host ""
        
        $STORAGE_AWS_REGION = Read-HostWithCancel -Prompt "AWS Region" -Default "us-east-1"
        
        $STORAGE_AWS_BUCKET = Read-HostWithCancel -Prompt "S3 Bucket Name"
        if ([string]::IsNullOrWhiteSpace($STORAGE_AWS_BUCKET)) {
            Write-Host "[ERROR] S3 bucket name is required" -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
        
        $STORAGE_AWS_ACCESS_KEY = Read-HostWithCancel -Prompt "AWS Access Key"
        if ([string]::IsNullOrWhiteSpace($STORAGE_AWS_ACCESS_KEY)) {
            Write-Host "[ERROR] AWS access key is required" -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
        
        $STORAGE_AWS_SECRET_KEY = Read-HostWithCancel -Prompt "AWS Secret Key"
        if ([string]::IsNullOrWhiteSpace($STORAGE_AWS_SECRET_KEY)) {
            Write-Host "[ERROR] AWS secret key is required" -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
        
        Update-EnvVarIfChanged -Key "STORAGE_DRIVER" -Value "s3"
        Update-EnvVarIfChanged -Key "STORAGE_AWS_REGION" -Value $STORAGE_AWS_REGION
        Update-EnvVarIfChanged -Key "STORAGE_AWS_BUCKET" -Value $STORAGE_AWS_BUCKET
        Update-EnvVarIfChanged -Key "STORAGE_AWS_ACCESS_KEY" -Value $STORAGE_AWS_ACCESS_KEY
        Update-EnvVarIfChanged -Key "STORAGE_AWS_SECRET_KEY" -Value $STORAGE_AWS_SECRET_KEY
        Write-Host "[OK] AWS S3 storage configured" -ForegroundColor Green
    }
    default {
        Write-Host "[ERROR] Invalid choice" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}

# ========================================================================
#                      NETWORK CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      NETWORK CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""
Write-Host "Enter PUBLIC_HOST configuration (press Enter to keep existing values):"
Write-Host "(Examples: 192.168.1.100, myapp.example.com, or localhost for local development)"
Write-Host ""

if ($AutoYes) {
    $currentPublicHost = Get-EnvVar -Key "PUBLIC_HOST"
    $PUBLIC_HOST = if ([string]::IsNullOrWhiteSpace($currentPublicHost)) { "localhost" } else { $currentPublicHost }
} else {
    $currentPublicHost = Get-EnvVar -Key "PUBLIC_HOST"
    $defaultPublicHost = if ([string]::IsNullOrWhiteSpace($currentPublicHost)) { "localhost" } else { $currentPublicHost }
    $PUBLIC_HOST = Read-HostWithCancel -Prompt "PUBLIC_HOST" -Default $defaultPublicHost
}

# Ensure PUBLIC_HOST is never empty
if ([string]::IsNullOrWhiteSpace($PUBLIC_HOST)) {
    $PUBLIC_HOST = "localhost"
}

Update-EnvVarIfChanged -Key "PUBLIC_HOST" -Value $PUBLIC_HOST
Update-EnvVarIfChanged -Key "SERVER_IP" -Value $PUBLIC_HOST
Update-EnvVarIfChanged -Key "BASEUI_VITE_API_BASE_URL" -Value "http://${PUBLIC_HOST}:8080"
Update-EnvVarIfChanged -Key "CORS_ALLOWED_ORIGINS" -Value "http://localhost:5050,http://127.0.0.1:5050,http://${PUBLIC_HOST}:5050,http://base-ui:5050,http://serenibase:8080"
Update-EnvVarIfChanged -Key "STORAGE_SERVER_IP" -Value $PUBLIC_HOST
Update-EnvVarIfChanged -Key "AUTH_RESET_PASSWORD_URL" -Value "http://${PUBLIC_HOST}:5050/reset-password?token=%s"

Write-Host "[OK] Configured PUBLIC_HOST" -ForegroundColor Green
Write-Host "[OK] Configured SERVER_IP" -ForegroundColor Green
Write-Host "[OK] Configured BASEUI_VITE_API_BASE_URL" -ForegroundColor Green
Write-Host "[OK] Configured AUTH_RESET_PASSWORD_URL" -ForegroundColor Green

# ========================================================================
#                   OWNER REGISTRATION CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                   OWNER REGISTRATION CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""
Write-Host "Enter owner registration details (press Enter to keep existing values):"
Write-Host ""

if ($AutoYes) {
    $OWNER_FIRST_NAME = "Admin"
    $OWNER_LAST_NAME = "User"
    $OWNER_EMAIL = "admin@example.com"
    $OWNER_PASSWORD = "Admin@123"
} else {
    $OWNER_FIRST_NAME = Read-EnvVar -Key "OWNER_FIRST_NAME" -DefaultValue "Admin" -Prompt "First Name"
    $OWNER_LAST_NAME = Read-EnvVar -Key "OWNER_LAST_NAME" -DefaultValue "User" -Prompt "Last Name"
    $OWNER_EMAIL = Read-EnvVar -Key "OWNER_EMAIL" -DefaultValue "admin@example.com" -Prompt "Email"
    $OWNER_PASSWORD = Read-EnvVar -Key "OWNER_PASSWORD" -DefaultValue "Admin@123" -Prompt "Password" -IsPassword $true
}

Update-EnvVarIfChanged -Key "OWNER_FIRST_NAME" -Value $OWNER_FIRST_NAME
Update-EnvVarIfChanged -Key "OWNER_LAST_NAME" -Value $OWNER_LAST_NAME
Update-EnvVarIfChanged -Key "OWNER_EMAIL" -Value $OWNER_EMAIL
Update-EnvVarIfChanged -Key "OWNER_PASSWORD" -Value $OWNER_PASSWORD

Write-Host "[OK] Owner configuration updated" -ForegroundColor Green

# ========================================================================
#                      PORT CONFIGURATION
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      PORT CONFIGURATION"
Write-Host "========================================================================"
Write-Host ""
Write-Host "Configure container ports (press Enter to use defaults):"
Write-Host ""

if ($AutoYes) {
    $RUSTFS_API_PORT = "9000"
    $RUSTFS_CONSOLE_PORT = "9001"
    $BASE_UI_PORT = "5050"
    $ANTIVIRUS_CLAMAV_PORT = "3310"
} else {
    $RUSTFS_API_PORT = Read-HostWithCancel -Prompt "RustFS API Port" -Default "9000"
    $RUSTFS_CONSOLE_PORT = Read-HostWithCancel -Prompt "RustFS Console Port" -Default "9001"
    $BASE_UI_PORT = Read-HostWithCancel -Prompt "Base UI Port" -Default "5050"
    $ANTIVIRUS_CLAMAV_PORT = Read-HostWithCancel -Prompt "ClamAV Port" -Default "3310"
}

Update-EnvVarIfChanged -Key "RUSTFS_API_PORT" -Value $RUSTFS_API_PORT
Update-EnvVarIfChanged -Key "RUSTFS_CONSOLE_PORT" -Value $RUSTFS_CONSOLE_PORT
Update-EnvVarIfChanged -Key "BASE_UI_PORT" -Value $BASE_UI_PORT
Update-EnvVarIfChanged -Key "ANTIVIRUS_CLAMAV_PORT" -Value $ANTIVIRUS_CLAMAV_PORT

Write-Host "[OK] Port configuration updated" -ForegroundColor Green

# ========================================================================
#                      CLONING REPOSITORIES
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      CLONING REPOSITORIES"
Write-Host "========================================================================"
Write-Host ""

if (Test-Path "build\scripts\clone-services.ps1") {
    Write-Host "Cloning microservices..."
    & powershell -NoProfile -ExecutionPolicy Bypass -File "build\scripts\clone-services.ps1"
    Write-Host "[OK] Cloned microservices" -ForegroundColor Green
}

# ========================================================================
#                      STARTING SERVICES
# ========================================================================

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      STARTING SERVICES"
Write-Host "========================================================================"
Write-Host ""

docker compose -f docker-compose.all.yaml up --build -d

Write-Host ""
Write-Host "Waiting for services to start..."
Start-Sleep -Seconds 10

docker compose -f docker-compose.all.yaml ps

$psOutput = docker compose -f docker-compose.all.yaml ps | Out-String
if ($psOutput -match "(?im)unhealthy|exited|restarting") {
    Write-Host ""
    Write-Host "[!] Some services are not healthy yet. Collecting quick diagnostics..." -ForegroundColor Yellow
    Write-Host $psOutput

    $badServices = @()
    foreach ($line in ($psOutput -split "`r?`n")) {
        if ($line -match "(?i)unhealthy|exited|restarting") {
            $serviceName = ($line -split "\s+")[0]
            if (-not [string]::IsNullOrWhiteSpace($serviceName)) {
                $badServices += $serviceName
            }
        }
    }
    $badServices = $badServices | Select-Object -Unique

    foreach ($service in $badServices) {
        Write-Host ""
        Write-Host "----- Last logs for $service -----" -ForegroundColor Yellow
        docker compose -f docker-compose.all.yaml logs --tail=80 $service
    }
    Write-Host ""
    Write-Host "[!] Fix the failing services above, then run: docker compose -f docker-compose.all.yaml up -d" -ForegroundColor Yellow
}

# Read final values from .env for display (with fallbacks)
$displayPublicHost = Get-EnvVar -Key "PUBLIC_HOST"
if ([string]::IsNullOrWhiteSpace($displayPublicHost)) { $displayPublicHost = "localhost" }
$displayOwnerEmail = Get-EnvVar -Key "OWNER_EMAIL"
if ([string]::IsNullOrWhiteSpace($displayOwnerEmail)) { $displayOwnerEmail = "admin@example.com" }
$displayOwnerPassword = Get-EnvVar -Key "OWNER_PASSWORD"
if ([string]::IsNullOrWhiteSpace($displayOwnerPassword)) { $displayOwnerPassword = "Admin@123" }

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      SETUP COMPLETE!"
Write-Host "========================================================================"
Write-Host ""
Write-Host "Access your application at:"
Write-Host "  Frontend:  http://${displayPublicHost}:5050"
Write-Host "  Backend:   http://${displayPublicHost}:8080"
Write-Host "  RustFS:     http://${displayPublicHost}:9001"
Write-Host ""
Write-Host "Default admin credentials:"
Write-Host "  Email:    $displayOwnerEmail"
Write-Host "  Password: $displayOwnerPassword"
Write-Host ""
Write-Host "NOTE: Timezone is set to UTC. You can change it from Profile settings." -ForegroundColor Yellow
Write-Host ""
Write-Host "WARNING: Remember to change default passwords in production!" -ForegroundColor Yellow
Write-Host ""
Write-Host "Useful commands:"
Write-Host "  make logs      - View service logs"
Write-Host "  make down-all  - Stop all services"
Write-Host "  make clean     - Remove all data"
Write-Host ""

if (-not $AutoYes) {
    Read-Host "Press Enter to exit"
}

