# ========================================================================
#                    SERENIBASE SETUP SCRIPT (AUTO-YES)
#                    Windows PowerShell Version
# ========================================================================
#
# This script runs the setup with default values but REQUIRES SMTP credentials.
#
# Usage:
#   .\setup-y.ps1 -SmtpHost "your_email_host" -SmtpPort "587" -SmtpUsername "your@email.com" -SmtpPassword "your-app-password" -SmtpFromEmail "your@email.com"
#
# ========================================================================

param(
    [Parameter(Mandatory=$true)]
    [string]$SmtpHost,
    [Parameter(Mandatory=$true)]
    [string]$SmtpPort,
    [Parameter(Mandatory=$true)]
    [string]$SmtpUsername,
    [Parameter(Mandatory=$true)]
    [string]$SmtpPassword,
    [Parameter(Mandatory=$true)]
    [string]$SmtpFromEmail
)

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

& powershell -NoProfile -ExecutionPolicy Bypass -File "$ScriptDir\setup.ps1" -AutoYes -SmtpHost $SmtpHost -SmtpPort $SmtpPort -SmtpUsername $SmtpUsername -SmtpPassword $SmtpPassword -SmtpFromEmail $SmtpFromEmail
