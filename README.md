# SereniBase - Open-Source No-Code Database Platform

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go">
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

> **Build and manage databases visually, no code required.** SereniBase is a production-ready, open-source platform for creating and managing business data with a spreadsheet-like interface. Self-host on your own infrastructure with full data control.

## Overview

**SereniBase** is an open-source alternative to Airtable, Notion databases, and NocoDB - a no-code platform that lets teams build custom databases, workflows, and applications without writing code. It provides a REST API backend that powers a React frontend, enabling users to create workspaces, design database schemas, and manage data through an intuitive spreadsheet-like interface.

### Why SereniBase?

| Problem | Solution |
|---------|----------|
| Cloud-only SaaS with vendor lock-in | ✅ **100% Self-Hosted** - Deploy on your infrastructure |
| Limited customization and extensibility | ✅ **Open Source** - MIT licensed, fork and customize |
| Expensive as data and users scale | ✅ **Zero Per-User Costs** - Pay only for infrastructure |
| Privacy concerns with sensitive data | ✅ **Complete Data Control** - Your data never leaves your servers |

## Key Features

- **No-Code Database Management**: Create tables, define columns, add relationships through visual interface
- **Multi-Tenant Architecture**: Workspaces provide complete isolation for organizations and teams
- **Dynamic Schema**: Add/remove tables and columns at runtime without database migrations
- **RESTful API**: Complete REST API with Swagger/OpenAPI documentation
- **Microservices Architecture**: Modular services for authentication, email, storage, and antivirus
- **Production-Ready**: RBAC, audit logging, connection pooling, health checks, and testing

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

| Field | Value |
|-------|-------|
| Email | `admin@example.com` |
| Password | `Admin@123` |

> **⚠️ Important:** Change the default password immediately after first login.

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

