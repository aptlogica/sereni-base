#!/usr/bin/env pwsh

# Test script to demonstrate the new UI format

Clear-Host

Write-Host ""
Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host "                     NETWORK CONFIGURATION                              " -ForegroundColor Cyan
Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "💡 Examples: localhost (local dev), 192.168.1.100 (LAN access), yourdomain.com (production)" -ForegroundColor Cyan
Write-Host ""
Write-Host "Custom IP/domain (for LAN or production access [localhost]: " -NoNewline
Write-Host ""
Write-Host ""

Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host "                  OWNER REGISTRATION CONFIGURATION                      " -ForegroundColor Cyan
Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Enter owner registration details (press Enter to use defaults):"
Write-Host ""
Write-Host "First Name [Admin]: "
Write-Host ""
Write-Host "Last Name [User]: "
Write-Host ""
Write-Host "Email [admin@example.com]: "
Write-Host ""
Write-Host "Password [Admin@123]: "
Write-Host ""
Write-Host ""

Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host "                    SECURITY CONFIGURATION                              " -ForegroundColor Cyan
Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "JWT secret is used to sign authentication tokens."
Write-Host "⚠️  Use a strong random string (at least 32 characters) for production!"
Write-Host ""
Write-Host "JWT Secret Key [change-this-to-a-secure-random-string-min32chars]: "
Write-Host ""
Write-Host ""

Write-Host "✅ Setup UI Preview Complete!" -ForegroundColor Green
Write-Host ""
