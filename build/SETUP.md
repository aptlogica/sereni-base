# SereniBase Setup Guide

For a full beginner-friendly from-scratch guide, see `build/SETUP_COMPLETE_GUIDE.md`.

## Prerequisites

Before you begin, make sure you have the following installed:

| Tool | Required | Installation |
|------|----------|--------------|
| Docker | Yes | https://www.docker.com/products/docker-desktop |
| Docker Compose | Yes | Included with Docker Desktop |
| Make | Yes | Windows: `choco install make` or http://gnuwin32.sourceforge.net/packages/make.htm |
| Git | Required for first-time setup | https://git-scm.com/downloads |

### Verify prerequisites
```bash
docker --version
docker compose version
make --version
git --version
```

---

## Quick Start (One Click)

### Option 1: Interactive Setup (Recommended)

Run the setup wizard which will prompt you for all required configuration:

```bash
make setup
```

The wizard will ask you for:
1. **Database Configuration**: Choose between default PostgreSQL or custom database
2. **Authentication**: JWT secret (can be auto-generated)
3. **Email Configuration**: SMTP credentials for notifications
4. **Network**: Public host/domain
5. **Admin Account**: Owner registration details

### Option 2: Pre-configured Setup

If you prefer to prepare your `.env` file first, create it with minimum required keys:

```env
# These will be asked during setup if not provided
DATABASE_USER=postgres
DATABASE_PASSWORD=CHANGEME_DB_PASSWORD
DATABASE_NAME=serenibase
AUTH_JWT_SECRET=replace-with-a-strong-random-secret

EMAIL_SMTP_HOST=your_email_host
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=your_email@gmail.com
EMAIL_SMTP_PASSWORD=your_app_password
EMAIL_FROM_EMAIL=your_email@gmail.com
```

Then run:
```bash
make setup
```

**Note**: If `.env` exists, the setup will preserve your existing values and only add missing variables.

---

## Environment Variables

### Automatically Configured by Setup Wizard

These are prompted and configured when you run `make setup`:

| Variable | Default | Description | When Asked |
|----------|---------|-------------|------------|
| `DATABASE_USER` | `postgres` | Database username | Database setup |
| `DATABASE_PASSWORD` | `postgres` | Database password | Database setup |
| `DATABASE_NAME` | `serenibase` | Database name | Database setup |
| `DATABASE_HOST` | `postgres` or custom | Database host | Database setup (if custom) |
| `DATABASE_PORT` | `5432` | Database port | Database setup (if custom) |
| `DATABASE_SSL_MODE` | `disable` | SSL mode | Database setup (if custom) |
| `AUTH_JWT_SECRET` | Auto-generated | JWT signing secret | Authentication setup |
| `EMAIL_SMTP_HOST` | `your_email_host` | SMTP server host | Email setup |
| `EMAIL_SMTP_PORT` | `587` | SMTP server port | Email setup |
| `EMAIL_SMTP_USERNAME` | (required) | SMTP username | Email setup |
| `EMAIL_SMTP_PASSWORD` | (required) | SMTP password | Email setup |
| `EMAIL_FROM_EMAIL` | Same as username | Sender email | Email setup |
| `STORAGE_DRIVER` | `minio` | Storage driver (local/minio/s3) | Storage setup |
| `STORAGE_DEV_PATH` | `./uploads` | Local storage path | Storage setup (if local) |
| `STORAGE_MINIO_ENDPOINT` | `minio:9000` | MinIO endpoint | Storage setup (if MinIO) |
| `STORAGE_MINIO_ACCESS_KEY` | `minioadmin` | MinIO access key | Storage setup (if MinIO) |
| `STORAGE_MINIO_SECRET_KEY` | `minioadmin` | MinIO secret key | Storage setup (if MinIO) |
| `STORAGE_MINIO_BUCKET` | `serenibase` | MinIO bucket name | Storage setup (if MinIO) |
| `STORAGE_MINIO_USE_SSL` | `false` | Use SSL for MinIO | Storage setup (if MinIO) |
| `STORAGE_AWS_REGION` | `us-east-1` | AWS region | Storage setup (if S3) |
| `STORAGE_AWS_BUCKET` | (required) | S3 bucket name | Storage setup (if S3) |
| `STORAGE_AWS_ACCESS_KEY` | (required) | AWS access key | Storage setup (if S3) |
| `STORAGE_AWS_SECRET_KEY` | (required) | AWS secret key | Storage setup (if S3) |
| `PUBLIC_HOST` | `localhost` | Your server IP or domain | Network setup |
| `OWNER_FIRST_NAME` | `Admin` | Admin user first name | Owner setup |
| `OWNER_LAST_NAME` | `User` | Admin user last name | Owner setup |
| `OWNER_EMAIL` | `admin@example.com` | Admin login email | Owner setup |
| `OWNER_PASSWORD` | `Admin@123` | Admin login password | Owner setup |

### Pre-filled System Variables

These are automatically set by the setup script with recommended values:

