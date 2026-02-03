# PowerShell script for interactive owner configuration setup
# This script prompts the user for owner registration details and updates .env file
# Usage: .\setup-owner.ps1           (interactive mode)
#        .\setup-owner.ps1 -UseDefaults (use defaults without prompts)

param(
    [switch]$UseDefaults
)

# Function to get current value from .env file
function Get-EnvValue {
    param(
        [string]$Key
    )
    
    if (Test-Path ".env") {
        $line = Select-String -Path ".env" -Pattern "^$Key=" -ErrorAction SilentlyContinue
        if ($line) {
            $value = $line.Line -replace "^$Key=", ""
            return $value
        }
    }
    return $null
}

# Function to update .env file
function Update-EnvValue {
    param(
        [string]$Key,
        [string]$Value
    )
    
    $envFile = ".env"
    
    if (Test-Path $envFile) {
        $content = Get-Content $envFile -Raw
        
        if ($content -match "^$Key=") {
            # Update existing key
            $content = $content -replace "^$Key=.*$", "$Key=$Value"
        } else {
            # Add new key if it doesn't exist
            if (-not $content.EndsWith("`n")) {
                $content += "`n"
            }
            $content += "$Key=$Value`n"
        }
        
        Set-Content -Path $envFile -Value $content -Encoding UTF8 -NoNewline
    }
}

# Get current values or use defaults
$currentFirstName = Get-EnvValue "OWNER_FIRST_NAME"
if (-not $currentFirstName) { $currentFirstName = "Admin" }

$currentLastName = Get-EnvValue "OWNER_LAST_NAME"
if (-not $currentLastName) { $currentLastName = "User" }

$currentEmail = Get-EnvValue "OWNER_EMAIL"
if (-not $currentEmail) { $currentEmail = "admin@example.com" }

$currentPassword = Get-EnvValue "OWNER_PASSWORD"
if (-not $currentPassword) { $currentPassword = "Admin@123" }

# Set values based on mode
if ($UseDefaults) {
    $firstName = "Admin"
    $lastName = "User"
    $email = "admin@example.com"
    $password = "Admin@123"
    $isDefaultMode = $true
} else {
    # Prompt for new values
    Write-Host "First Name (current: $currentFirstName): " -NoNewline
    $firstName = Read-Host
    if ([string]::IsNullOrWhiteSpace($firstName)) {
        $firstName = $currentFirstName
    }

    Write-Host "Last Name (current: $currentLastName): " -NoNewline
    $lastName = Read-Host
    if ([string]::IsNullOrWhiteSpace($lastName)) {
        $lastName = $currentLastName
    }

    Write-Host "Email (current: $currentEmail): " -NoNewline
    $email = Read-Host
    if ([string]::IsNullOrWhiteSpace($email)) {
        $email = $currentEmail
    }

    Write-Host "Password (current: $currentPassword): " -NoNewline
    $password = Read-Host
    if ([string]::IsNullOrWhiteSpace($password)) {
        $password = $currentPassword
    }
    $isDefaultMode = $false
}

# Update .env file
Write-Host "`n"
Write-Host "Updating .env file..."

Update-EnvValue "OWNER_FIRST_NAME" $firstName
Update-EnvValue "OWNER_LAST_NAME" $lastName
Update-EnvValue "OWNER_EMAIL" $email
Update-EnvValue "OWNER_PASSWORD" $password

Write-Host "`n========================================================================`n"
if ($isDefaultMode) {
    Write-Host "Owner Configuration Set to Defaults!`n"
} else {
    Write-Host "Owner Configuration Updated Successfully!`n"
}
Write-Host "First Name: $firstName"
Write-Host "Last Name: $lastName"
Write-Host "Email: $email"
Write-Host "Password: $password"
Write-Host "`n========================================================================"
