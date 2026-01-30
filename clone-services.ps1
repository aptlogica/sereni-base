$ErrorActionPreference = "Stop"

$servicesDir  = "services"
$servicesFile = "services.list"

# Load .env if present
if (Test-Path ".env") {
    Get-Content ".env" | ForEach-Object {
        if ($_ -match "^\s*([^#=]+)\s*=\s*(.+)\s*$") {
            [System.Environment]::SetEnvironmentVariable($matches[1], $matches[2])
        }
    }
}

# Create services directory if missing
if (!(Test-Path $servicesDir)) {
    New-Item -ItemType Directory $servicesDir | Out-Null
}

# Process services
Get-Content $servicesFile | ForEach-Object {

    if ([string]::IsNullOrWhiteSpace($_)) {
        return
    }


    $parts  = $_ -split '\s+'
    $name   = $parts[0]
    $repo   = $parts[1]
    $branch = $null
    if ($parts.Count -ge 3) {
        $branch = $parts[2]
    }
    $target = Join-Path $servicesDir $name


    # Repo exists -> check remote and branch
    if (Test-Path (Join-Path $target ".git")) {
        $needReclone = $false
        Push-Location $target
        $currentUrl = git remote get-url origin 2>$null
        $currentBranch = git rev-parse --abbrev-ref HEAD 2>$null
        Pop-Location
        if ($currentUrl -ne $repo) {
            Write-Host "REMOTE URL mismatch for $name. Re-cloning..."
            $needReclone = $true
        } elseif ($branch -and $currentBranch -ne $branch) {
            Write-Host "BRANCH mismatch for $name. Re-cloning..."
            $needReclone = $true
        }
        if ($needReclone) {
            Remove-Item -Recurse -Force $target
        } else {
            Write-Host "PULLING: $name"
            Push-Location $target
            git pull
            Pop-Location
            return
        }
    }

    # Repo missing -> clone
    if ($env:GIT_TOKEN) {
        $repo = $repo -replace '^https://', "https://$($env:GIT_TOKEN)@"
    }

    Write-Host "CLONING: $name"
    if ($branch) {
        git clone --branch $branch $repo $target
    } else {
        git clone $repo $target
    }
}