| Variable | Value | Description |
|----------|-------|-------------|
| `SERVER_HOST` | `0.0.0.0` | Server bind address |
| `SERVER_PORT` | `8080` | Server port |
| `SERVER_ENV` | `dev` | Environment mode |
| `AUTH_URL` | `http://jwt-provider:8081` | Auth service URL |
| `EMAIL_URL` | `http://email-service:8082/api/v1/email` | Email service URL |
| `STORAGE_URL` | `http://sereni-storage-provider:8083/api/v1` | Storage service URL |
| `ANTIVIRUS_URL` | `http://antivirus-service:8084` | Antivirus service URL |

### Optional Variables

You can add these to `.env` if needed. The setup preserves custom variables:

| Variable | Description |
|----------|-------------|
| `GIT_TOKEN` | GitHub PAT for private repos (if needed) |
| `LOG_LEVEL` | Logging level (default: `info`) |
| `CORS_ALLOWED_ORIGINS` | Custom CORS origins |

---

## Step-by-Step Setup

### Step 1: Clone the repository
```bash
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base
```

### Step 2: Run setup wizard
```bash
make setup
```

### Step 3: Follow the prompts

The wizard will guide you through:

#### 1. Database Configuration
```
Choose database setup:
  1. Use default PostgreSQL (Docker container)
  2. Use custom database credentials

Enter choice [1]:
```

If you choose **option 1** (recommended for local development):
```
Database User [postgres]:
Database Password [postgres]:
Database Name [serenibase]:
```

If you choose **option 2** (for external database):
```
Database Host:
Database Port [5432]:
Database User:
Database Password:
Database Name:
SSL Mode [disable]:
```

#### 2. Authentication Setup
```
JWT Secret (min 32 chars) [press Enter to generate]:
```
Press Enter to auto-generate a secure JWT secret.

#### 3. Email Configuration
```
SMTP Host [your_email_host]:
SMTP Port [587]:
SMTP Username (email):
SMTP Password (app password):
From Email [your_email@gmail.com]:
```

**Note**: For Gmail, you need to:
1. Enable 2-factor authentication
2. Generate an App Password: https://myaccount.google.com/apppasswords

#### 4. Storage Configuration
```
Choose storage driver:
  1. Local filesystem (for development only)
  2. MinIO (Docker container - recommended)
  3. MinIO Custom (external MinIO server)
  4. AWS S3

Enter choice [2]:
```

**Option 1: Local Filesystem** (not recommended for production)
```
Storage path [./uploads]:
```

**Option 2: MinIO Docker** (recommended for local development)
```
MinIO Access Key [minioadmin]:
MinIO Secret Key [minioadmin]:
Bucket Name [serenibase]:
```

**Option 3: MinIO Custom** (external MinIO server)
```
MinIO Endpoint (host:port): minio.example.com:9000
MinIO Access Key: your_access_key
MinIO Secret Key: ••••••••••
Bucket Name [serenibase]:
Use SSL (true/false) [false]:
```

**Option 4: AWS S3**
```
AWS Region [us-east-1]:
S3 Bucket Name: my-bucket
AWS Access Key: AKIAIOSFODNN7EXAMPLE
AWS Secret Key: ••••••••••••••••••••
```

#### 5. Network Configuration
```
Enter IP/domain [localhost]:
```

#### 6. Admin Account
```
First Name [Admin]:
Last Name [User]:
Email [admin@example.com]:
Password [Admin@123]:
```

### Step 4: Access the application
```
Frontend:  http://localhost:5050
Backend:   http://localhost:8080
MinIO:     http://localhost:9001
```

Default login credentials will be displayed at the end of setup.

---

## Useful Commands

| Command | Description |
|---------|-------------|
| `make setup` | Run full setup wizard |
| `make up` | Start all services |
| `make down` | Stop all services |
| `make logs` | View service logs |
| `make restart` | Restart all services |
| `make clean` | Remove all data and containers |

---

## Default Login Credentials

| Field | Value |
|-------|-------|
| Email | `admin@example.com` |
| Password | `Admin@123` |

Change these in production.

---

## Troubleshooting

### Docker not found
Install Docker Desktop from https://www.docker.com/products/docker-desktop

### Make not found (Windows)
```powershell
choco install make
```

### Permission denied (Linux/macOS)
```bash
chmod +x build/scripts/*.sh
./build/scripts/setup.sh
```

### Port already in use
```bash
# Check what's using the port
netstat -ano | findstr :8080  # Windows
lsof -i :8080                 # Linux/macOS
```

### Reset everything
```bash
make clean
make setup
```

---

## Project Structure

```
sereni-base/
├── .env                    # Environment configuration (created by setup)
├── docker-compose.all.yaml # Docker services definition
├── Makefile               # Make commands
├── build/
│   ├── config/
│   │   └── .env.example   # Environment template
│   └── scripts/
│       ├── setup.bat      # Windows setup script
│       ├── setup.sh       # Linux/macOS setup script
│       └── ...
└── services/              # Microservices (cloned by setup)
    ├── auth-service/
    ├── base-ui/
    ├── email-service/
    └── storage-service/
```

---

## Additional Documentation

- ../docs/ENV_QUICK_REFERENCE_CARD.md
- ../docs/ENVIRONMENT_SETUP_GUIDE.md
- ../docs/API_RESPONSE_CODES.md
