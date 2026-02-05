@echo off
REM ========================================================================
REM                 COMMON FUNCTIONS FOR WINDOWS BATCH
REM                 Shared utilities for setup scripts
REM ========================================================================

REM This file should be called with CALL to include functions
REM Usage: CALL common.bat

REM ========================================================================
REM                      PREREQUISITE CHECKING
REM ========================================================================

:check_prerequisites
    echo Checking prerequisites...
    echo.
    
    set "PREREQ_FAILED=0"
    
    docker --version >nul 2>&1
    if %errorlevel% neq 0 (
        echo [X] Docker is not installed. Please install Docker Desktop first.
        set "PREREQ_FAILED=1"
    ) else (
        echo [OK] Docker is installed
    )
    
    docker compose version >nul 2>&1
    if %errorlevel% neq 0 (
        echo [X] Docker Compose is not installed.
        set "PREREQ_FAILED=1"
    ) else (
        echo [OK] Docker Compose is installed
    )
    
    git --version >nul 2>&1
    if %errorlevel% neq 0 (
        echo [X] Git is not installed. Please install Git first.
        set "PREREQ_FAILED=1"
    ) else (
        echo [OK] Git is installed
    )
    
    if "%PREREQ_FAILED%"=="1" (
        pause
        exit /b 1
    )
    
    echo.
    echo All prerequisites satisfied!
    echo.
    
    exit /b 0

REM ========================================================================
REM                      ENVIRONMENT SETUP
REM ========================================================================

:setup_environment
    set "TEMPLATE_SOURCE=%~1"
    set "TARGET_ENV=%~2"
    
    if "%TEMPLATE_SOURCE%"=="" set "TEMPLATE_SOURCE=%~dp0.env.template"
    if "%TARGET_ENV%"=="" set "TARGET_ENV=.env"
    
    echo Setting up environment...
    
    if not exist "%TARGET_ENV%" (
        REM If .env doesn't exist, create it from template
        copy "%TEMPLATE_SOURCE%" "%TARGET_ENV%" >nul
        echo [OK] Created %TARGET_ENV% with default environment variables
    ) else (
        REM If .env exists, append missing variables
        echo [!] %TARGET_ENV% already exists. Checking for missing variables...
        powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0append-env-vars.ps1" -TargetEnv "%TARGET_ENV%" -TemplateSource "%TEMPLATE_SOURCE%"
    )
    
    exit /b 0

REM ========================================================================
REM                      UPDATE ENVIRONMENT VARIABLE
REM ========================================================================

:update_env_var
    set "VAR_NAME=%~1"
    set "VAR_VALUE=%~2"
    set "ENV_FILE=%~3"
    
    if "%ENV_FILE%"=="" set "ENV_FILE=.env"
    
    powershell -Command "$content = Get-Content '%ENV_FILE%' -Raw; if ($content -match '(?m)^%VAR_NAME%=') { $content = $content -replace '(?m)^%VAR_NAME%=.*', '%VAR_NAME%=%VAR_VALUE%' } else { $content += \"`n%VAR_NAME%=%VAR_VALUE%\" }; Set-Content '%ENV_FILE%' -Value $content -NoNewline"
    
    exit /b 0

REM ========================================================================
REM                      CLONE REPOSITORIES
REM ========================================================================

:clone_repositories
    echo.
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
    
    exit /b 0

REM ========================================================================
REM                      START DOCKER SERVICES
REM ========================================================================

:start_docker_services
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
    
    exit /b 0

REM ========================================================================
REM                      COMPLETION MESSAGE
REM ========================================================================

:print_completion
    set "PUBLIC_HOST=%~1"
    set "OWNER_EMAIL=%~2"
    set "OWNER_PASSWORD=%~3"
    
    if "%PUBLIC_HOST%"=="" set "PUBLIC_HOST=localhost"
    if "%OWNER_EMAIL%"=="" set "OWNER_EMAIL=admin@example.com"
    if "%OWNER_PASSWORD%"=="" set "OWNER_PASSWORD=Admin@123"
    
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
    echo   Email:    %OWNER_EMAIL%
    echo   Password: %OWNER_PASSWORD%
    echo.
    echo WARNING: Remember to change default passwords in production!
    echo.
    echo Useful commands:
    echo   make logs      - View service logs
    echo   make down-all  - Stop all services
    echo   make clean     - Remove all data
    echo.
    
    exit /b 0
