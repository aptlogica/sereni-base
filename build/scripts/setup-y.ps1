# ========================================================================
#                    SERENIBASE SETUP SCRIPT (AUTO-YES MODE)
#                    Windows PowerShell Version
# ========================================================================
#
# Full automated setup with default values
# Supports all parameters that setup.ps1 supports
#
# Priority for environment variables:
#   1. Script parameters (highest priority)
#   2. Existing values from .env file (protected - never overridden)
#   3. Default variable values (lowest priority)
#
# Usage Examples:
#   .\setup-y.ps1
#   .\setup-y.ps1 -PublicHost "192.168.1.100"
#   .\setup-y.ps1 -DatabasePort "5433" -SmtpHost "smtp.gmail.com"
#
# ========================================================================

param(
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]]$Parameters
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Pass all arguments to setup.ps1 with -AutoYes prepended
& powershell -NoProfile -ExecutionPolicy Bypass -File "$ScriptDir\setup.ps1" -AutoYes @Parameters
