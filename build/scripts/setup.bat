@echo off
REM ========================================================================
REM                    SERENIBASE SETUP SCRIPT
REM                    Windows Batch Version
REM ========================================================================

setlocal enabledelayedexpansion

REM Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"
REM Navigate to project root (two levels up from build/scripts/)
cd /d "%SCRIPT_DIR%..\.."

echo.
echo ========================================================================
echo                     SERENIBASE SETUP WIZARD
echo ========================================================================
echo.

REM Check prerequisites
echo Checking prerequisites...
echo.

docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [X] Docker is not installed. Please install Docker Desktop first.
    pause
    exit /b 1
)
echo [OK] Docker is installed

docker compose version >nul 2>&1
if %errorlevel% neq 0 (
    echo [X] Docker Compose is not installed.
    pause
    exit /b 1
)
echo [OK] Docker Compose is installed

git --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [X] Git is not installed. Please install Git first.
    pause
    exit /b 1
)
echo [OK] Git is installed

echo.
echo All prerequisites satisfied!
echo.

REM Setup environment
echo Setting up environment...
if not exist ".env" (
    if exist "build\config\.env.example" (
        copy "build\config\.env.example" ".env" >nul
        echo [OK] Created .env from build\config\.env.example
    ) else (
        echo [X] build\config\.env.example not found!
        pause
        exit /b 1
    )
) else (
    echo [!] .env already exists. Skipping creation.
)

echo.
echo ========================================================================
echo                      NETWORK CONFIGURATION
echo ========================================================================
echo.
set /p PUBLIC_HOST="Enter IP/domain [localhost]: "
if "%PUBLIC_HOST%"=="" set PUBLIC_HOST=localhost

REM Update .env file with PUBLIC_HOST
powershell -Command "$content = Get-Content '.env' -Raw; if ($content -match '(?m)^PUBLIC_HOST=') { $content = $content -replace '(?m)^PUBLIC_HOST=.*', 'PUBLIC_HOST=%PUBLIC_HOST%' } else { $content += \"`nPUBLIC_HOST=%PUBLIC_HOST%\" }; Set-Content '.env' -Value $content -NoNewline"
echo [OK] Configured PUBLIC_HOST=%PUBLIC_HOST%

echo.
echo ========================================================================
echo                   OWNER REGISTRATION CONFIGURATION
echo ========================================================================
echo.
echo Enter owner registration details (press Enter to use defaults):
echo.

set /p OWNER_FIRST_NAME="First Name [Admin]: "
if "%OWNER_FIRST_NAME%"=="" set OWNER_FIRST_NAME=Admin

set /p OWNER_LAST_NAME="Last Name [User]: "
if "%OWNER_LAST_NAME%"=="" set OWNER_LAST_NAME=User

set /p OWNER_EMAIL="Email [admin@example.com]: "
if "%OWNER_EMAIL%"=="" set OWNER_EMAIL=admin@example.com

set /p OWNER_PASSWORD="Password [Admin@123]: "
if "%OWNER_PASSWORD%"=="" set OWNER_PASSWORD=Admin@123

REM Update .env file with owner configuration (add if not exists, replace if exists)
powershell -Command "$content = Get-Content '.env' -Raw; if ($content -match '(?m)^OWNER_FIRST_NAME=') { $content = $content -replace '(?m)^OWNER_FIRST_NAME=.*', 'OWNER_FIRST_NAME=%OWNER_FIRST_NAME%' } else { $content += \"`nOWNER_FIRST_NAME=%OWNER_FIRST_NAME%\" }; Set-Content '.env' -Value $content -NoNewline"
powershell -Command "$content = Get-Content '.env' -Raw; if ($content -match '(?m)^OWNER_LAST_NAME=') { $content = $content -replace '(?m)^OWNER_LAST_NAME=.*', 'OWNER_LAST_NAME=%OWNER_LAST_NAME%' } else { $content += \"`nOWNER_LAST_NAME=%OWNER_LAST_NAME%\" }; Set-Content '.env' -Value $content -NoNewline"
powershell -Command "$content = Get-Content '.env' -Raw; if ($content -match '(?m)^OWNER_EMAIL=') { $content = $content -replace '(?m)^OWNER_EMAIL=.*', 'OWNER_EMAIL=%OWNER_EMAIL%' } else { $content += \"`nOWNER_EMAIL=%OWNER_EMAIL%\" }; Set-Content '.env' -Value $content -NoNewline"
powershell -Command "$content = Get-Content '.env' -Raw; if ($content -match '(?m)^OWNER_PASSWORD=') { $content = $content -replace '(?m)^OWNER_PASSWORD=.*', 'OWNER_PASSWORD=%OWNER_PASSWORD%' } else { $content += \"`nOWNER_PASSWORD=%OWNER_PASSWORD%\" }; Set-Content '.env' -Value $content -NoNewline"

echo [OK] Owner configuration set
echo ========================================================================
echo                      CLONING REPOSITORIES
echo ========================================================================
echo.

REM Clone services using PowerShell scripts
if exist "build\scripts\clone-services.ps1" (
    echo Cloning microservices...
    powershell -NoProfile -ExecutionPolicy Bypass -File build\scripts\clone-services.ps1
    echo [OK] Cloned microservices
)

if exist "build\scripts\clone-go-postgres-rest.ps1" (
    echo Cloning go-postgres-rest...
    powershell -NoProfile -ExecutionPolicy Bypass -File build\scripts\clone-go-postgres-rest.ps1
    echo [OK] Cloned go-postgres-rest
)

echo.
echo ========================================================================
echo                      STARTING SERVICES
echo ========================================================================
echo.

docker compose -f docker-compose.all.yaml up --build -d

echo.
echo Waiting for services to start...
timeout /t 10 /nobreak >nul

docker compose -f docker-compose.all.yaml ps

echo.
echo ========================================================================
echo                      SETUP COMPLETE!
echo ========================================================================
echo.
echo Access your application at:
echo   Frontend:  http://%PUBLIC_HOST%:5050
echo   Backend:   http://%PUBLIC_HOST%:8080
echo   MinIO:     http://%PUBLIC_HOST%:9001
echo.
echo Default admin credentials:
echo   Email:    admin@example.com
echo   Password: Admin@123
echo.
echo WARNING: Remember to change default passwords in production!
echo.
echo Useful commands:
echo   make logs      - View service logs
echo   make down-all  - Stop all services
echo   make clean     - Remove all data
echo.
pause
