# 📊 Environment Variables Quick Reference Card

## 🎯 TL;DR - Minimum Required for Production

```bash
PUBLIC_HOST=your-domain.com
OWNER_EMAIL=admin@your-domain.com
OWNER_PASSWORD=YourStrong123!Password
AUTH_JWT_SECRET=at-least-32-chars-random-secret-key-here
DATABASE_PASSWORD=strong_database_password_123
TEMPORARY_USER_PASSWORD=random_temp_password_456

# Everything else uses smart defaults!
```

---

## 📋 Variable Classification Matrix

| Symbol | Meaning |
|--------|---------|
| 🔴 | **REQUIRED** - Must set before running |
| 🟡 | **SECURITY** - Has default but insecure |
| 🟢 | **OPTIONAL** - Good default, change only if needed |
| 🔵 | **FEATURE** - Required only for specific features |
| ⚙️ | **AUTO** - Auto-configured from other variables |

---

## 🌐 Network & Server

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `PUBLIC_HOST` | 🔴 | `localhost` | Your domain/IP |
| `SERVER_HOST` | 🟢 | `0.0.0.0` | Bind address |
| `SERVER_PORT` | 🟢 | `8080` | HTTP port |
| `SERVER_ENV` | 🟢 | `dev` | `dev`/`staging`/`production` |
| `SERVER_SCHEME` | 🟢 | `http` | `http` or `https` |
| `SERVER_READ_TIMEOUT` | 🟢 | `30` | Seconds |
| `SERVER_WRITE_TIMEOUT` | 🟢 | `30` | Seconds |

---

## 🗄️ Database

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `DATABASE_HOST` | 🟢 | `localhost` | Use `postgres` for Docker |
| `DATABASE_PORT` | 🟢 | `5432` | PostgreSQL port |
| `DATABASE_USER` | 🟢 | `postgres` | DB username |
| `DATABASE_PASSWORD` | 🟡 | `postgres` | ⚠️ Change for production! |
| `DATABASE_NAME` | 🟢 | `serenibase` | Database name |
| `DATABASE_SSL_MODE` | 🟢 | `disable` | Use `require` for production |
| `DATABASE_MAX_OPEN_CONNS` | 🟢 | `25` | Connection pool size |
| `DATABASE_MAX_IDLE_CONNS` | 🟢 | `5` | Idle connections |
| `DATABASE_CONN_MAX_LIFETIME` | 🟢 | `1h` | Connection lifetime |

---

## 🔐 Authentication & Security

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `AUTH_JWT_SECRET` | 🔴 | Insecure default | **32+ chars required** |
| `AUTH_URL` | ⚙️ | `http://localhost:8081` | Auto-set for Docker |
| `AUTH_RESET_PASSWORD_URL` | ⚙️ | Auto from `PUBLIC_HOST` | Password reset page |
| `AUTH_JWT_ACCESS_TOKEN_EXPIRY` | 🟢 | `3600` | 1 hour (seconds) |
| `AUTH_JWT_REFRESH_TOKEN_EXPIRY` | 🟢 | `86400` | 24 hours (seconds) |
| `AUTH_JWT_ISSUER` | 🟢 | `serenibase` | JWT issuer claim |

---

## 👤 Owner/Admin Account

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `OWNER_FIRST_NAME` | 🟢 | `Admin` | Admin first name |
| `OWNER_LAST_NAME` | 🟢 | `User` | Admin last name |
| `OWNER_EMAIL` | 🔴 | `your-admin-email@example.com` | **Must be valid email** |
| `OWNER_PASSWORD` | 🔴 | `your-strong-password` | **Set a strong password** |
| `TEMPORARY_USER_PASSWORD` | 🟡 | Random string | For new users |

---

