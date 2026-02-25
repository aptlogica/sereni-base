param(
    [string]$TargetEnv = ".env",
    [string]$TemplateSource = ".env.template"
)

# Write UTF-8 text without BOM to keep .env portable across shells and Docker.
function Set-TextFileNoBom {
    param(
        [string]$Path,
        [string]$Content
    )
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

# Function to extract variable value from .env content
function Get-EnvValue {
    param([string]$Content, [string]$VarName)
    
    if ($Content -match "(?m)^$VarName=(.*)$") {
        return $matches[1]
    }
    return $null
}

# Read existing .env content if it exists
$existingValues = @{}
if (Test-Path $TargetEnv) {
    $existingContent = Get-Content $TargetEnv -Raw
    $existingLines = Get-Content $TargetEnv
    
    # Extract all existing variable values
    foreach ($line in $existingLines) {
        if ($line -match '^([A-Z_][A-Z0-9_]*)=(.*)$') {
            $varName = $matches[1]
            $varValue = $matches[2]
            $existingValues[$varName] = $varValue
        }
    }
    
    Write-Host "[INFO] Found $($existingValues.Count) existing variables in $TargetEnv"
} else {
    $existingContent = ""
}

# Read template
$templateLines = Get-Content $TemplateSource

# Build new .env with proper formatting, preserving existing values
$newEnvContent = @()
$preservedCount = 0
$addedCount = 0

foreach ($line in $templateLines) {
    if ($line -match '^([A-Z_][A-Z0-9_]*)=(.*)$') {
        $varName = $matches[1]
        $templateValue = $matches[2]
        
        # If variable exists in old .env, use that value
        if ($existingValues.ContainsKey($varName)) {
            $newEnvContent += "$varName=$($existingValues[$varName])"
            $preservedCount++
        } else {
            # Use template value for new variables
            $newEnvContent += $line
            $addedCount++
        }
    } else {
        # Keep comments and formatting as-is
        $newEnvContent += $line
    }
}

# Check for any variables in old .env that are not in template (custom variables)
$customVars = @()
foreach ($varName in $existingValues.Keys) {
    $found = $false
    foreach ($line in $templateLines) {
        if ($line -match "^$varName=") {
            $found = $true
            break
        }
    }
    if (-not $found) {
        $customVars += "$varName=$($existingValues[$varName])"
    }
}

# Append custom variables at the end if any
if ($customVars.Count -gt 0) {
    $newEnvContent += ""
    $newEnvContent += "# Custom variables (not in template)"
    $newEnvContent += ""
    $newEnvContent += $customVars
}

# Write the new content
Set-TextFileNoBom -Path $TargetEnv -Content (($newEnvContent -join "`n") + "`n")

# Report
Write-Host ""
Write-Host "[OK] Updated $TargetEnv with proper formatting:"
Write-Host "     - Preserved: $preservedCount existing value(s)"
Write-Host "     - Added: $addedCount new variable(s)"
if ($customVars.Count -gt 0) {
    Write-Host "     - Retained: $($customVars.Count) custom variable(s)"
}
Write-Host ""
