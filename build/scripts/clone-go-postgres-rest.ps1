# Clone go-postgres-rest for Windows PowerShell
$ErrorActionPreference = "Stop"

# Get script directory and navigate to project root
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = Split-Path -Parent (Split-Path -Parent $scriptDir)
Set-Location $projectRoot

# Load .env if present for GIT_TOKEN
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match "^\s*([^#=]+)\s*=\s*(.+)\s*$") {
            [System.Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
}

$repoUrl = "https://github.com/aptlogica/go-postgres-rest.git"
$targetDir = "go-postgres-rest"

# Always remove and re-clone for a clean state
if (Test-Path $targetDir) {
    Write-Host "Removing existing $targetDir..."
    Remove-Item -Recurse -Force $targetDir
}

if ($env:GIT_TOKEN) {
    $repoUrl = $repoUrl -replace '^https://', "https://$($env:GIT_TOKEN)@"
}

Write-Host "Cloning $repoUrl into $targetDir..."
git clone $repoUrl $targetDir

# Clean Go module cache (if go is available)
if (Get-Command go -ErrorAction SilentlyContinue) {
    Write-Host "Cleaning Go module cache..."
    go clean -modcache
}

Write-Host "go-postgres-rest cloned successfully!"
