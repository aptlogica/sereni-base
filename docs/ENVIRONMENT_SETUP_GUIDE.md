# 📘 Environment Configuration Guide

## Quick Navigation
- [Overview](#overview)
- [Variable Categories](#variable-categories)
- [Interactive Setup](#interactive-setup)
- [Required Variables](#required-variables)
- [Optional Variables with Defaults](#optional-variables-with-defaults)
- [Configuration by Scenario](#configuration-by-scenario)

---

## Overview

SereniBase uses environment variables for configuration. This guide explains:
- **Which variables MUST be set** (required)
- **Which variables have defaults** (optional)
- **How to use the interactive setup script**

### Three Ways to Configure

1. **Interactive Setup (Recommended)** - Guided prompts with defaults
2. **Copy Template** - Edit `.env.example` manually
3. **Manual Creation** - Create `.env` from scratch

---

## Variable Categories

### 🔴 REQUIRED (Must Configure)

These MUST be set before deployment:

| Variable | Purpose | Example |
|----------|---------|---------|
| `PUBLIC_HOST` | Your domain/IP address | `example.com` or `192.168.1.100` |
| `OWNER_EMAIL` | Admin account email | `admin@company.com` |
| `OWNER_PASSWORD` | Admin account password | `SecurePass123!` |
| `AUTH_JWT_SECRET` | JWT signing secret (32+ chars) | `my-super-secret-key-min-32-characters` |

### 🟡 SECURITY-SENSITIVE (Change in Production)

These have defaults but MUST be changed for production:

| Variable | Insecure Default | Production Requirement |
|----------|------------------|------------------------|
| `DATABASE_PASSWORD` | `postgres` | Strong password |
| `TEMPORARY_USER_PASSWORD` | Random string | Change to secure value |

### 🟢 OPTIONAL (Good Defaults)

These have sensible defaults and only need changing for specific requirements:

<details>
<summary><b>Click to expand full list</b></summary>

| Category | Variable | Default | Change When |
|----------|----------|---------|-------------|
| **Server** | `SERVER_HOST` | `0.0.0.0` | Restrict to specific interface |
| **Server** | `SERVER_PORT` | `8080` | Port conflict |
| **Server** | `SERVER_ENV` | `dev` | Production deployment |
| **Database** | `DATABASE_HOST` | `localhost` | Using external DB |
| **Database** | `DATABASE_PORT` | `5432` | Non-standard PostgreSQL port |
| **Database** | `DATABASE_USER` | `postgres` | Security hardening |
| **Database** | `DATABASE_NAME` | `serenibase` | Custom database name |
| **Database** | `DATABASE_MAX_OPEN_CONNS` | `25` | Performance tuning |
| **Storage** | `STORAGE_DRIVER` | `local` | Using cloud storage |
| **Logging** | `LOG_LEVEL` | `info` | More/less verbosity |
| **Assets** | `ASSET_MAX_SIZE` | `5242880` (5MB) | Larger uploads needed |

</details>

### 🔵 CONDITIONALLY REQUIRED

Required only when using specific features:

| Feature | Variables Required | Example |
|---------|-------------------|---------|
| **Email** | `EMAIL_SMTP_*` | Password reset, notifications |
| **AWS S3** | `STORAGE_AWS_*` | When `STORAGE_DRIVER=aws` |
| **RustFS** | `STORAGE_RustFS_*` | When `STORAGE_DRIVER=RustFS` |
| **Antivirus** | `ANTIVIRUS_URL`, `ANTIVIRUS_CLAMAV_*` | File scanning enabled |

---

## Interactive Setup

### For Windows (PowerShell)

```powershell
# Run the interactive setup
.\setup-interactive.ps1

# Skip Docker check
.\setup-interactive.ps1 -SkipDocker

# Show help
.\setup-interactive.ps1 -Help
```

### For Linux/macOS (Bash)

```bash
# Make script executable
chmod +x setup-interactive.sh

# Run the interactive setup
./setup-interactive.sh

# Skip Docker check
./setup-interactive.sh --skip-docker

# Show help
./setup-interactive.sh --help
```

### What the Script Does

1. **Detects your system** - Shows OS and local IP
2. **Checks Docker** - Verifies Docker installation
3. **Prompts for values** - Shows defaults in brackets:
   ```
   IP Address or Domain [localhost]: 
   ```
4. **Validates input** - Email format, password confirmation
5. **Generates .env** - Creates complete configuration file
6. **Shows next steps** - How to start the application

### Example Interactive Session

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                         🚀 SERENIBASE SETUP                                   ║
║                     Interactive Configuration Wizard                          ║
╚══════════════════════════════════════════════════════════════════════════════╝

📡 Detected System Information:
   OS: Windows 10
   Local IP: 192.168.1.100

🐳 Checking Docker...
   ✓ Docker is available: Docker version 24.0.0

═══════════════════════════════════════════════════════════════
                    📋 CONFIGURATION                            
═══════════════════════════════════════════════════════════════

Press Enter to accept default values shown in [brackets]

┌─────────────────────────────────────────────────────────┐
│           🌐 NETWORK CONFIGURATION                      │
└─────────────────────────────────────────────────────────┘

This is how users will access your application.
Examples:
  - localhost (for testing on this machine)
  - 192.168.1.100 (for LAN access)
  - yourdomain.com (for production)

IP Address or Domain [localhost]: 192.168.1.100

┌─────────────────────────────────────────────────────────┐
│           👤 ADMIN ACCOUNT SETUP                        │
└─────────────────────────────────────────────────────────┘

Create the first administrator account.

Admin First Name [Admin]: John
Admin Last Name [User]: Doe
Admin Email [your-admin-email@example.com]: john.doe@company.com
Admin Password [your-strong-password]: MySecurePassword123!
Confirm Password: MySecurePassword123!

... (continues with more prompts)
```

---

## Required Variables

### PUBLIC_HOST
```bash
PUBLIC_HOST=localhost
```

**Purpose:** External access URL for your application

**When to set:**
- `localhost` - Testing on your machine
- `192.168.1.100` - LAN access
- `myapp.example.com` - Production domain

**Used By:**
- Password reset emails
- CORS configuration
- Frontend API calls
- Service communication

### OWNER_EMAIL
```bash
OWNER_EMAIL=your-admin-email@example.com
```

**Purpose:** First admin user's login email

**Requirements:**
- Valid email format
- Must be unique
- Used for login and notifications

### OWNER_PASSWORD
```bash
OWNER_PASSWORD=your-strong-password
```

**Purpose:** First admin user's password

**Requirements:**
- Minimum 8 characters (recommended 16+)
- Include uppercase, lowercase, numbers, symbols
- ⚠️ **Change immediately for production!**

### AUTH_JWT_SECRET
```bash
AUTH_JWT_SECRET=my-super-secret-jwt-key-minimum-32-characters-long
```

**Purpose:** Secret key for JWT token signing

**Requirements:**
- **Minimum 32 characters**
- Random and unpredictable
- Never commit to version control
- Change between environments

**Generate Secure Secret:**

```bash
# Linux/macOS
openssl rand -base64 32

# PowerShell
[Convert]::ToBase64String((1..32 | ForEach-Object { Get-Random -Maximum 256 }))

# Online (use with caution)
# Visit: https://generate-secret.now.sh/32
```

---

## Optional Variables with Defaults

### Application Defaults vs Environment Overrides

SereniBase has **built-in defaults** for all optional variables. You only need to set them in `.env` if you want to override the defaults.

#### How It Works

```go
// In internal/config/config.go

// Set application default
viper.SetDefault("server.port", "8080")

// Bind to environment variable (optional override)
viper.BindEnv("server.port", "SERVER_PORT")

// Result:
// - If SERVER_PORT is set in .env → uses that value
// - If SERVER_PORT is NOT set → uses "8080"
```

### Complete Defaults Reference

<details>
<summary><b>Server Configuration</b></summary>

```bash
# All have built-in defaults - override only if needed

SERVER_HOST=0.0.0.0           # Listen on all interfaces
SERVER_PORT=8080              # HTTP port
SERVER_READ_TIMEOUT=30        # Request read timeout (seconds)
SERVER_WRITE_TIMEOUT=30       # Response write timeout (seconds)
SERVER_ENV=dev                # Environment: dev, staging, production
SERVER_SCHEME=http            # Protocol: http or https
```

**Application Defaults (from code):**
- `SERVER_HOST`: `0.0.0.0`
- `SERVER_PORT`: `8080`
- `SERVER_READ_TIMEOUT`: `30`
- `SERVER_WRITE_TIMEOUT`: `30`
- `SERVER_SCHEME`: `http`
- `SERVER_ENV`: `dev`

</details>

<details>
<summary><b>Database Configuration</b></summary>

```bash
# Override only if using external database

DATABASE_HOST=localhost       # PostgreSQL host
DATABASE_PORT=5432           # PostgreSQL port
DATABASE_USER=postgres       # Database username
DATABASE_PASSWORD=postgres   # ⚠️ Change in production!
DATABASE_NAME=serenibase     # Database name
DATABASE_SSL_MODE=disable    # SSL mode: disable, require, verify-ca, verify-full
DATABASE_MAX_OPEN_CONNS=25   # Connection pool size
DATABASE_MAX_IDLE_CONNS=5    # Idle connections
DATABASE_CONN_MAX_LIFETIME=1h # Connection lifetime
```

**Application Defaults (from code):**
- `DATABASE_HOST`: `localhost`
- `DATABASE_PORT`: `5432`
- `DATABASE_USER`: `postgres`
- `DATABASE_PASSWORD`: `postgres`
- `DATABASE_NAME`: `serenibase`
- `DATABASE_SSL_MODE`: `disable`
- `DATABASE_MAX_OPEN_CONNS`: `25`
- `DATABASE_MAX_IDLE_CONNS`: `5`
- `DATABASE_DRIVER`: `postgres`

**Docker Override:**
```bash
DATABASE_HOST=postgres  # Use Docker service name
```

</details>

<details>
<summary><b>Logging Configuration</b></summary>

```bash
# Good defaults - change only for specific needs

LOG_LEVEL=info              # Verbosity: debug, info, warn, error
LOG_FILE=app.log            # Log file path
LOG_MAX_SIZE=50             # Max file size (MB)
LOG_MAX_BACKUPS=10          # Number of old files to keep
LOG_MAX_AGE=30              # Max age of log files (days)
LOG_COMPRESS=true           # Compress rotated logs
```

**Application Defaults (from code):**
- `LOG_LEVEL`: `info`
- `LOG_FILE`: `app.log`
- `LOG_MAX_SIZE`: `50`
- `LOG_MAX_BACKUPS`: `10`
- `LOG_MAX_AGE`: `30`
- `LOG_COMPRESS`: `true`

</details>

<details>
<summary><b>Storage Configuration</b></summary>

```bash
# Choose storage backend

STORAGE_DRIVER=local        # Options: local, RustFS, aws
STORAGE_DEV_PATH=./uploads  # Local storage path

# RustFS Configuration (if STORAGE_DRIVER=RustFS)
STORAGE_RustFS_ENDPOINT=RustFS:9000
STORAGE_RustFS_ACCESS_KEY=RustFSadmin
STORAGE_RustFS_SECRET_KEY=RustFSadmin
STORAGE_RustFS_BUCKET=serenibase
STORAGE_RustFS_USE_SSL=false

# AWS S3 Configuration (if STORAGE_DRIVER=aws)
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=my-bucket
STORAGE_AWS_ACCESS_KEY=your-access-key
STORAGE_AWS_SECRET_KEY=your-secret-key
```

**Application Defaults (from code):**
- `STORAGE_DRIVER`: `local`
- `STORAGE_DEV_PATH`: `./assets`
- All RustFS/AWS settings have placeholder defaults

</details>

<details>
<summary><b>CORS Configuration</b></summary>

```bash
# Auto-configured based on PUBLIC_HOST

CORS_ALLOWED_ORIGINS=*                    # ⚠️ Restrict in production!
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH
CORS_ALLOWED_HEADERS=Content-Type,Authorization,schema,workspace,base
CORS_ALLOW_CREDENTIALS=true
```

**Application Defaults (from code):**
- `CORS_ALLOWED_ORIGINS`: `*` (allow all - change for production!)
- `CORS_ALLOWED_METHODS`: Standard HTTP methods
- `CORS_ALLOWED_HEADERS`: Standard + custom headers
- `CORS_ALLOW_CREDENTIALS`: `true`

**Production Override:**
```bash
CORS_ALLOWED_ORIGINS=https://myapp.com,https://www.myapp.com
```

</details>

---

## Configuration by Scenario

### Scenario 1: Quick Local Testing

**Goal:** Get SereniBase running on your machine ASAP

**Minimum .env:**
```bash
# Just the essentials
PUBLIC_HOST=localhost
OWNER_EMAIL=test@test.com
OWNER_PASSWORD=Test123!
AUTH_JWT_SECRET=test-secret-key-at-least-32-characters-long-please

# Everything else uses defaults
```

**Start:**
```bash
docker-compose up -d
```

**Access:**
- Application: http://localhost:8080
- Frontend: http://localhost:5050

---

### Scenario 2: LAN Development

**Goal:** Share with team on local network

**Required Changes:**
```bash
PUBLIC_HOST=192.168.1.100  # Your machine's IP

OWNER_EMAIL=admin@company.local
OWNER_PASSWORD=DevPassword123!
AUTH_JWT_SECRET=dev-environment-secret-key-min-32-chars

# Docker database
DATABASE_HOST=postgres
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
```

**Share with Team:**
- Application: http://192.168.1.100:8080
- Frontend: http://192.168.1.100:5050

---

### Scenario 3: Production Deployment

**Goal:** Secure, scalable production setup

**Full Production .env:**
```bash
# ============================================
# PRODUCTION CONFIGURATION
# ============================================

# Network
PUBLIC_HOST=myapp.example.com

# Server
SERVER_ENV=production
SERVER_SCHEME=https

# Database (external)
DATABASE_HOST=prod-db.internal
DATABASE_PORT=5432
DATABASE_USER=serenibase_prod
DATABASE_PASSWORD=<STRONG_RANDOM_PASSWORD>
DATABASE_NAME=serenibase_prod
DATABASE_SSL_MODE=require
DATABASE_MAX_OPEN_CONNS=100
DATABASE_MAX_IDLE_CONNS=10

# Security
AUTH_JWT_SECRET=<STRONG_64_CHAR_RANDOM_STRING>

# Admin Account
OWNER_EMAIL=admin@mycompany.com
OWNER_PASSWORD=<STRONG_RANDOM_PASSWORD>
TEMPORARY_USER_PASSWORD=<STRONG_RANDOM_PASSWORD>

# Email (required for production)
EMAIL_SMTP_HOST=smtp.sendgrid.net
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=apikey
EMAIL_SMTP_PASSWORD=<SENDGRID_API_KEY>
EMAIL_FROM_EMAIL=noreply@myapp.example.com

# Storage (AWS S3)
STORAGE_DRIVER=aws
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=myapp-prod-storage
STORAGE_AWS_ACCESS_KEY=<AWS_ACCESS_KEY>
STORAGE_AWS_SECRET_KEY=<AWS_SECRET_KEY>

# CORS (restrict origins)
CORS_ALLOWED_ORIGINS=https://myapp.example.com,https://www.myapp.example.com

# Logging
LOG_LEVEL=warn
LOG_MAX_SIZE=100
LOG_MAX_BACKUPS=30
LOG_MAX_AGE=90
```

**Production Checklist:**

- [ ] Strong passwords (16+ characters, random)
- [ ] JWT secret (64+ characters, random)
- [ ] Email configured and tested
- [ ] CORS restricted to your domains
- [ ] DATABASE_SSL_MODE=require
- [ ] External database with backups
- [ ] Cloud storage (S3/RustFS)
- [ ] HTTPS enabled (SERVER_SCHEME=https)
- [ ] LOG_LEVEL=warn or error
- [ ] Firewall rules configured
- [ ] Secrets in secure vault (not in .env committed to git)

---

### Scenario 4: Docker Compose All Services

**Goal:** Run everything with Docker (backend, frontend, services, databases)

**Docker-Optimized .env:**
```bash
PUBLIC_HOST=localhost

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Docker Services (use service names)
DATABASE_HOST=postgres
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=serenibase

AUTH_URL=http://jwt-provider:8081
EMAIL_URL=http://email-service:8082/api/v1/email
STORAGE_URL=http://sereni-storage-provider:8083/api/v1
ANTIVIRUS_URL=http://antivirus-service:8084

# Storage (RustFS in Docker)
STORAGE_DRIVER=RustFS
STORAGE_RustFS_ENDPOINT=RustFS:9000
STORAGE_RustFS_ACCESS_KEY=RustFSadmin
STORAGE_RustFS_SECRET_KEY=RustFSadmin
STORAGE_RustFS_BUCKET=serenibase
STORAGE_RustFS_USE_SSL=false

# Email (configure your SMTP)
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=your_email@gmail.com
EMAIL_SMTP_PASSWORD=your_app_password
EMAIL_FROM_EMAIL=your_email@gmail.com

# Security
OWNER_EMAIL=your-admin-email@example.com
OWNER_PASSWORD=your-strong-password
AUTH_JWT_SECRET=docker-dev-secret-key-min-32-characters
```

**Start Everything:**
```bash
docker-compose -f docker-compose.all.yaml up -d
```

---

## Finding Application Defaults

### Method 1: Check Source Code

All defaults are defined in `internal/config/config.go`:

```go
// Look for viper.SetDefault() calls
viper.SetDefault("server.port", "8080")
viper.SetDefault("database.host", "localhost")
viper.SetDefault("log.level", "info")
```

### Method 2: Check This Documentation

See the [Complete Defaults Reference](#complete-defaults-reference) section above.

### Method 3: Run Without .env

Start the application without a `.env` file and check the startup logs. It will show which values are being used (defaults).

---

## Common Questions

### Q: Do I need to set ALL variables in .env?

**A:** No! Only set variables you want to override. SereniBase has sensible defaults for everything except:
- `PUBLIC_HOST`
- `OWNER_EMAIL`
- `OWNER_PASSWORD`
- `AUTH_JWT_SECRET`

### Q: What happens if I don't set a variable?

**A:** The application uses the built-in default from `internal/config/config.go`.

### Q: How do I know what the default is?

**A:** Check:
1. This documentation (defaults are listed)
2. Source code `internal/config/config.go`
3. Comments in `build/config/.env.example`

### Q: Can I change defaults after deployment?

**A:** Yes! Edit `.env` and restart the application:
```bash
docker-compose restart
```

### Q: Should I commit .env to git?

**A:** **NO!** Never commit `.env` with secrets. Instead:
- Commit `.env.example` as a template
- Add `.env` to `.gitignore`
- Use secret management (AWS Secrets Manager, Azure Key Vault, etc.) for production

---

## Related Documentation

- [API Response Codes](./API_RESPONSE_CODES.md) - Complete API error reference
- [Advanced Setup](./ADVANCED_SETUP.md) - Advanced deployment scenarios
- [Setup Guide](../README.md) - Getting started guide

---

**Last Updated:** February 4, 2026

