# SereniBase


### Open-Source Self-Hosted Backend Platform — PostgreSQL + Auto REST APIs + JWT + Storage + ClamAV

A production-ready, self-hosted backend platform built on PostgreSQL. Every table you create automatically
generates a documented REST API. JWT authentication, S3/RustFS file storage with ClamAV malware scanning, and
SMTP email delivery run as independent microservices — deploy together with a single `docker-compose up`, or
standalone as needed. Licensed under the Apache License 2.0. Full data sovereignty.



---

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.2+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-18+-61DAFB?style=for-the-badge&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-15+-4169E1?style=for-the-badge&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <a href="https://gin-gonic.com/"><img src="https://img.shields.io/badge/Gin-Framework-008ECF?style=for-the-badge&logo=gin&logoColor=white" alt="Gin"></a>
  <img src="https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/RustFS-Storage-EF2D5E?style=for-the-badge&logo=RustFS&logoColor=white" alt="RustFS">
  <img src="https://img.shields.io/badge/ClamAV-Antivirus-3776AB?style=for-the-badge&logo=clamav&logoColor=white" alt="ClamAV">
  <a href="https://swagger.io/"><img src="https://img.shields.io/badge/Swagger-Documented-85EA2D?style=for-the-badge&logo=swagger&logoColor=black" alt="Swagger"></a>
  
</p>
<p align="center">
  <img src="https://github.com/aptlogica/sereni-base/actions/workflows/ci.yml/badge.svg" alt="Github Actions">
  <img src="https://github.com/aptlogica/sereni-base/actions/workflows/github-code-scanning/codeql/badge.svg" alt="CodeQL">
  <a href="https://www.bestpractices.dev/projects/12425"><img src="https://www.bestpractices.dev/projects/12425/badge"></a>
  <img src="https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat" alt="Go Report">
  <a href="https://app.fossa.com/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base?ref=badge_shield&issueType=security" alt="FOSSA Status"><img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base.svg?type=shield&issueType=security"/></a>
  <a href="https://app.fossa.com/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base?ref=badge_shield&issueType=license" alt="FOSSA Status"><img src="https://app.fossa.com/api/projects/git%2Bgithub.com%2Faptlogica%2Fsereni-base.svg?type=shield&issueType=license"/></a>
  <a href="https://sonarcloud.io/summary/new_code?id=aptlogica_sereni-base">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=aptlogica_sereni-base&metric=alert_status" alt="Quality Gate"></a>
  <a href="https://sonarcloud.io/summary/new_code?id=aptlogica_sereni-base">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=aptlogica_sereni-base&metric=coverage" alt="Coverage">
  </a>
  <a href="https://sonarcloud.io/summary/new_code?id=aptlogica_sereni-base">
    <img src="https://sonarcloud.io/api/project_badges/measure?project=aptlogica_sereni-base&metric=security_rating" alt="Security">
  </a>
</p>
<p align="center">
  <img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/PRs-Welcome-brightgreen.svg?style=flat-square" alt="PRs Welcome">
  <img src="https://img.shields.io/badge/Status-Beta-orange.svg?style=flat-square" alt="Beta">
</p>


---
## ?? Live Demo

![SereniBase Demo](https://assets.aptlogica.com/sereni-base-demo.gif)


<a href="https://demo.serenibase.com/login"><img src="https://img.shields.io/badge/Demo-Live%20Demo-blue?style=for-the-badge" alt="Live Demo"></a> 

**Try the live demo:** [https://demo.serenibase.com/login](https://demo.serenibase.com/login)<br>_See SereniBase in action with all features enabled._ 

### Default login credentials

| Field | Value |
|-------|-------|
| Email | admin@example.com |
| Password | Admin@123 |
| Live Demo (no install) | https://demo.serenibase.com |

? **If this project helps you, please consider giving it a star!**  
?? https://github.com/aptlogica/sereni-base

---

## ?? What is [SereniBase](https://www.aptlogica.com/sereni-base/)?

A backend operating system. Not another tool.

Instead of stitching together separate services, developers get a unified backend where database, APIs, auth, storage, email, and security are already designed to work together.

- Create a table ? get a REST API. Instantly. No code.
- JWT auth, S3 storage, SMTP email, ClamAV scanning — all running as microservices.
- Integrate seamlessly with your existing systems via REST API or TypeScript SDK.
- Visual UI for teams + TypeScript SDK for developers.
- Self-hosted or cloud-ready. Full data sovereignty. Zero vendor lock-in.

![SereniBase Demo](https://assets.aptlogica.com/exploreAPI.jpg)
![SereniBase Demo](https://assets.aptlogica.com/APIcurl.jpg)

---

## ?? Why SereniBase?

Most no-code tools work… until they don’t.

| Problem | SereniBase Solution |
|--------|-------------------|
| Vendor lock-in | ? Self-hosted |
| Limited extensibility | ? Open-source & modular |
| Expensive scaling | ? Infrastructure-based cost |
| Privacy concerns | ? Full data ownership |
| No built-in file security | ? ClamAV antivirus scans every upload before storage |
| Separate email service needed | ? SMTP microservice with Redis queue and retry — included |
| Assembling disconnected tools | ? Database + API + Auth + Storage + Email + Security — one stack |

---


## ? Key Features

- ??? No-Code + Developer Friendly  
- ?? REST API (OpenAPI/Swagger)  
- ?? Microservices Architecture  
- ?? Multi-Tenant Workspaces  
- ? Dynamic Schema (no migrations)  
- ?? Enterprise Security (RBAC, audit logs)  

---
## Services Architecture

| Service | Description | Port |
|---------|-------------|------|
| **sereni-base** | Core REST API server | 8080 |
| **PostgreSQL** | Primary database | 5432 |
| **JWT Provider** | Authentication service | 8081 |
| **Email Service** | SMTP email notifications | 8082 |
| **Storage Provider** | File storage (RustFS/S3) | 8083 |
| **Antivirus Service** | ClamAV malware scanning | 8084 |
| **RustFS** | Object storage | 9000/9001 |
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
| RustFS Console | `http://localhost:9001` |

### Default Login

Default credentials are configured via environment variables. See `.env.example` for setup.

> **?? Security:** Never use default credentials in production. Always configure secure values via environment variables.

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

## Ecosystem

SereniBase is the core of a full backend platform. All modules are open-source and can be used independently or together:

| Module | Purpose | License |
|--------|---------|---------|
| [sereni-jwt-provider](https://github.com/aptlogica/sereni-jwt-provider) | JWT auth microservice | Apache 2.0 |
| [sereni-storage-provider](https://github.com/aptlogica/sereni-storage-provider) | S3/RustFS/local storage | Apache 2.0 |
| [sereni-email-smtp](https://github.com/aptlogica/sereni-email-smtp) | SMTP email + Redis queue | Apache 2.0 |
| [sereni-antivirus-clamav](https://github.com/aptlogica/sereni-antivirus-clamav) | ClamAV file scanning | Apache 2.0 |
| [go-postgres-rest](https://github.com/aptlogica/go-postgres-rest) | PostgreSQL REST API lib | Apache 2.0 |
| [base-sdk](https://github.com/aptlogica/base-sdk) | TypeScript SDK | Apache 2.0 |
| [base-ui](https://github.com/aptlogica/base-ui) | React frontend | MIT |

## License

Licensed under the Apache License, Version 2.0. Copyright 2026-2030 Aptlogica Technologies Pvt Ltd.

