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

<h1 align="center">SereniBase</h1>

<p align="center">
  <strong>A modern, open-source platform for creating and managing business data</strong>
</p>

<p align="center">
  <a href="#quick-start">Quick Start</a> |
  <a href="#features">Features</a> |
  <a href="#documentation">Documentation</a> |
  <a href="#contributing">Contributing</a> |
  <a href="#community">Community</a>
</p>

---

## Features

- Zero-config setup with an interactive wizard
- Modern React UI with responsive layout
- Secure authentication with JWT and role-based access control
- Built-in email service with SMTP support
- Flexible storage: Local, MinIO (Docker/Custom), and AWS S3
- Integrated ClamAV antivirus scanning
- One-command deployment with Docker Compose
- Cross-platform support for Windows, macOS, and Linux
- Auto-generated Swagger/OpenAPI docs
- Environment-driven configuration
- Safe updates that preserve existing configuration

---

## Quick Start

Get SereniBase running in under 5 minutes. No manual configuration needed.

### Prerequisites

| Requirement | Version | Installation Guide |
|-------------|---------|-------------------|
| Docker | v20.10+ | [Install Docker](https://docs.docker.com/get-docker/) |
| Docker Compose | v2.0+ | [Install Compose](https://docs.docker.com/compose/install/) |
| Chocolatey (Windows) | Latest | Install: https://chocolatey.org/install |
| Make (GNU Make) | Latest | Windows: `choco install make` |
| Git | Latest | [Install Git](https://git-scm.com/downloads) |
| SMTP access | Optional | Required for email notifications |

### Installation

```bash
# Clone the repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Run the interactive setup wizard
make setup
```

### Setup Wizard

The wizard prompts for:

1. Database configuration (default PostgreSQL or custom)
2. Authentication (auto-generate JWT secret or provide your own)
3. Email (SMTP configuration)
4. Storage (Local, MinIO, or AWS S3)
5. Network (public host/domain)
6. Admin account credentials

Press Enter to use recommended defaults at any prompt.

### Access the Application

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:5050 | Main application |
| API | http://localhost:8080 | Backend REST API |
| API Docs | http://localhost:8080/swagger/index.html | Swagger documentation |
| MinIO | http://localhost:9001 | Object storage console |

### Default Credentials

```
Email:    admin@example.com
Password: Admin@123
```

Security note: Change the default credentials before deploying to production.

---

## Documentation

| Document | Description |
|----------|-------------|
| [Docs Overview](docs/README.md) | Index of all documentation |
| [Environment Configuration](docs/ENV_CONFIGURATION.md) | Complete guide to environment variables |
| [Environment Setup Guide](docs/ENVIRONMENT_SETUP_GUIDE.md) | Setup flow and examples |
| [Environment Variables Reference](docs/ENVIRONMENT_VARIABLES.md) | Quick reference for variables |
| [Email Configuration](docs/EMAIL_CONFIGURATION.md) | SMTP and email service setup |
| [Role Access Guide](docs/ROLE_ACCESS_GUIDE.md) | Roles and permissions |
| [API Response Codes](docs/API_RESPONSE_CODES.md) | Standard API responses |
| [Advanced Setup](docs/ADVANCED_SETUP.md) | Advanced scenarios and customization |
| [Setup System Summary](docs/SETUP_SYSTEM_SUMMARY.md) | What the setup system does |
| [API Documentation](http://localhost:8080/swagger/index.html) | Interactive API reference (requires running server) |

---

## Development

### Project Structure

```
sereni-base/
|- build/                     build and setup files
|  |- config/.env.example      environment template
|  |- scripts/                 setup and utility scripts
|- docs/                       documentation
|- services/                   microservices (auto-cloned)
|- internal/                   backend source
|- cmd/                        application entry points
|- docker-compose.all.yaml     full stack deployment
|- Makefile                    build automation
|- .env                        local configuration
```

---

## Architecture

Core services:

- Frontend (React) on port 5050
- Backend (Go/Gin) on port 8080
- Auth service on port 8081
- Email service on port 8082
- Storage service on port 8083
- Antivirus service on port 8084
- PostgreSQL on port 5432
- MinIO on port 9000 (console on 9001)
- ClamAV on port 3310

---

## Contributing

We welcome contributors of all skill levels.

### Ways to Contribute

- Report bugs by opening an issue
- Suggest features via Discussions
- Improve documentation
- Submit PRs

### Getting Started

1. Fork the repository
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/sereni-base.git
   ```
3. Create a feature branch:
   ```bash
   git checkout -b feature/amazing-feature
   ```
4. Make your changes
5. Test your changes:
   ```bash
   make rebuild
   ```
6. Commit with a clear message:
   ```bash
   git commit -m "feat: add amazing feature"
   ```
7. Push to your fork:
   ```bash
   git push origin feature/amazing-feature
   ```
8. Open a Pull Request

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

## Security

### Before Production Deployment

- [ ] Change `AUTH_JWT_SECRET` to a strong random value
- [ ] Change all default passwords
- [ ] Configure HTTPS/SSL
- [ ] Update `CORS_ALLOWED_ORIGINS`
- [ ] Enable database SSL mode
- [ ] Remove development tokens

### Reporting Vulnerabilities

Found a security issue? Please email `security@aptlogica.com` instead of opening a public issue.

---

## Troubleshooting

<details>
<summary><strong>CORS Errors</strong></summary>

Update `CORS_ALLOWED_ORIGINS` in your `.env`:
```bash
CORS_ALLOWED_ORIGINS=http://YOUR_IP:5050,http://localhost:5050
```
Then restart: `make rebuild`
</details>

<details>
<summary><strong>Services Not Starting</strong></summary>

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
<summary><strong>Database Connection Failed</strong></summary>

```bash
# Check if PostgreSQL is running
docker compose -f docker-compose.all.yaml ps postgres

# View database logs
docker compose -f docker-compose.all.yaml logs postgres
```
</details>

<details>
<summary><strong>Port Already in Use</strong></summary>

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

## Community

- Issue Tracker: https://github.com/aptlogica/sereni-base/issues
- Discussions: https://github.com/aptlogica/sereni-base/discussions
- Email Support: support@aptlogica.com

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

<p align="center">
  <a href="https://github.com/aptlogica/sereni-base/stargazers">
    <img src="https://img.shields.io/github/stars/aptlogica/sereni-base?style=social" alt="GitHub Stars">
  </a>
</p>

<p align="center">
  Made with care by <a href="https://github.com/aptlogica">Aptlogica</a> and <a href="https://github.com/aptlogica/sereni-base/graphs/contributors">contributors</a>
</p>
