# 📋 Environment Variables Documentation

This document describes all environment variables used in SereniBase. Use this as a reference when configuring your deployment.

## Quick Start

1. Copy the example file to create your `.env`:
   ```bash
   cp config/.env.example .env
   ```

2. Update the `PUBLIC_HOST` variable with your server's IP or domain:
   ```bash
   PUBLIC_HOST=192.168.1.100  # or your domain
   ```

3. Configure required services (database, email, etc.)

4. Run the setup:
   ```bash
   make setup-all
   ```

---

## 🌐 Network Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `PUBLIC_HOST` | Your server's public IP or domain name | `localhost` | ✅ Yes |

**Examples:**
- Local development: `localhost`
- LAN access: `192.168.1.100`
- Production: `app.yourcompany.com`

---

## 🖥️ Server Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `SERVER_HOST` | IP address to bind the server | `0.0.0.0` | No |
| `SERVER_PORT` | Port number for the API server | `8080` | No |
| `SERVER_READ_TIMEOUT` | Request read timeout (seconds) | `30` | No |
| `SERVER_WRITE_TIMEOUT` | Response write timeout (seconds) | `30` | No |
| `SERVER_ENV` | Environment mode | `dev` | No |
| `SERVER_SCHEME` | Protocol (http/https) | `http` | No |

**Note:** Keep `SERVER_HOST=0.0.0.0` for Docker deployments to allow container networking.

---

## 🗄️ Database Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `DATABASE_HOST` | PostgreSQL host | `postgres` | ✅ Yes |
| `DATABASE_PORT` | PostgreSQL port | `5432` | No |
| `DATABASE_USER` | Database username | `postgres` | ✅ Yes |
| `DATABASE_PASSWORD` | Database password | `postgres` | ✅ Yes |
| `DATABASE_NAME` | Database name | `serenibase` | ✅ Yes |
| `DATABASE_SSL_MODE` | SSL mode (disable/require) | `disable` | No |
| `DATABASE_MAX_OPEN_CONNS` | Max open connections | `25` | No |
| `DATABASE_MAX_IDLE_CONNS` | Max idle connections | `5` | No |
| `DATABASE_CONN_MAX_LIFETIME` | Connection lifetime | `1h` | No |

---

## 🔐 Authentication Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `AUTH_URL` | Internal auth service URL | `http://jwt-provider:8081` | No |
| `AUTH_JWT_SECRET` | JWT signing secret (min 32 chars) | - | ✅ Yes |
| `AUTH_PORT` | Auth service port | `8081` | No |
| `AUTH_HOST` | Auth service bind address | `0.0.0.0` | No |
| `AUTH_RESET_PASSWORD_URL` | Password reset URL template | - | ✅ Yes |
| `AUTH_ALLOWED_ORIGINS` | CORS allowed origins | - | No |
| `AUTH_ENV` | Environment mode | `development` | No |
| `AUTH_LOG_LEVEL` | Log level | `info` | No |

**⚠️ Security:** Generate a strong random string for `AUTH_JWT_SECRET`:
```bash
openssl rand -base64 32
```

---

## 👤 Owner/Admin Account

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `OWNER_FIRST_NAME` | Admin first name | `Admin` | No |
| `OWNER_LAST_NAME` | Admin last name | `User` | No |
| `OWNER_EMAIL` | Admin email | `admin@example.com` | ✅ Yes |
| `OWNER_PASSWORD` | Admin password | `Admin@123` | ✅ Yes |
| `TEMPORARY_USER_PASSWORD` | Default password for new users | - | ✅ Yes |

**⚠️ Security:** Change default passwords in production!

---

## 📧 Email Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `EMAIL_URL` | Internal email service URL | `http://email-service:8082/api/v1/email` | No |
| `EMAIL_HOST` | Email service bind address | `0.0.0.0` | No |
| `EMAIL_PORT` | Email service port | `8082` | No |
| `EMAIL_SMTP_HOST` | SMTP server hostname | `smtp.gmail.com` | ✅ Yes |
| `EMAIL_SMTP_PORT` | SMTP server port | `587` | ✅ Yes |
| `EMAIL_SMTP_USERNAME` | SMTP username/email | - | ✅ Yes |
| `EMAIL_SMTP_PASSWORD` | SMTP password/app password | - | ✅ Yes |
| `EMAIL_FROM_EMAIL` | Sender email address | - | ✅ Yes |

### Gmail Setup
1. Enable 2-Factor Authentication on your Google account
2. Generate an App Password: Google Account → Security → App passwords
3. Use the app password for `EMAIL_SMTP_PASSWORD`

---

## 📁 Storage Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `STORAGE_URL` | Internal storage service URL | - | No |
| `STORAGE_DRIVER` | Storage backend: `local`, `minio`, `aws` | `local` | ✅ Yes |
| `STORAGE_DEV_PATH` | Local storage path | `./uploads` | For local |

