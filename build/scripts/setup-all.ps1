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
