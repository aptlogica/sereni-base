# 🚀 SereniBase Setup Guide

## 📋 Prerequisites

Before you begin, make sure you have the following installed:

| Tool | Required | Installation |
|------|----------|--------------|
| **Docker** | ✅ Yes | [Download Docker Desktop](https://www.docker.com/products/docker-desktop) |
| **Docker Compose** | ✅ Yes | Included with Docker Desktop |
| **Make** | ✅ Yes | Windows: `choco install make` or [GnuWin32](http://gnuwin32.sourceforge.net/packages/make.htm) |
| **Git** | ✅ Yes | [Download Git](https://git-scm.com/downloads) |

### Verify Prerequisites
```bash
docker --version
docker compose version
make --version
git --version
```

---

## 🚀 Quick Start (One Command)

```bash
make setup
```

This single command will:
1. ✅ Check prerequisites (Docker, Git)
2. ✅ Clone required microservices
3. ✅ Prompt for configuration (or use defaults)
4. ✅ Create `.env` file
5. ✅ Build and start all Docker containers

---

## ⚙️ Environment Variables

### Variables Set by Setup Script

These variables are configured when you run `make setup`. Press Enter to use defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| `PUBLIC_HOST` | `localhost` | Your server IP or domain |
| `OWNER_FIRST_NAME` | `Admin` | Admin user first name |
| `OWNER_LAST_NAME` | `User` | Admin user last name |
| `OWNER_EMAIL` | `admin@example.com` | Admin login email |
| `OWNER_PASSWORD` | `Admin@123` | Admin login password |

### Variables in `.env` File (Pre-configured)

These have working defaults and typically don't need changes:

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_HOST` | `postgres` | Database hostname |
| `DATABASE_USER` | `postgres` | Database username |
| `DATABASE_PASSWORD` | `postgres` | Database password |
| `DATABASE_NAME` | `serenibase` | Database name |
| `AUTH_JWT_SECRET` | (set) | JWT signing secret |

### Email Configuration

> ⚠️ **Email ** - Only configure if you need password reset and email notifications.
> This may require a paid SMTP service.

| Variable | Example | Description |
|----------|---------|-------------|
| `EMAIL_SMTP_HOST` | `smtp.gmail.com` | SMTP server host |
| `EMAIL_SMTP_PORT` | `587` | SMTP server port |
| `EMAIL_SMTP_USERNAME` | `your_email@gmail.com` | SMTP username |
| `EMAIL_SMTP_PASSWORD` | `your_app_password` | SMTP password or app password |
| `EMAIL_FROM_EMAIL` | `your_email@gmail.com` | Sender email address |

---

## 📖 Step-by-Step Setup

### Step 1: Clone the Repository
```bash
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base
```

### Step 2: Run Setup
```bash
make setup
```

### Step 3: Follow the Prompts
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
> 💡 Press **Enter** to accept default values shown in brackets

### Step 4: Wait for Services to Start
The setup will automatically:
- Build Docker images
- Start all containers
- Initialize the database

### Step 5: Access the Application
```
Frontend:  http://localhost:5050
Backend:   http://localhost:8080
MinIO:     http://localhost:9001
```

---

## 🔧 Useful Commands

| Command | Description |
|---------|-------------|
| `make setup` | Run full setup wizard |
| `make up` | Start all services |
| `make down` | Stop all services |
| `make logs` | View service logs |
| `make restart` | Restart all services |
| `make clean` | Remove all data and containers |

---

## 🔑 Default Login Credentials

| Field | Value |
|-------|-------|
| Email | `admin@example.com` |
| Password | `Admin@123` |

> ⚠️ **Change these in production!**

---

## 🐛 Troubleshooting

### Docker Not Found
```bash
# Install Docker Desktop from:
https://www.docker.com/products/docker-desktop
```

### Make Not Found (Windows)
```powershell
# Option 1: Using Chocolatey
choco install make

# Option 2: Run setup script directly
.\build\scripts\setup.bat
```

### Permission Denied (Linux/macOS)
```bash
chmod +x build/scripts/*.sh
./build/scripts/setup.sh
```

### Port Already in Use
```bash
# Check what's using the port
netstat -ano | findstr :8080  # Windows
lsof -i :8080                 # Linux/macOS

# Stop conflicting services or change ports in .env
```

### Reset Everything
```bash
make clean
make setup
```

---

## 📁 Project Structure

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

## 📚 Additional Documentation

- [Environment Variables Reference](../docs/ENV_QUICK_REFERENCE_CARD.md)
- [Environment Setup Guide](../docs/ENVIRONMENT_SETUP_GUIDE.md)
- [API Response Codes](../docs/API_RESPONSE_CODES.md)

---

**Happy Coding! 🎉**
