param(
    [string]$TargetEnv = ".env",
    [string]$TemplateSource = ".env.template"
)

# Read existing .env content if it exists
if (Test-Path $TargetEnv) {
    $existing = Get-Content $TargetEnv -Raw
} else {
    $existing = ""
}

# Read template
$template = Get-Content $TemplateSource

# Find missing variables
$missing = @()
foreach ($line in $template) {
    if ($line -match '^([A-Z_]+)=') {
        $varName = $matches[1]
        if ($existing -notmatch "(?m)^$varName=") {
            $missing += $line
        }
    }
}

# Append missing variables if any
if ($missing.Count -gt 0) {
    Add-Content $TargetEnv "`n# Added by setup script on $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
    Add-Content $TargetEnv $missing
    Write-Host "[OK] Added $($missing.Count) missing variable(s) to $TargetEnv"
} else {
    Write-Host "[OK] All variables already exist in $TargetEnv"
}
