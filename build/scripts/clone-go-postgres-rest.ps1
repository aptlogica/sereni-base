# ========================================================================
#                    CLONE GO-POSTGRES-REST SCRIPT
#                       PowerShell Version
# ========================================================================

$ErrorActionPreference = "Stop"

# Cleanup function to handle Ctrl+C
function Cleanup {
    Write-Host ""
    Write-Host "[!] Clone interrupted by user." -ForegroundColor Yellow
    exit 1
}

# Register Ctrl+C handler
$null = Register-EngineEvent PowerShell.Exiting -Action { Cleanup }

# Get script directory and project root
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
$PROJECT_ROOT = Split-Path -Parent (Split-Path -Parent $SCRIPT_DIR)

# Change to project root
Set-Location $PROJECT_ROOT

# Load .env if present for GIT_TOKEN
$GIT_TOKEN = ""
if (Test-Path ".env") {
    Write-Host "[INFO] Loading .env file..." -ForegroundColor Cyan
    $envContent = Get-Content ".env" -Raw
    if ($envContent -match "GIT_TOKEN=(.+?)(\r?\n|$)") {
        $GIT_TOKEN = $matches[1].Trim()
    }
}

# Debug: Check if GIT_TOKEN is loaded
if ([string]::IsNullOrEmpty($GIT_TOKEN)) {
    Write-Host "[INFO] GIT_TOKEN not set, cloning without authentication" -ForegroundColor Yellow
} else {
    Write-Host "[INFO] GIT_TOKEN is set" -ForegroundColor Green
}

$REPO_URL = "https://github.com/aptlogica/go-postgres-rest.git"
$TARGET_DIR = "go-postgres-rest"

# Always remove and re-clone for a clean state
if (Test-Path $TARGET_DIR) {
    Write-Host "Removing existing $TARGET_DIR..." -ForegroundColor Yellow
    Remove-Item -Recurse -Force $TARGET_DIR
}

# Inject GIT_TOKEN if available
$authRepoUrl = $REPO_URL
if (![string]::IsNullOrEmpty($GIT_TOKEN)) {
    $authRepoUrl = $REPO_URL -replace "https://", "https://${GIT_TOKEN}@"
}

Write-Host "Cloning $REPO_URL into $TARGET_DIR..." -ForegroundColor Cyan

try {
    git clone $authRepoUrl $TARGET_DIR
    Write-Host "[OK] go-postgres-rest cloned successfully!" -ForegroundColor Green
} catch {
    Write-Host "[ERROR] Failed to clone go-postgres-rest: $_" -ForegroundColor Red
    exit 1
}

# Clean Go module cache (if go is available)
if (Get-Command go -ErrorAction SilentlyContinue) {
    Write-Host "Cleaning Go module cache..." -ForegroundColor Cyan
    go clean -modcache
}

Write-Host ""
Write-Host "go-postgres-rest setup complete!" -ForegroundColor Green
Write-Host ""
