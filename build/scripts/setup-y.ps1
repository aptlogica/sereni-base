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
#   2. Existing values from .env file (if exists)
#   3. Default variable values (lowest priority)
#
# Usage (with SMTP credentials - recommended):
#   .\setup-y.ps1 -SmtpHost "your_email_host" -SmtpPort "587" -SmtpUsername "your@email.com" -SmtpPassword "your-app-password" -SmtpFromEmail "your@email.com"
#
# Usage (with any environment variables):
#   .\setup-y.ps1 -SmtpHost "..." -DatabaseHost "..." -PublicHost "..."
#
# Usage (use all defaults from .env or defaults):
#   .\setup-y.ps1
#
# ========================================================================

param(
    # SMTP Configuration (optional)
    [string]$SmtpHost = "",
    [string]$SmtpPort = "",
    [string]$SmtpUsername = "",
    [string]$SmtpPassword = "",
    [string]$SmtpFromEmail = "",
    # Additional environment variables can be passed as parameters
    # Example: .\setup-y.ps1 -PublicHost "myhost.com" -DatabaseHost "db.example.com"
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]]$AdditionalParameters
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

# Build argument list for setup.ps1
$argumentList = @("-AutoYes")

if ($SmtpHost) { $argumentList += "-SmtpHost"; $argumentList += $SmtpHost }
if ($SmtpPort) { $argumentList += "-SmtpPort"; $argumentList += $SmtpPort }
if ($SmtpUsername) { $argumentList += "-SmtpUsername"; $argumentList += $SmtpUsername }
if ($SmtpPassword) { $argumentList += "-SmtpPassword"; $argumentList += $SmtpPassword }
if ($SmtpFromEmail) { $argumentList += "-SmtpFromEmail"; $argumentList += $SmtpFromEmail }

# Pass additional parameters
if ($AdditionalParameters) {
    $argumentList += $AdditionalParameters
}

# Call setup.ps1 with all arguments
& powershell -NoProfile -ExecutionPolicy Bypass -File "$ScriptDir\setup.ps1" @argumentList
