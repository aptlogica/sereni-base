# ========================================================================
#                    SERENIBASE SETUP SCRIPT (AUTO-YES)
#                    Windows PowerShell Version
# ========================================================================
#
# This script runs the setup with default values but REQUIRES SMTP credentials.
#
# Usage:
#   .\setup-y.ps1 -SmtpHost "smtp.gmail.com" -SmtpPort "587" -SmtpUsername "your@email.com" -SmtpPassword "your-app-password"
#
# ========================================================================

param(
    [string]$SmtpHost = "smtp.gmail.com",
    [string]$SmtpPort = "587",
    [Parameter(Mandatory=$true)]
    [string]$SmtpUsername,
    [Parameter(Mandatory=$true)]
    [string]$SmtpPassword,
    [string]$SmtpFromEmail = ""
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

if ([string]::IsNullOrWhiteSpace($SmtpFromEmail)) {
    $SmtpFromEmail = $SmtpUsername
}

& powershell -NoProfile -ExecutionPolicy Bypass -File "$ScriptDir\setup.ps1" -AutoYes -SmtpHost $SmtpHost -SmtpPort $SmtpPort -SmtpUsername $SmtpUsername -SmtpPassword $SmtpPassword -SmtpFromEmail $SmtpFromEmail
