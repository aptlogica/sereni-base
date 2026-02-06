# SereniBase Setup Guide

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

### Step 1: Create the required `.env`
Create a `.env` file in the project root with the required keys below.

```env
DATABASE_USER=postgres
DATABASE_PASSWORD=postgres
DATABASE_NAME=serenibase
AUTH_JWT_SECRET=replace-with-a-strong-random-secret

GIT_TOKEN=your_github_pat

EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=your_email@gmail.com
EMAIL_SMTP_PASSWORD=your_app_password
EMAIL_FROM_EMAIL=your_email@gmail.com
```

Notes:
1. `GIT_TOKEN` must be a GitHub Personal Access Token that can access the microservice repos.
2. Email settings are required for password reset and notifications. Use an app password if your provider requires it.
3. Do not commit `.env` to source control.

### Step 2: Run setup
```bash
make setup
```

This will:
1. Check prerequisites (Docker, Git)
2. Clone required microservices
3. Prompt for configuration (or use defaults)
4. Build and start all Docker containers

---

## Environment Variables

### Variables set by the setup script

These are configured when you run `make setup`. Press Enter to use defaults.

| Variable | Default | Description |
|----------|---------|-------------|
| `PUBLIC_HOST` | `localhost` | Your server IP or domain |
| `OWNER_FIRST_NAME` | `Admin` | Admin user first name |
| `OWNER_LAST_NAME` | `User` | Admin user last name |
| `OWNER_EMAIL` | `admin@example.com` | Admin login email |
| `OWNER_PASSWORD` | `Admin@123` | Admin login password |

### Required in `.env` before setup

These must already exist in `.env` before you run `make setup`.

| Variable | Example | Description |
|----------|---------|-------------|
| `DATABASE_USER` | `postgres` | Database username |
| `DATABASE_PASSWORD` | `postgres` | Database password |
| `DATABASE_NAME` | `serenibase` | Database name |
| `AUTH_JWT_SECRET` | `replace-with-strong-secret` | JWT signing secret |
| `GIT_TOKEN` | `your_github_pat` | GitHub token for cloning services |
| `EMAIL_SMTP_HOST` | `smtp.gmail.com` | SMTP server host |
| `EMAIL_SMTP_PORT` | `587` | SMTP server port |
| `EMAIL_SMTP_USERNAME` | `your_email@gmail.com` | SMTP username |
| `EMAIL_SMTP_PASSWORD` | `your_app_password` | SMTP password or app password |
| `EMAIL_FROM_EMAIL` | `your_email@gmail.com` | Sender email address |

---

## Step-by-Step Setup

### Step 1: Clone the repository
```bash
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base
```

### Step 2: Add `.env`
Create `.env` in the project root using the required keys listed above.

### Step 3: Run setup
```bash
make setup
```

### Step 4: Follow the prompts
```
========================================================================
                     SERENIBASE SETUP WIZARD
========================================================================

Enter IP/domain [localhost]:
First Name [Admin]:
Last Name [User]:
Email [admin@example.com]:
Password [Admin@123]:
```
Press Enter to accept default values shown in brackets.

### Step 5: Access the application
```
Frontend:  http://localhost:5050
Backend:   http://localhost:8080
MinIO:     http://localhost:9001
```

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
