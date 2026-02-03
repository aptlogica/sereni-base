<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-18+-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-15+-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
</p>

<p align="center">
  <a href="https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b">
    <img src="https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=alert_status&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb" alt="Quality Gate">
  </a>
  <a href="https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b">
    <img src="https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=coverage&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb" alt="Coverage">
  </a>
  <a href="https://sonar.aptlogica.com/dashboard?id=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b">
    <img src="https://sonar.aptlogica.com/api/project_badges/measure?project=aptlogica_sereni-base_58d1e56c-7e79-4847-b5b2-68b244b3b65b&metric=software_quality_security_rating&token=sqb_f3e90f5e2b4263ed304f449df73cd8b1f6d6d7cb" alt="Security">
  </a>
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/PRs-Welcome-brightgreen.svg?style=flat-square" alt="PRs Welcome">
</p>

<p align="center">
  <strong>🚀 A modern, open-source platform for creating and managing business data</strong>
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-features">Features</a> •
  <a href="#-documentation">Docs</a> •
  <a href="#-contributing">Contributing</a> •
  <a href="#-community">Community</a>
</p>

---

## ✨ Features

- 🎨 **Modern React UI** - Beautiful, responsive interface built with React 18+ and TailwindCSS
- 🔐 **Secure Authentication** - JWT-based auth with role-based access control
- 📧 **Email Notifications** - Built-in email service with SMTP support
- 📁 **Flexible Storage** - Support for local, MinIO, and AWS S3 storage
- 🦠 **Antivirus Scanning** - Integrated ClamAV for file security
- 🐳 **Docker Ready** - One-command deployment with Docker Compose
- 🌍 **Cross-Platform** - Works on Windows, macOS, and Linux
- 📊 **API Documentation** - Auto-generated Swagger/OpenAPI docs
- 🔧 **Highly Configurable** - Easy configuration via environment variables

---

## 🚀 Quick Start

Get SereniBase running in under 5 minutes!

### Prerequisites

Before you begin, ensure you have:

