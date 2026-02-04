#!/usr/bin/env pwsh

# Global flag to track cancellation
$script:SetupCancelled = $false

# Trap handler for Ctrl+C
trap {
    $script:SetupCancelled = $true
    Write-Host ""
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host "  Setup cancelled by user." -ForegroundColor Yellow
    Write-Host "========================================" -ForegroundColor Yellow
    Write-Host ""
    exit 130
}

# Set error action
$ErrorActionPreference = "Stop"

function Read-UserInput {
    param([string]$Prompt, [string]$Default = "")
    
    if ($script:SetupCancelled) {
        exit 130
    }
    
    try {
        if ($Default) {
            $displayPrompt = "$Prompt [$Default]" + ": "
        } else {
            $displayPrompt = "$Prompt" + ": "
        }
        
        try {
            $value = Read-Host $displayPrompt
        }
        catch {
            throw
        }
        
        if ($script:SetupCancelled) {
            exit 130
        }
        
        if ([string]::IsNullOrWhiteSpace($value)) {
            return $Default
        }
        
        return $value.Trim()
    }
    catch {
        $script:SetupCancelled = $true
        Write-Host ""
        Write-Host ""
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host "  Setup cancelled by user." -ForegroundColor Yellow
        Write-Host "========================================" -ForegroundColor Yellow
        Write-Host ""
        exit 130
    }
}

Write-Host "Test Ctrl+C handling" -ForegroundColor Cyan
Write-Host "Press Ctrl+C to cancel" -ForegroundColor Yellow
Write-Host ""

$name = Read-UserInput -Prompt "Your name" -Default "Test User"
$email = Read-UserInput -Prompt "Your email" -Default "test@example.com"

Write-Host ""
Write-Host "Thank you, $name ($email)!" -ForegroundColor Green
