# ========================================================================
#                    SERENIBASE SETUP SCRIPT (AUTO-YES)
#                    Windows PowerShell Version
# ========================================================================
#
# This script runs the setup with all default values (no prompts).
#
# ========================================================================

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

& powershell -NoProfile -ExecutionPolicy Bypass -File "$ScriptDir\setup.ps1" -AutoYes