| Requirement | Version | Installation Guide |
|-------------|---------|-------------------|
| Docker | v20.10+ | [Install Docker](https://docs.docker.com/get-docker/) |
| Docker Compose | v2.0+ | [Install Compose](https://docs.docker.com/compose/install/) |
| Git | Latest | [Install Git](https://git-scm.com/downloads) |

### 🐧 Linux / macOS

```bash
# Clone the repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Run the interactive setup wizard
make setup
```

### 🪟 Windows

```powershell
# Clone the repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Run the setup script (choose one):
.\build\scripts\setup.bat           # CMD users
.\build\scripts\setup-all.ps1       # PowerShell users
```

### 🎉 That's it!

Once setup completes, open your browser:

| Service | URL | Description |
|---------|-----|-------------|
| 🎨 **Frontend** | http://localhost:5050 | Main application |
| 🔌 **API** | http://localhost:8080 | Backend REST API |
| 📖 **API Docs** | http://localhost:8080/swagger/index.html | Swagger documentation |
| 📦 **MinIO** | http://localhost:9001 | Object storage console |

### 🔑 Default Login

```
Email:    admin@example.com
Password: Admin@123
```

> ⚠️ **Security Note:** Please change the default credentials before deploying to production!

---

## � Documentation

| Document | Description |
|----------|-------------|
| 📋 [Environment Configuration](docs/ENV_CONFIGURATION.md) | Complete guide to all environment variables |
| 🔌 [API Documentation](http://localhost:8080/swagger/index.html) | Interactive API reference (requires running server) |
| 🐳 [Docker Deployment](docs/ENV_CONFIGURATION.md#-deployment-examples) | Production deployment guide |

---

## 🛠️ Development

### Available Commands

```bash
# Show all commands
make help

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🚀 SETUP
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
make setup          # Interactive setup wizard (recommended)
make setup-all      # Automated full setup

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔧 DEVELOPMENT
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
make up-all         # Start all services
make down-all       # Stop all services
make rebuild        # Rebuild and restart
make logs           # View live logs
make status         # Show service status

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🧹 MAINTENANCE
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
make clean          # Remove all containers & data
```

### Project Structure

```
sereni-base/
│
├── 📁 build/                  # Build & setup related files
│   ├── config/               # Configuration templates
│   │   └── .env.example      # Environment variables template
│   ├── scripts/              # Setup & utility scripts
│   │   ├── setup.sh          # Linux/macOS setup wizard
│   │   ├── setup.bat         # Windows CMD setup
│   │   ├── setup-all.ps1     # PowerShell full setup
│   │   ├── clone-services.*  # Service cloning scripts
│   │   └── clone-go-postgres-rest.* # Dependency cloning
│   └── README.md             # Build documentation
│
├── 📁 docs/                   # Documentation
│   └── ENV_CONFIGURATION.md  # Env vars guide
│
├── 📁 services/               # Microservices (auto-cloned)
│   ├── auth-service/         # 🔐 JWT authentication
│   ├── email-service/        # 📧 Email notifications
│   ├── storage-service/      # 📁 File storage
│   ├── antivirus-service/    # 🦠 File scanning
│   └── base-ui/              # 🎨 React frontend
│
├── 📁 internal/               # Backend source code
├── 📁 cmd/                    # Application entry points
│
├── 🐳 docker-compose.all.yaml # Full stack deployment
├── 📄 Makefile               # Build automation
└── 📄 .env                   # Your configuration (create from template)
```

---

## 🏗️ Architecture

<p align="center">
  <img src="https://img.shields.io/badge/Frontend-React-61DAFB?style=flat-square&logo=react" alt="React">
  <img src="https://img.shields.io/badge/Backend-Go/Gin-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Database-PostgreSQL-4169E1?style=flat-square&logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Storage-MinIO-C72E49?style=flat-square&logo=minio" alt="MinIO">
</p>

```
                            ┌─────────────────┐
                            │   🌐 Browser    │
                            └────────┬────────┘
                                     │
                    ┌────────────────┴────────────────┐
                    │                                 │
              ┌─────▼─────┐                    ┌──────▼──────┐
              │  Frontend │                    │   Backend   │
              │  (React)  │◄──────────────────►│  (Go/Gin)   │
              │   :5050   │                    │    :8080    │
              └───────────┘                    └──────┬──────┘
                                                     │
                    ┌────────────────┬───────────────┼───────────────┐
                    │                │               │               │
              ┌─────▼─────┐   ┌──────▼─────┐  ┌─────▼─────┐  ┌──────▼──────┐
              │   Auth    │   │   Email    │  │  Storage  │  │  Antivirus  │
              │  Service  │   │  Service   │  │  Service  │  │   Service   │
              │   :8081   │   │   :8082    │  │   :8083   │  │    :8084    │
              └─────┬─────┘   └────────────┘  └─────┬─────┘  └──────┬──────┘
                    │                              │               │
              ┌─────▼─────┐                  ┌─────▼─────┐   ┌─────▼─────┐
              │ PostgreSQL│                  │   MinIO   │   │  ClamAV   │
              │   :5432   │                  │   :9000   │   │   :3310   │
              └───────────┘                  └───────────┘   └───────────┘
```

---

## 🤝 Contributing

We love contributions! SereniBase is open source and we welcome contributors of all skill levels.

### Ways to Contribute

- 🐛 **Report Bugs** - Found a bug? [Open an issue](https://github.com/aptlogica/sereni-base/issues/new?template=bug_report.md)
- � **Suggest Features** - Have an idea? [Start a discussion](https://github.com/aptlogica/sereni-base/discussions)
- 📝 **Improve Docs** - Help make our docs better
- 🔧 **Submit PRs** - Fix bugs or add features

### Getting Started

1. **Fork** the repository
2. **Clone** your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/sereni-base.git
   ```
3. **Create** a feature branch:
   ```bash
   git checkout -b feature/amazing-feature
   ```
4. **Make** your changes
5. **Test** your changes:
   ```bash
   make rebuild
   ```
6. **Commit** with a clear message:
   ```bash
   git commit -m "feat: add amazing feature"
   ```
7. **Push** to your fork:
   ```bash
   git push origin feature/amazing-feature
   ```
8. **Open** a Pull Request

### Commit Convention

We follow [Conventional Commits](https://www.conventionalcommits.org/):

| Type | Description |
|------|-------------|
| `feat:` | New feature |
| `fix:` | Bug fix |
| `docs:` | Documentation |
| `style:` | Formatting |
| `refactor:` | Code restructure |
| `test:` | Tests |
| `chore:` | Maintenance |

---

## 🔒 Security

### Before Production Deployment

- [ ] Change `AUTH_JWT_SECRET` to a strong random value
- [ ] Change all default passwords
- [ ] Configure HTTPS/SSL
- [ ] Update `CORS_ALLOWED_ORIGINS`
- [ ] Enable database SSL mode
- [ ] Remove development tokens

### Reporting Vulnerabilities

Found a security issue? Please email **security@aptlogica.com** instead of opening a public issue.

---

## 🆘 Troubleshooting

<details>
<summary><strong>🔴 CORS Errors</strong></summary>

Update `CORS_ALLOWED_ORIGINS` in your `.env`:
```bash
CORS_ALLOWED_ORIGINS=http://YOUR_IP:5050,http://localhost:5050
```
Then restart: `make rebuild`
</details>

<details>
<summary><strong>🔴 Services Not Starting</strong></summary>

```bash
# Check logs
make logs

# Rebuild everything
make rebuild

# Full reset
make clean && make setup-all
```
</details>

<details>
<summary><strong>🔴 Database Connection Failed</strong></summary>

```bash
# Check if PostgreSQL is running
docker compose -f docker-compose.all.yaml ps postgres

# View database logs
docker compose -f docker-compose.all.yaml logs postgres
```
</details>

<details>
<summary><strong>🔴 Port Already in Use</strong></summary>

```bash
# Linux/macOS
lsof -i :5050
lsof -i :8080

# Windows
netstat -ano | findstr :5050
netstat -ano | findstr :8080
```
</details>

---

## 💬 Community

- 🐛 [Issue Tracker](https://github.com/aptlogica/sereni-base/issues)
- 💬 [Discussions](https://github.com/aptlogica/sereni-base/discussions)
- 📧 [Email Support](mailto:support@aptlogica.com)

---

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

```
MIT License

Copyright (c) 2024 Aptlogica

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software...
```

---

## ⭐ Star History

If you find SereniBase useful, please consider giving it a star! It helps others discover the project.

<p align="center">
  <a href="https://github.com/aptlogica/sereni-base/stargazers">
    <img src="https://img.shields.io/github/stars/aptlogica/sereni-base?style=social" alt="GitHub Stars">
  </a>
</p>

---

<p align="center">
  Made with ❤️ by <a href="https://github.com/aptlogica">Aptlogica</a> and <a href="https://github.com/aptlogica/sereni-base/graphs/contributors">contributors</a>
</p>
