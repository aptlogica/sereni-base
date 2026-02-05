# ========================================================================
#                    CLONE MICROSERVICES SCRIPT
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

$SERVICES_DIR = "services"
$SERVICES_FILE = "build\scripts\services.list"

# Load .env if present
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

# Create services directory if missing
if (-not (Test-Path $SERVICES_DIR)) {
    New-Item -ItemType Directory -Path $SERVICES_DIR | Out-Null
}

# Process services
if (-not (Test-Path $SERVICES_FILE)) {
    Write-Host "[ERROR] Services list file not found: $SERVICES_FILE" -ForegroundColor Red
    exit 1
}

Get-Content $SERVICES_FILE | ForEach-Object {
    $line = $_.Trim()
    
    # Skip empty lines and comments
    if ([string]::IsNullOrWhiteSpace($line) -or $line.StartsWith("#")) {
        return
    }
    
    # Parse line: name repo [branch]
    $parts = $line -split '\s+'
    $name = $parts[0]
    $repo = $parts[1]
    $branch = if ($parts.Length -gt 2) { $parts[2] } else { "" }
    $target = "$SERVICES_DIR\$name"
    
    # Check if repo exists
    if (Test-Path "$target\.git") {
        $needReclone = $false
        
        # Check remote URL
        try {
            $currentUrl = (git -C $target remote get-url origin 2>$null).Trim()
            $currentBranch = (git -C $target rev-parse --abbrev-ref HEAD 2>$null).Trim()
            
            if ($currentUrl -ne $repo) {
                Write-Host "REMOTE URL mismatch for $name. Re-cloning..." -ForegroundColor Yellow
                $needReclone = $true
            } elseif (![string]::IsNullOrEmpty($branch) -and $currentBranch -ne $branch) {
                Write-Host "BRANCH mismatch for $name. Re-cloning..." -ForegroundColor Yellow
                $needReclone = $true
            }
        } catch {
            Write-Host "Error checking repository $name. Re-cloning..." -ForegroundColor Yellow
            $needReclone = $true
        }
        
        if ($needReclone) {
            Remove-Item -Recurse -Force $target
        } else {
            Write-Host "PULLING: $name" -ForegroundColor Cyan
            
            # Inject GIT_TOKEN for pull operation if available
            if (![string]::IsNullOrEmpty($GIT_TOKEN)) {
                # Construct authenticated URL for pull
                $authRepo = $repo -replace "https://", "https://${GIT_TOKEN}@"
                # Temporarily set remote URL with token
                git -C $target remote set-url origin $authRepo
                git -C $target pull
                # Restore original URL without token (for security)
                git -C $target remote set-url origin $repo
            } else {
                git -C $target pull
            }
            return
        }
    }
    
    # Inject GIT_TOKEN if available
    $authRepo = $repo
    if (![string]::IsNullOrEmpty($GIT_TOKEN)) {
        $authRepo = $repo -replace "https://", "https://${GIT_TOKEN}@"
    }
    
    # Clone
    Write-Host "CLONING: $name" -ForegroundColor Green
    try {
        if (![string]::IsNullOrEmpty($branch)) {
            git clone --branch $branch $authRepo $target
        } else {
            git clone $authRepo $target
        }
        Write-Host "[OK] Successfully cloned $name" -ForegroundColor Green
    } catch {
        Write-Host "[ERROR] Failed to clone $name : $_" -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "All microservices processed successfully!" -ForegroundColor Green
Write-Host ""
