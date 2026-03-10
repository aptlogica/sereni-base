param(
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]]$Args
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$Target = Join-Path $ScriptDir "build\scripts\setup.ps1"

if (-not (Test-Path $Target)) {
    Write-Host "Setup script not found at $Target" -ForegroundColor Red
    exit 1
}

& $Target @Args
exit $LASTEXITCODE
