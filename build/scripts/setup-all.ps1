# Full setup script for Windows PowerShell
# Clones all services and starts Docker Compose

$ErrorActionPreference = "Stop"

# Get script directory and navigate to project root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)
Set-Location $projectRoot

Write-Host ""
Write-Host "========================================================================"
Write-Host "                     SERENIBASE FULL SETUP"
Write-Host "========================================================================"
Write-Host ""

# Initialize .env file with defaults if it doesn't exist
Write-Host "Setting up environment configuration..."
if (-not (Test-Path ".env")) {
    Write-Host "Creating .env file from template..."
    Copy-Item "build\config\.env.example" ".env"
    
    # Set PUBLIC_HOST to localhost default (case-insensitive)
    $content = Get-Content ".env" -Raw
    $content = $content -replace '\$\{PUBLIC_HOST\}', 'localhost'
    $content = $content -replace '\$\{public_host\}', 'localhost'
    Set-Content ".env" -Value $content -NoNewline
    Write-Host "[OK] .env file created with defaults"
} else {
    Write-Host "[OK] .env file already exists"
}

Write-Host ""

# Clone main services
Write-Host "Cloning all service repos..."
& "$scriptDir\clone-services.ps1"

# Clone go-postgres-rest
Write-Host "Cloning go-postgres-rest..."
& "$scriptDir\clone-go-postgres-rest.ps1"

# Run docker compose
Write-Host ""
Write-Host "Starting all services with Docker Compose..."
docker compose -f docker-compose.all.yaml up --build -d

Write-Host ""
Write-Host "========================================================================"
Write-Host "                      SETUP COMPLETE!"
Write-Host "========================================================================"
Write-Host ""
Write-Host "Access your application at:"
Write-Host "  - Frontend: http://localhost:5050"
Write-Host "  - Backend:  http://localhost:8080"
Write-Host ""