## 📧 Email Service

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `EMAIL_URL` | ⚙️ | Auto-configured | Internal service URL |
| `EMAIL_SMTP_HOST` | 🔵 | None | **Required for email features** |
| `EMAIL_SMTP_PORT` | 🔵 | None | Usually `587` (TLS) or `465` (SSL) |
| `EMAIL_SMTP_USERNAME` | 🔵 | None | SMTP auth username |
| `EMAIL_SMTP_PASSWORD` | 🔵 | None | SMTP auth password |
| `EMAIL_FROM_EMAIL` | 🔵 | None | Sender address |

**Note:** Email features (password reset, notifications) won't work without SMTP config.

---

## 📁 Storage Service

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `STORAGE_URL` | ⚙️ | Auto-configured | Internal service URL |
| `STORAGE_DRIVER` | 🟢 | `local` | `local`, `minio`, or `aws` |
| `STORAGE_DEV_PATH` | 🟢 | `./uploads` | Local storage path |

### If STORAGE_DRIVER=minio

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `STORAGE_MINIO_ENDPOINT` | 🔵 | `minio:9000` | MinIO server |
| `STORAGE_MINIO_ACCESS_KEY` | 🔵 | `minioadmin` | Access key |
| `STORAGE_MINIO_SECRET_KEY` | 🔵 | `minioadmin` | Secret key |
| `STORAGE_MINIO_BUCKET` | 🔵 | `serenibase` | Bucket name |
| `STORAGE_MINIO_USE_SSL` | 🔵 | `false` | SSL/TLS |

### If STORAGE_DRIVER=aws

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `STORAGE_AWS_REGION` | 🔵 | `us-east-1` | AWS region |
| `STORAGE_AWS_BUCKET` | 🔵 | None | **S3 bucket name** |
| `STORAGE_AWS_ACCESS_KEY` | 🔵 | None | **AWS access key** |
| `STORAGE_AWS_SECRET_KEY` | 🔵 | None | **AWS secret key** |

---

## 🦠 Antivirus Service

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `ANTIVIRUS_URL` | 🔵 | None | Enable file scanning |
| `ANTIVIRUS_DRIVER` | 🟢 | `clamav` | Only ClamAV supported |
| `ANTIVIRUS_CLAMAV_ADDRESS` | 🔵 | `clamav:3310` | ClamAV daemon |
| `ANTIVIRUS_CLAMAV_TIMEOUT_SECONDS` | 🟢 | `30` | Scan timeout |
| `ANTIVIRUS_MAX_UPLOAD_SIZE_MB` | 🟢 | `32` | Max file size |

---

## 🎨 Frontend

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `BASEUI_VITE_API_BASE_URL` | ⚙️ | Auto from `PUBLIC_HOST` | Frontend API endpoint |

---

## 🔒 CORS Configuration

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `CORS_ALLOWED_ORIGINS` | ⚙️ | Auto from `PUBLIC_HOST` | ⚠️ Should restrict in production |
| `CORS_ALLOWED_METHODS` | 🟢 | Standard methods | HTTP methods |
| `CORS_ALLOWED_HEADERS` | 🟢 | Standard + custom | Request headers |
| `CORS_ALLOW_CREDENTIALS` | 🟢 | `true` | Allow credentials |

**Production:** Explicitly set `CORS_ALLOWED_ORIGINS` to your domains only!

---

## 📝 Logging

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `LOG_LEVEL` | 🟢 | `info` | `debug`/`info`/`warn`/`error` |
| `LOG_FILE` | 🟢 | `app.log` | Log file path |
| `LOG_MAX_SIZE` | 🟢 | `50` | Max file size (MB) |
| `LOG_MAX_BACKUPS` | 🟢 | `10` | Old files to keep |
| `LOG_MAX_AGE` | 🟢 | `30` | Max age (days) |
| `LOG_COMPRESS` | 🟢 | `true` | Compress old logs |

---

## 📦 Assets

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `ASSET_MAX_SIZE` | 🟢 | `5242880` | 5MB in bytes |

---

## 🔄 Redis (Optional)