### Local Storage
```env
STORAGE_DRIVER=local
STORAGE_DEV_PATH=./uploads
```

### MinIO Storage
| Variable | Description |
|----------|-------------|
| `STORAGE_MINIO_ENDPOINT` | MinIO server endpoint |
| `STORAGE_MINIO_ACCESS_KEY` | Access key |
| `STORAGE_MINIO_SECRET_KEY` | Secret key |
| `STORAGE_MINIO_BUCKET` | Bucket name |
| `STORAGE_MINIO_USE_SSL` | Use SSL (true/false) |

### AWS S3 Storage
| Variable | Description |
|----------|-------------|
| `STORAGE_AWS_REGION` | AWS region |
| `STORAGE_AWS_BUCKET` | S3 bucket name |
| `STORAGE_AWS_ACCESS_KEY` | AWS access key |
| `STORAGE_AWS_SECRET_KEY` | AWS secret key |

---

## 🦠 Antivirus Service

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `ANTIVIRUS_URL` | Internal antivirus service URL | - | No |
| `ANTIVIRUS_DRIVER` | Antivirus backend | `clamav` | No |
| `ANTIVIRUS_CLAMAV_ADDRESS` | ClamAV daemon address | `clamav:3310` | For ClamAV |
| `ANTIVIRUS_CLAMAV_TIMEOUT_SECONDS` | Scan timeout | `30` | No |
| `ANTIVIRUS_MAX_UPLOAD_SIZE_MB` | Max file size for scanning | `32` | No |

---

## 🎨 Frontend (Base UI)

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `BASEUI_VITE_API_BASE_URL` | Backend API URL for frontend | `http://localhost:8080` | ✅ Yes |

**Important:** This must be accessible from the user's browser, so use your public IP or domain.

---

## 🔒 CORS Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `CORS_ALLOWED_ORIGINS` | Allowed origins (comma-separated) | - | ✅ Yes |
| `CORS_ALLOWED_METHODS` | Allowed HTTP methods | `GET,POST,PUT,DELETE,OPTIONS,PATCH` | No |
| `CORS_ALLOWED_HEADERS` | Allowed headers | - | No |
| `CORS_ALLOW_CREDENTIALS` | Allow credentials | `true` | No |

**Example for production:**
```env
CORS_ALLOWED_ORIGINS=https://app.yourcompany.com,https://admin.yourcompany.com
```

---

## 📝 Logging Configuration

| Variable | Description | Default | Required |
|----------|-------------|---------|----------|
| `LOG_LEVEL` | Log level (debug/info/warn/error) | `info` | No |
| `LOG_FILE` | Log file name | `app.log` | No |
| `LOG_MAX_SIZE` | Max log file size (MB) | `50` | No |
| `LOG_MAX_BACKUPS` | Number of backup files | `10` | No |
| `LOG_MAX_AGE` | Max age of log files (days) | `30` | No |
| `LOG_COMPRESS` | Compress old log files | `true` | No |

---

## 🔧 Development Only

| Variable | Description | Required |
|----------|-------------|----------|
| `GIT_TOKEN` | GitHub token for cloning private repos | Dev only |

**⚠️ Warning:** Never commit real tokens to version control!

---

## 🚀 Deployment Examples

### Local Development
```env
PUBLIC_HOST=localhost
SERVER_ENV=dev
STORAGE_DRIVER=local
```

### LAN/Team Access
```env
PUBLIC_HOST=192.168.1.100
SERVER_ENV=dev
CORS_ALLOWED_ORIGINS=http://192.168.1.100:5050,http://localhost:5050
BASEUI_VITE_API_BASE_URL=http://192.168.1.100:8080
```

### Production
```env
PUBLIC_HOST=app.yourcompany.com
SERVER_ENV=production
SERVER_SCHEME=https
DATABASE_SSL_MODE=require
STORAGE_DRIVER=aws
CORS_ALLOWED_ORIGINS=https://app.yourcompany.com
BASEUI_VITE_API_BASE_URL=https://api.yourcompany.com
```

---

## 🔐 Security Checklist

Before going to production, ensure you have:

- [ ] Changed `AUTH_JWT_SECRET` to a strong random value
- [ ] Changed `OWNER_PASSWORD` and `TEMPORARY_USER_PASSWORD`
- [ ] Changed `DATABASE_PASSWORD` to a strong password
- [ ] Configured proper SMTP credentials
- [ ] Removed `GIT_TOKEN` or set it to empty
- [ ] Updated `CORS_ALLOWED_ORIGINS` to only include your domains
- [ ] Enabled `DATABASE_SSL_MODE=require` if using remote database
- [ ] Set `SERVER_ENV=production`
