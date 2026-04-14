# SereniBase

### Where Data Becomes Software

An open-source, self-hosted alternative to Airtable, Notion databases, and NocoDB — built to create apps, workflows, and APIs on top of your data.

---

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-18+-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-15+-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
</p>
<p align="center">
  <img src="https://github.com/aptlogica/sereni-base/actions/workflows/ci.yml/badge.svg" alt="Github Actions">
  
  <img src="https://github.com/aptlogica/sereni-base/actions/workflows/github-code-scanning/codeql/badge.svg" alt="CodeQL">
  <a href="https://www.bestpractices.dev/projects/12425"><img src="https://www.bestpractices.dev/projects/12425/badge"></a>
  <img src="https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat" alt="Go Report">
  <a href="https://app.fossa.com/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base?ref=badge_shield&issueType=security" alt="FOSSA Status"><img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base.svg?type=shield&issueType=security"/></a>
  <a href="https://app.fossa.com/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base?ref=badge_shield&issueType=license" alt="FOSSA Status"><img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base.svg?type=shield&issueType=license"/></a>
</p>

<p align="center">
  <a href="https://sonarcloud.io/summary/new_code?id=aptlogica_sereni-base">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=aptlogica_sereni-base&metric=alert_status" alt="Quality Gate">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=aptlogica_sereni-base">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=aptlogica_sereni-base&metric=coverage" alt="Coverage">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=aptlogica_sereni-base">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=aptlogica_sereni-base&metric=security_rating" alt="Security">
  </a>
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/PRs-Welcome-brightgreen.svg?style=flat-square" alt="PRs Welcome">
  <img src="https://img.shields.io/badge/Status-Beta-orange.svg?style=flat-square" alt="Beta">
</p>


---

⭐ **If this project helps you, please consider giving it a star!**  
👉 https://github.com/aptlogica/sereni-base

---

## 🚀 What is SereniBase?

SereniBase is a **production-ready, open-source platform** for building data-driven systems.

Think:

👉 Airtable + NocoDB  
👉 But **modular, extensible, API-first, and self-hosted**

It allows teams to:
- Build internal tools  
- Manage structured data  
- Create workflows  
- Extend backend systems  

---

## 🔥 Why SereniBase?

Most no-code tools work… until they don’t.

| Problem | SereniBase Solution |
|--------|-------------------|
| Vendor lock-in | ✅ Self-hosted |
| Limited extensibility | ✅ Open-source & modular |
| Expensive scaling | ✅ Infrastructure-based cost |
| Privacy concerns | ✅ Full data ownership |

---

## ⚡ Key Features

- 🗄️ No-Code + Developer Friendly  
- 🔌 REST API (OpenAPI/Swagger)  
- 🧩 Microservices Architecture  
- 🏢 Multi-Tenant Workspaces  
- ⚡ Dynamic Schema (no migrations)  
- 🔐 Enterprise Security (RBAC, audit logs)  

---
## Services Architecture

| Service | Description | Port |
|---------|-------------|------|
| **sereni-base** | Core REST API server | 8080 |
| **PostgreSQL** | Primary database | 5432 |
| **JWT Provider** | Authentication service | 8081 |
| **Email Service** | SMTP email notifications | 8082 |
| **Storage Provider** | File storage (MinIO/S3) | 8083 |
| **Antivirus Service** | ClamAV malware scanning | 8084 |
| **MinIO** | Object storage | 9000/9001 |
| **Base UI** | Frontend application | 5050 |

## Quick Start

### Prerequisites

| Requirement | Version | Installation |
|-------------|---------|--------------|
| **Docker** | 20.10+ | [Install Docker](https://docs.docker.com/get-docker/) |
| **Docker Compose** | 2.0+ | [Install Compose](https://docs.docker.com/compose/install/) |
| **Git** | Latest | [Install Git](https://git-scm.com/downloads) |
| **Make** | Latest | Windows: `choco install make` |
| **SMTP Access** | - | Gmail, SendGrid, Mailgun, or custom SMTP |

### Installation

```bash
# Step 1: Clone the repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Step 2: Run interactive setup wizard
make setup

# Alternative (without Make):
# Windows: .\setup-interactive.ps1
# Linux/macOS: ./setup-interactive.sh
```

The setup wizard will:
- Prompt for configuration (press Enter for defaults)
- Generate `.env` file
- Start all services with Docker Compose

### Access Points

| Service | URL |
|---------|-----|
| Frontend | `http://localhost:5050` |
| Backend API | `http://localhost:8080` |
| API Documentation | `http://localhost:8080/swagger/index.html` |
| MinIO Console | `http://localhost:9001` |

### Default Login

Default credentials are configured via environment variables. See `.env.example` for setup.

> **⚠️ Security:** Never use default credentials in production. Always configure secure values via environment variables.

## Commands Reference

| Command | Description |
|---------|-------------|
| `make setup` | Run interactive setup wizard |
| `make setup-y` | Run setup with default values (non-interactive) |
| `make up` | Start all services |
| `make down` | Stop services (preserve data) |
| `make down-all` | Stop services and remove volumes |
| `make logs` | View service logs |
| `make restart` | Restart all services |
| `make ps` | Show running services |
| `make status` | Show detailed service status |
| `make clean` | Remove containers (preserve data) |
| `make clean-all` | Full cleanup (containers + volumes + images) |

## Documentation

| Document | Description |
|----------|-------------|
| [Complete Setup Guide](build/SETUP_COMPLETE_GUIDE.md) | Comprehensive beginner guide |
| [Setup Reference](build/SETUP.md) | Quick reference setup guide |
| [Interactive Setup](INTERACTIVE_SETUP_README.md) | Setup wizard documentation |
| [Environment Variables](docs/ENVIRONMENT_VARIABLES.md) | Configuration reference |

## Security

See [SECURITY.md](SECURITY.md) for reporting vulnerabilities.

## Contributing

We welcome contributions! See our contribution guidelines for details.

## License

MIT License. Copyright (c) 2026 Aptlogica Technologies.