| Variable | Type | Default | Notes |
|----------|------|---------|-------|
| `REDIS_ENABLED` | 🟢 | `true` | Enable Redis caching |
| `REDIS_URL` | 🔵 | `redis://localhost:6379` | If enabled |
| `REDIS_PASSWORD` | 🔵 | Empty | If Redis has auth |

---

## 📊 Configuration Decision Tree

```
Do you need to change this variable?
│
├─ Is it marked 🔴 (REQUIRED)?
│  └─ YES → Must set it
│
├─ Is it marked 🟡 (SECURITY)?
│  └─ Production? → YES → Change it to secure value
│                  NO → Can use default for testing
│
├─ Is it marked 🔵 (FEATURE)?
│  └─ Using that feature? → YES → Must set it
│                           NO → Skip it
│
├─ Is it marked ⚙️ (AUTO)?
│  └─ NO → It's auto-configured from PUBLIC_HOST
│
└─ Is it marked 🟢 (OPTIONAL)?
   └─ Default works? → YES → Skip it
                      NO → Override it
```

---

## 🎯 Configuration Scenarios

### Scenario: Quick Local Test
**Set only:**
- 🔴 `PUBLIC_HOST=localhost`
- 🔴 `OWNER_EMAIL=test@test.com`
- 🔴 `OWNER_PASSWORD=Test123!`
- 🔴 `AUTH_JWT_SECRET=test-secret-min-32-chars`

**Everything else uses defaults!**

### Scenario: LAN Development
**Additionally set:**
- 🟢 `PUBLIC_HOST=192.168.1.100` (your IP)
- ⚙️ All CORS, URLs auto-update

### Scenario: Production
**Additionally set:**
- 🟡 All security variables (strong passwords)
- 🔵 Email configuration (SMTP)
- 🔵 Storage (AWS S3 or MinIO)
- 🟢 `SERVER_ENV=production`
- 🟢 `LOG_LEVEL=warn`
- 🟢 `DATABASE_SSL_MODE=require`
- 🔒 Restrict `CORS_ALLOWED_ORIGINS`

---

## 🔐 Security Checklist

For production, ensure these are **NOT** using insecure defaults:

- [ ] `OWNER_PASSWORD` - Strong password (16+ chars)
- [ ] `AUTH_JWT_SECRET` - Random 64+ chars
- [ ] `DATABASE_PASSWORD` - Strong password
- [ ] `TEMPORARY_USER_PASSWORD` - Random password
- [ ] `CORS_ALLOWED_ORIGINS` - Specific domains only
- [ ] `DATABASE_SSL_MODE` - Set to `require`
- [ ] Email SMTP credentials configured
- [ ] `.env` file in `.gitignore`
- [ ] Secrets in secure vault (not .env)

---

## 🚀 Quick Start Commands

### Use Interactive Setup (Recommended)
```bash
# Windows
.\setup-interactive.ps1

# Linux/macOS
./setup-interactive.sh
```

### Manual Setup
```bash
# Copy template
cp build/config/.env.example .env

# Edit (set 🔴 REQUIRED variables)
nano .env

# Start
docker-compose up -d
```

---

## 📚 Full Documentation

- **[Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md)** - Complete reference
- **[Interactive Setup README](../INTERACTIVE_SETUP_README.md)** - Script documentation
- **[Environment Variables](./ENVIRONMENT_VARIABLES.md)** - Detailed docs

---

## 💡 Pro Tips

### Generate Secure JWT Secret
```bash
# Linux/macOS
openssl rand -base64 64

# PowerShell
[Convert]::ToBase64String((1..64 | ForEach-Object { Get-Random -Maximum 256 }))
```

### Check What's Being Used
```bash
# View your .env
cat .env

# Check specific variable
grep "PUBLIC_HOST" .env
```

### Verify Configuration
```bash
# Start and check logs
docker-compose up -d
docker-compose logs serenibase
```

### Reset Configuration
```bash
# Delete and regenerate
rm .env
./setup-interactive.sh
```

---

**Print this card for quick reference! 📄**

---

**Last Updated:** February 4, 2026
