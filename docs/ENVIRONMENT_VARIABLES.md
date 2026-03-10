# Environment Variables Guide

This document categorizes all environment variables in SereniBase into what users need to configure vs what should have sensible defaults.

## 🔧 User-Configurable Variables (Setup Prompts)

These variables SHOULD be prompted during setup:

### Network Configuration
- **PUBLIC_HOST** - The IP/domain where users access the application
  - Examples: `localhost`, `192.168.1.100`, `myapp.example.com`

### Owner/Admin Account
- **OWNER_FIRST_NAME** - First admin user's first name
- **OWNER_LAST_NAME** - First admin user's last name  
- **OWNER_EMAIL** - First admin user's email (login)
- **OWNER_PASSWORD** - First admin user's password

### Optional: Email Configuration (if user wants email features)
- **EMAIL_SMTP_HOST** - SMTP server (e.g., smtp.gmail.com)
- **EMAIL_SMTP_PORT** - SMTP port (e.g., 587)
- **EMAIL_SMTP_USERNAME** - Email username
- **EMAIL_SMTP_PASSWORD** - Email password
- **EMAIL_FROM_EMAIL** - From email address

---

## ⚙️ System Variables (Auto-Configured, No User Input Needed)

These variables should NOT be prompted - they are automatically configured based on PUBLIC_HOST or have sensible defaults:

### Automatically Derived from PUBLIC_HOST
- `AUTH_RESET_PASSWORD_URL=http://${PUBLIC_HOST}:5050/reset-password?token=%s`
- `ANTIVIRUS_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,...`
- `AUTH_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,...`
- `EMAIL_ALLOWED_ORIGIN=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,...`
- `STORAGE_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:8080,http://${PUBLIC_HOST}:5050,...`
- `BASEUI_VITE_API_BASE_URL=http://${PUBLIC_HOST}:8080`
- `CORS_ALLOWED_ORIGINS=http://${PUBLIC_HOST}:5050,http://${PUBLIC_HOST}:8080,...`

### Server Configuration (Sensible Defaults)
- `SERVER_HOST=0.0.0.0` - Bind to all interfaces
- `SERVER_PORT=8080` - Backend port
- `SERVER_SCHEME=http` - HTTP protocol
- `SERVER_ENV=dev` - Development environment

### Database Configuration (Docker defaults)
- `DATABASE_HOST=postgres` - Docker service name
- `DATABASE_PORT=5432` - PostgreSQL default port
- `DATABASE_USER=postgres` - Default user
- `DATABASE_PASSWORD=postgres` - Default password (should be changed in production)
- `DATABASE_NAME=serenibase` - Database name

### Authentication Service (Internal)
- `AUTH_URL=http://jwt-provider:8081` - Internal Docker network URL
- `AUTH_JWT_SECRET=CHANGE_THIS_TO_A_SECURE_RANDOM_STRING_MIN_32_CHARS` - Should be auto-generated
- `AUTH_PORT=8081`
- `AUTH_HOST=0.0.0.0`

### Email Service (Internal)
- `EMAIL_URL=http://email-service:8082/api/v1/email`
- `EMAIL_HOST=0.0.0.0`
- `EMAIL_PORT=8082`

### Storage Service (Defaults)
- `STORAGE_URL=http://sereni-storage-provider:8083/api/v1`
- `STORAGE_DRIVER=local` - Local filesystem by default
- `STORAGE_DEV_PATH=./uploads`
- All cloud storage settings (AWS, MinIO) - Not needed unless user selects cloud storage

### Antivirus Service (Internal)
- `ANTIVIRUS_URL=http://antivirus-service:8084`
- `ANTIVIRUS_HOST=0.0.0.0`
- `ANTIVIRUS_PORT=8084`
- `ANTIVIRUS_DRIVER=clamav`
- `ANTIVIRUS_CLAMAV_ADDRESS=clamav:3310`

### CORS Configuration (Auto-derived)
- `CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH`
- `CORS_ALLOWED_HEADERS=Content-Type,...`
- `CORS_ALLOW_CREDENTIALS=true`

### Logging (Defaults)
- `LOG_LEVEL=info`
- `LOG_FILE=app.log`
- `LOG_MAX_SIZE=50`
- `LOG_MAX_BACKUPS=10`
- `LOG_MAX_AGE=30`
- `LOG_COMPRESS=true`

### Assets (Defaults)
- `ASSET_MAX_SIZE=5242880` - 5MB default
- `TEMPORARY_USER_PASSWORD=ChangeMe@123`

---

## 📝 Setup Script Workflow

### Minimal Setup (Recommended)
1. Prompt for **PUBLIC_HOST** (IP/domain)
2. Prompt for **Owner Account** details (first name, last name, email, password)
3. Auto-configure all other variables based on PUBLIC_HOST
4. Start services

### Advanced Setup (Optional)
1. Ask "Do you want to configure email notifications?" (Yes/No)
   - If Yes: Prompt for SMTP settings
   - If No: Skip email configuration
2. Continue with minimal setup

---

## 🔐 Security Recommendations

### Variables that should be auto-generated (not prompted):
- `AUTH_JWT_SECRET` - Generate a secure random 32+ character string
- `DATABASE_PASSWORD` - Generate a random password in production

### Variables that should be changed in production:
- `TEMPORARY_USER_PASSWORD` - Used for new user accounts

### Variables that should NOT be committed:
- `GIT_TOKEN` - Only for development, should be in .gitignore

---

## Example: Simplified Setup Flow

```bash
========================================================================
                     SERENIBASE SETUP WIZARD
========================================================================

Network Configuration
--------------------
Enter your IP address or domain name: 192.168.1.100

Owner Account Setup
------------------
First Name: John
Last Name: Doe
Email: john@example.com
Password: ********

Email Configuration (Optional)
-----------------------------
Configure email notifications? [y/N]: n

========================================================================
Configuring SereniBase with:
  - Public Host: 192.168.1.100
  - Admin Email: john@example.com
  - Email Notifications: Disabled

Starting services...
========================================================================
```

This keeps the setup simple while still allowing advanced users to configure optional features!
