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

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
- [Core Concepts](#core-concepts)
- [API Documentation](#api-documentation)
- [Usage Examples](#usage-examples)
- [Microservices](#microservices)
- [Security](#security)
- [Deployment](#deployment)
- [Development](#development)
- [Testing](#testing)
- [Troubleshooting](#troubleshooting)
- [Performance](#performance)
- [Best Practices](#best-practices)
- [Migration Guide](#migration-guide)
- [FAQ](#faq)
- [Contributing](#contributing)
- [License](#license)

## Overview

**SereniBase** is an open-source alternative to Airtable, Notion databases, and NocoDB - a no-code platform that lets teams build custom databases, workflows, and applications without writing code. It provides a REST API backend that powers a React frontend, enabling users to create workspaces, design database schemas, and manage data through an intuitive spreadsheet-like interface.

### Key Characteristics

- **No-Code Database Management**: Create tables, define columns, add relationships - all through a visual interface
- **Multi-Tenant Architecture**: Workspaces provide complete isolation for organizations and teams
- **Dynamic Schema**: Add/remove tables and columns at runtime without database migrations
- **RESTful API**: Complete REST API for all operations with Swagger/OpenAPI documentation
- **Microservices Architecture**: Modular services for authentication, email, storage, and antivirus scanning
- **Self-Hosted**: Deploy on your own infrastructure with Docker - you own your data
- **Production-Ready**: RBAC, audit logging, connection pooling, health checks, and comprehensive testing

### Why SereniBase?

**Problem:** Most no-code database platforms are:
- Cloud-only SaaS with vendor lock-in
- Limited customization and extensibility
- Expensive as data and users scale
- Privacy concerns with sensitive business data

**Solution:** SereniBase provides:
- ✅ **100% Self-Hosted** - Deploy on your infrastructure
- ✅ **Open Source** - MIT licensed, fork and customize as needed
- ✅ **Zero Per-User Costs** - Pay only for infrastructure
- ✅ **Complete Data Control** - Your data never leaves your servers
- ✅ **Microservices Architecture** - Scale components independently
- ✅ **REST API First** - Build custom integrations and automations
- ✅ **Enterprise Features** - RBAC, audit logs, SSO-ready

### Use Cases

| Use Case | Description |
|----------|-------------|
| **CRM Systems** | Build custom customer relationship management databases with contacts, deals, and pipelines |
| **Project Management** | Create project tracking systems with tasks, milestones, and team assignments |
| **Content Management** | Manage blog posts, media assets, and publishing workflows |
| **Inventory Management** | Track products, stock levels, suppliers, and purchase orders |
| **HR Systems** | Employee databases with onboarding workflows and document management |
| **Education Platforms** | Student databases, course management, and grade tracking |
| **Event Management** | Registrations, attendee lists, and event scheduling |
| **Internal Tools** | Replace spreadsheets with proper databases for team workflows |

## Features

### 🗄️ Database Management

**Dynamic Table Creation**
- Create unlimited tables with custom schemas
- Support for 25+ field types (text, number, date, dropdown, attachments, etc.)
- Real-time schema updates without downtime
- Foreign key relationships (one-to-one, one-to-many, many-to-many)
- Automatic index creation for performance

**Advanced Data Types**
- Text (single line, multi-line, rich text)
- Number (integer, decimal, currency, percentage)
- Date & Time (date, datetime, time)
- Boolean (checkbox)
- Select (single select, multi-select)
- User (single user, multiple users)
- File attachments (with antivirus scanning)
- URL, Email, Phone
- Formula fields
- Lookup fields
- Rollup/aggregation fields

**Query Capabilities**
- Complex filtering (AND/OR logic, 15+ operators)
- Multi-level sorting
- Pagination
- Full-text search
- Aggregations (COUNT, SUM, AVG, MIN, MAX)
- Joins across tables
- Saved views
- Export to CSV/JSON

### 👥 Multi-Tenancy & Workspaces

**Workspace Management**
- Create unlimited workspaces for complete data isolation
- Each workspace has independent users, bases, and permissions
- Workspace-level settings and branding
- Cross-workspace user assignments

**Base Management**
- Multiple bases (databases) per workspace
- Each base contains related tables
- Base-level permissions
- Import/export entire bases

### 🔐 Authentication & Authorization

**Authentication**
- JWT-based authentication via dedicated microservice
- Access tokens (15 minutes) and refresh tokens (7 days)
- Secure password hashing with bcrypt
- Password reset via email
- Session management

**Role-Based Access Control (RBAC)**
- Workspace-level roles (Owner, Admin, Member, Guest)
- Base-level permissions
- Table-level permissions
- Row-level permissions (coming soon)
- Custom role creation
- Fine-grained permission matrix

**User Management**
- User profiles with avatars
- Invite users via email
- Activate/deactivate users
- Audit logs for user actions
- Multi-workspace user support

### 📧 Communication

**Email Service Integration**
- SMTP email delivery via dedicated microservice
- Transactional emails (welcome, password reset, invitations)
- Custom email templates
- Multiple SMTP provider support (Gmail, SendGrid, Mailgun, etc.)
- Email queue and retry logic

### 📁 File Storage & Management

**Multi-Backend Storage**
- **Local Storage**: File system storage for development
- **MinIO**: S3-compatible self-hosted object storage
- **AWS S3**: Cloud storage integration
- Storage abstraction layer for easy provider switching

**File Features**
- File upload with size limits
- File attachments on any table
- Image resizing and optimization
- Antivirus scanning (ClamAV integration)
- Secure signed URLs
- Automatic cleanup of orphaned files

### 🛡️ Security

**Antivirus Protection**
- Real-time file scanning with ClamAV
- Quarantine infected files
- Scan results logging
- Configurable scan policies

**Security Features**
- HTTPS/TLS support
- CORS configuration
- Request rate limiting
- SQL injection prevention
- XSS protection
- CSRF protection
- Secure password policies
- JWT token rotation

### 🚀 Developer Experience

**REST API**
- Complete REST API for all operations
- Swagger/OpenAPI 3.0 documentation
- API versioning (/api/v1)
- Standard HTTP methods (GET, POST, PUT, PATCH, DELETE)
- JSON request/response format
- Error handling with meaningful messages
- Request ID tracing

**SDK & Integration**
- TypeScript SDK (gopostgrest-sdk)
- JavaScript/Node.js support
- Python SDK (coming soon)
- Webhook support (coming soon)
- Zapier integration (coming soon)

**Developer Tools**
- Interactive setup wizard
- Environment-based configuration
- Docker Compose for local development
- Hot reload in development
- Comprehensive logging
- Postman collection included

### 📊 Views & Visualization

**Multiple View Types**
- **Grid View**: Spreadsheet-like table view
- **Form View**: Data entry forms
- **Gallery View**: Card-based view for visual data
- **Kanban View**: Drag-and-drop boards
- **Calendar View**: Date-based calendar
- **Timeline View**: Gantt-style timeline
- Custom view filters and sorting
- Saved view configurations

### 🔌 Extensibility

**Plugin System**
- Custom field types
- Custom view types
- API extensions
- Webhook handlers
- Custom authentication providers

## Architecture

### System Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                         CLIENT LAYER                              │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │         base-ui (React Frontend)                         │    │
│  │  - Spreadsheet UI - User Management - Forms              │    │
│  │  - Views & Filters - Authentication UI                   │    │
│  └─────────────────────┬────────────────────────────────────┘    │
└────────────────────────┼──────────────────────────────────────────┘
                         │ HTTPS/REST API
┌────────────────────────▼──────────────────────────────────────────┐
│                      API GATEWAY LAYER                            │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │               Nginx Reverse Proxy                        │    │
│  │  - SSL Termination - Load Balancing - Rate Limiting      │    │
│  └─────────────────────┬────────────────────────────────────┘    │
└────────────────────────┼──────────────────────────────────────────┘
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼────────┐ ┌────▼────────┐ ┌────▼──────────────┐
│  SereniBase    │ │ JWT Provider│ │ Email Service     │
│  (REST API)    │ │ Service     │ │ (SMTP)            │
│                │ │             │ │                   │
│ • Workspaces   │ │ • JWT Signing│ • Email Sending   │
│ • Bases        │ │ • Validation│ • Templates       │
│ • Tables       │ │ • Refresh   │ • Queue           │
│ • CRUD Ops     │ │             │                   │
│ • RBAC         │ └─────────────┘ └───────────────────┘
└────────┬───────┘
         │
    ┌────┼────────────────────┐
    │    │                    │
┌───▼────▼───┐ ┌─────────────▼──┐ ┌──────────────────┐
│ PostgreSQL │ │ Storage Service│ │ Antivirus Service│
│            │ │                │ │ (ClamAV)         │
│ • Tables   │ │ • Local Storage│ │                  │
│ • Schemas  │ │ • MinIO/S3     │ │ • File Scanning  │
│ • Users    │ │ • File Mgmt    │ │ • Quarantine     │
└────────────┘ └────────────────┘ └──────────────────┘
```

### Service Communication

```
┌─────────────────────────────────────────────────────────────┐
│                    SereniBase REST API                      │
│                        (Port 8080)                          │
└──────────┬────────┬──────────┬─────────┬───────────────────┘
           │        │          │         │
   ┌───────▼───┐  ┌─▼────────┐ ┌────────▼──┐  ┌──────────▼───┐
   │ JWT       │  │ Email    │ │ Storage   │  │ Antivirus    │
   │ Provider  │  │ Service  │ │ Provider  │  │ Service      │
   │ :8081     │  │ :8082    │ │ :8083     │  │ :8084        │
   └───────────┘  └──────────┘ └───────────┘  └──────────────┘
           HTTP REST Communication
```

### Technology Stack

**Backend Services**
- **Go 1.24+**: Main programming language
- **Gin Framework**: HTTP web framework with excellent performance
- **go-postgres-rest**: Custom PostgreSQL abstraction library
- **PostgreSQL 15+**: Primary database with JSONB support
- **JWT (golang-jwt/jwt)**: Authentication tokens
- **bcrypt**: Password hashing
- **Swagger/OpenAPI**: API documentation generation

**Frontend**
- **React 18+**: UI framework
- **TypeScript**: Type-safe JavaScript
- **Vite**: Build tool and dev server
- **TanStack Query**: Data fetching and caching
- **Tailwind CSS**: Utility-first CSS framework
- **gopostgrest-sdk**: TypeScript SDK for SereniBase API

**Infrastructure**
- **Docker & Docker Compose**: Containerization
- **Nginx**: Reverse proxy and load balancer
- **MinIO**: S3-compatible object storage
- **ClamAV**: Antivirus scanning

**Development Tools**
- **Zerolog**: Structured logging
- **Testify**: Testing framework
- **go-mock**: Mocking for unit tests
- **Postman**: API testing
- **SonarQube**: Code quality and security analysis

## Quick Start

### Prerequisites

| Requirement | Version | Installation Guide |
|-------------|---------|-------------------|
| **Docker** | 20.10+ | [Install Docker](https://docs.docker.com/get-docker/) |
| **Docker Compose** | 2.0+ | [Install Compose](https://docs.docker.com/compose/install/) |
| **Git** | Latest | [Install Git](https://git-scm.com/downloads) |
| **Make** | Latest | Windows: `choco install make` |
| **SMTP Access** | - | Gmail, SendGrid, Mailgun, or custom SMTP |

### 5-Minute Setup

```bash
# Step 1: Clone the repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Step 2: Run interactive setup wizard
make setup

# Alternative (without Make):
# Windows: .\setup-interactive.ps1
# Linux/macOS: ./setup-interactive.sh

# The wizard will:
# - Prompt for configuration (press Enter for defaults)
# - Generate .env file
# - Start all services with Docker Compose

# Step 3: Access the application
# Frontend: http://localhost:5050
# API: http://localhost:8080
# API Docs: http://localhost:8080/swagger/index.html
```

### First Login

After setup completes, log in with your admin account:

**Default credentials** (if you used defaults in setup):
- Email: `admin@example.com`
- Password: `Admin@123`

**⚠️ Important:** Change the default password immediately after first login.

### Quick Commands

```bash
# Start all services
make up

# Stop services (data preserved)
make down

# View logs
make logs

# Restart services
make restart

# Show service status
make ps

# Full cleanup (removes all data!)
make clean-all
```

## Installation

### Option 1: Docker Compose (Recommended)

**Production Deployment:**

```bash
# Clone repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Run interactive setup
make setup

# Configure production settings in .env:
# - Change default passwords
# - Configure your domain
# - Set up HTTPS/TLS
# - Configure SMTP for production
# - Set appropriate connection limits

# Start services
make up
```

**Development Environment:**

```bash
# Clone repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Copy example environment file
cp .env.example .env

# Edit .env with your settings
nano .env

# Start services
docker compose -f docker-compose.all.yaml up -d

# View logs
docker compose logs -f serenibase
```

### Option 2: Manual Installation

**Prerequisites:**
- Go 1.24+
- PostgreSQL 15+
- Node.js 18+ (for frontend)

**Backend Setup:**

```bash
# Clone repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Install Go dependencies
go mod download

# Configure database
createdb serenibase
psql serenibase < init.sql

# Create .env file
cp .env.example .env
nano .env  # Edit with your settings

# Build
go build -o serenibase ./cmd/server

# Run
./serenibase
```

**Frontend Setup:**

```bash
# Clone base-ui
git clone https://github.com/aptlogica/base-ui.git
cd base-ui

# Install dependencies
npm install

# Configure API endpoint
cp .env.example .env
echo "VITE_API_URL=http://localhost:8080" >> .env

# Build
npm run build

# Serve (production)
npm run preview

# Or development mode
npm run dev
```

## Configuration

### Environment Variables

Create `.env` file in the project root:

```dotenv
# ==============================================================================
#                         SERENIBASE CONFIGURATION
# ==============================================================================

# ------------------------------------------------------------------------------
#                           NETWORK CONFIGURATION
# ------------------------------------------------------------------------------

PUBLIC_HOST=localhost                    # Your domain (e.g., serenibase.example.com)

# ------------------------------------------------------------------------------
#                           SERVER CONFIGURATION
# ------------------------------------------------------------------------------

SERVER_HOST=0.0.0.0                      # Bind address
SERVER_PORT=8080                         # API port
SERVER_READ_TIMEOUT=30                   # Request read timeout (seconds)
SERVER_WRITE_TIMEOUT=30                  # Response write timeout (seconds)
SERVER_ENV=production                    # Environment: dev, staging, production
SERVER_SCHEME=https                      # http or https

# ------------------------------------------------------------------------------
#                           DATABASE CONFIGURATION
# ------------------------------------------------------------------------------

DATABASE_HOST=postgres                   # Database host
DATABASE_PORT=5432                       # Database port
DATABASE_USER=postgres                   # Database user
DATABASE_PASSWORD=YOUR_SECURE_PASSWORD   # Database password (change this!)
DATABASE_NAME=serenibase                 # Database name
DATABASE_SSL_MODE=require                # SSL mode: disable, require, verify-ca, verify-full
DATABASE_MAX_OPEN_CONNS=100              # Max concurrent connections
DATABASE_MAX_IDLE_CONNS=25               # Max idle connections
DATABASE_CONN_MAX_LIFETIME=1h            # Connection lifetime

# ------------------------------------------------------------------------------
#                           AUTHENTICATION CONFIGURATION
# ------------------------------------------------------------------------------

AUTH_URL=http://jwt-provider:8081                  # JWT provider service URL
AUTH_RESET_PASSWORD_URL=http://localhost:5050/reset-password?token=%s
AUTH_JWT_SECRET=CHANGE_THIS_TO_RANDOM_256_BIT_KEY  # JWT signing secret (change this!)
ACCESS_TOKEN_DURATION=15m                          # Access token lifetime
REFRESH_TOKEN_DURATION=168h                        # Refresh token lifetime (7 days)
AUTH_PORT=8081
AUTH_HOST=0.0.0.0
AUTH_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050
AUTH_ENV=production
AUTH_LOG_LEVEL=info

# ------------------------------------------------------------------------------
#                           ADMIN ACCOUNT
# ------------------------------------------------------------------------------

OWNER_FIRST_NAME=Admin                   # Admin first name
OWNER_LAST_NAME=User                     # Admin last name
OWNER_EMAIL=admin@example.com            # Admin email (change this!)
OWNER_PASSWORD=CHANGE_THIS_PASSWORD      # Admin password (change this!)
TEMPORARY_USER_PASSWORD=CHANGE_THIS      # Default password for new users

# ------------------------------------------------------------------------------
#                           EMAIL CONFIGURATION
# ------------------------------------------------------------------------------

EMAIL_URL=http://email-service:8082/api/v1/email
EMAIL_HOST=0.0.0.0
EMAIL_PORT=8082
EMAIL_ALLOWED_ORIGIN=http://localhost:8080,http://localhost:5050
EMAIL_SMTP_HOST=smtp.gmail.com           # SMTP server
EMAIL_SMTP_PORT=587                      # SMTP port (587 for TLS, 465 for SSL)
EMAIL_SMTP_USERNAME=your-email@gmail.com # SMTP username
EMAIL_SMTP_PASSWORD=your-app-password    # SMTP password (use app password for Gmail)
EMAIL_FROM_EMAIL=your-email@gmail.com    # From email address
EMAIL_FROM_NAME=SereniBase               # From name

# ------------------------------------------------------------------------------
#                           STORAGE CONFIGURATION
# ------------------------------------------------------------------------------

STORAGE_URL=http://sereni-storage-provider:8083/api/v1
STORAGE_SERVER_PORT=8083
STORAGE_SERVER_HOST=0.0.0.0
STORAGE_SERVER_SCHEME=http
STORAGE_DRIVER=minio                     # Options: local, minio, s3

# Local storage (development)
STORAGE_DEV_PATH=./uploads

# AWS S3 (production)
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=serenibase-files
STORAGE_AWS_ACCESS_KEY=your-aws-access-key
STORAGE_AWS_SECRET_KEY=your-aws-secret-key

# MinIO (self-hosted S3-compatible)
STORAGE_MINIO_ENDPOINT=minio:9000
STORAGE_MINIO_ACCESS_KEY=minioadmin      # Change in production
STORAGE_MINIO_SECRET_KEY=minioadmin      # Change in production
STORAGE_MINIO_BUCKET=serenibase
STORAGE_MINIO_USE_SSL=false
STORAGE_ALLOWED_ORIGINS=http://localhost:8080,http://localhost:5050

# ------------------------------------------------------------------------------
#                           ANTIVIRUS CONFIGURATION
# ------------------------------------------------------------------------------

ANTIVIRUS_URL=http://antivirus-service:8084/api/v1
ANTIVIRUS_HOST=0.0.0.0
ANTIVIRUS_PORT=8084
ANTIVIRUS_CLAMAV_HOST=clamav
ANTIVIRUS_CLAMAV_PORT=3310
ANTIVIRUS_ENABLED=true                   # Enable/disable antivirus scanning
ANTIVIRUS_ALLOWED_ORIGINS=http://localhost:8080,http://serenibase:8080

# ------------------------------------------------------------------------------
#                           CORS CONFIGURATION
# ------------------------------------------------------------------------------

CORS_ALLOWED_ORIGINS=http://localhost:5050,http://localhost:8080
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Request-ID
CORS_ALLOW_CREDENTIALS=true

# ------------------------------------------------------------------------------
#                           LOGGING
# ------------------------------------------------------------------------------

LOG_LEVEL=info                           # Levels: debug, info, warn, error
LOG_FORMAT=json                          # Formats: json, text
LOG_OUTPUT=stdout                        # Output: stdout, file
LOG_FILE_PATH=./logs/serenibase.log      # Log file path (if LOG_OUTPUT=file)
```

### Configuration for Different Environments

**Development (.env.development):**
```dotenv
SERVER_ENV=dev
SERVER_SCHEME=http
PUBLIC_HOST=localhost
DATABASE_SSL_MODE=disable
LOG_LEVEL=debug
ANTIVIRUS_ENABLED=false
REDIS_ENABLED=false
```

**Staging (.env.staging):**
```dotenv
SERVER_ENV=staging
SERVER_SCHEME=https
PUBLIC_HOST=staging.serenibase.com
DATABASE_SSL_MODE=require
LOG_LEVEL=info
ANTIVIRUS_ENABLED=true
```

**Production (.env.production):**
```dotenv
SERVER_ENV=production
SERVER_SCHEME=https
PUBLIC_HOST=serenibase.example.com
DATABASE_SSL_MODE=verify-full
DATABASE_MAX_OPEN_CONNS=200
LOG_LEVEL=warn
ANTIVIRUS_ENABLED=true
STORAGE_DRIVER=s3
```

## Core Concepts

### 1. Workspaces

**Workspaces** provide complete multi-tenant isolation. Each workspace is an independent environment with its own:
- Users and permissions
- Bases (databases)
- Data storage
- Settings and configuration

```
Workspace "Marketing Team"
├── Base "Content Calendar"
│   ├── Table "Blog Posts"
│   ├── Table "Social Media"
│   └── Table "Authors"
├── Base "Campaign Tracker"
│   ├── Table "Campaigns"
│   └── Table "Metrics"
└── Users
    ├── john@company.com (Owner)
    ├── sarah@company.com (Admin)
    └── mike@company.com (Member)
```

**Use Workspace for:**
- Separate departments (Marketing, Sales, Engineering)
- Different clients (if you're an agency)
- Development vs. Production environments
- Complete data isolation requirements

### 2. Bases

**Bases** are databases within a workspace. Think of a base as a collection of related tables that form a complete application.

```
Base "Project Management"
├── Table "Projects"
├── Table "Tasks"
├── Table "Team Members"
├── Table "Time Tracking"
└── Table "Milestones"
```

**Use Bases for:**
- Grouping related data (CRM base, Inventory base)
- Different applications within same workspace
- Logical separation of concerns
- Base-level permission boundaries

### 3. Tables

**Tables** are the core data structures. Each table has:
- **Columns** (fields with specific data types)
- **Rows** (individual records)
- **Views** (different ways to visualize data)
- **Permissions** (who can read/write)

```
Table "Projects"
┌────────┬──────────────┬─────────┬────────────┬──────────┐
│ ID     │ Name         │ Status  │ Due Date   │ Owner    │
├────────┼──────────────┼─────────┼────────────┼──────────┤
│ 1      │ Website      │ Active  │ 2026-04-15 │ John     │
│ 2      │ Mobile App   │ Planning│ 2026-05-20 │ Sarah    │
│ 3      │ Backend API  │ Done    │ 2026-03-01 │ Mike     │
└────────┴──────────────┴─────────┴────────────┴──────────┘
```

### 4. Field Types

SereniBase supports 25+ field types:

| Category | Field Types |
|----------|-------------|
| **Text** | Single line text, Multi-line text, Rich text, URL, Email, Phone |
| **Number** | Integer, Decimal, Currency, Percentage |
| **Date/Time** | Date, Date & Time, Time, Duration |
| **Select** | Single select, Multi-select |
| **Boolean** | Checkbox |
| **Files** | File attachment, Image |
| **Relationships** | Link to another table (foreign key) |
| **Users** | Single user, Multiple users |
| **Computed** | Formula, Lookup, Rollup |
| **Special** | Auto-number, Created time, Modified time, Created by, Modified by |

### 5. Views

**Views** are different ways to visualize the same table data:

- **Grid View**: Spreadsheet-like table (default)
- **Form View**: Data entry form
- **Gallery View**: Card-based visual layout
- **Kanban View**: Drag-and-drop board (grouped by status)
- **Calendar View**: Date-based calendar
- **Timeline View**: Gantt chart timeline

Each view can have:
- Custom filters
- Sort orders
- Hidden/shown columns
- Saved configurations

### 6. Relationships

**Relationships** connect tables together:

**One-to-Many:**
```
Users (1) ----< Projects (many)
One user can have many projects
```

**Many-to-Many:**
```
Projects (many) >----< Tags (many)
Projects can have many tags, tags can apply to many projects
```

**One-to-One:**
```
User (1) ----< Profile (1)
Each user has exactly one profile
```

### 7. Permissions

**Workspace Roles:**
- **Owner**: Full control over workspace
- **Admin**: Manage bases, users, and settings
- **Member**: Create/edit bases they have access to
- **Guest**: View-only access to specific bases

**Base Permissions:**
- **Edit**: Full CRUD access to base
- **Read**: View-only access
- **None**: No access

**Table Permissions:**
- **Create**: Can add new records
- **Read**: Can view records
- **Update**: Can modify records
- **Delete**: Can delete records

## API Documentation

### Swagger/OpenAPI Docs

**Access interactive API documentation:**
```
http://localhost:8080/swagger/index.html
```

Browse all endpoints, test API calls directly from the browser, and view request/response schemas.

### Authentication

All API requests (except login/register) require a JWT Bearer token:

```bash
# Login to get token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin@123"
  }'

# Response
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}

# Use token in subsequent requests
curl -X GET http://localhost:8080/api/v1/workspace/list \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."
```

### Core API Endpoints

#### Authentication

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | User login |
| `/api/v1/auth/register` | POST | User registration |
| `/api/v1/auth/logout` | POST | User logout |
| `/api/v1/auth/refresh` | POST | Refresh access token |
| `/api/v1/auth/forgot-password` | POST | Request password reset |
| `/api/v1/auth/reset-password` | POST | Reset password with token |
| `/api/v1/auth/verify-email` | POST | Verify email address |

#### Workspaces

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/workspace/create` | POST | Create new workspace |
| `/api/v1/workspace/list` | GET | List all workspaces |
| `/api/v1/workspace/:id` | GET | Get workspace details |
| `/api/v1/workspace/:id` | PATCH | Update workspace |
| `/api/v1/workspace/:id` | DELETE | Delete workspace |
| `/api/v1/workspace/:id/users` | GET | List workspace users |
| `/api/v1/workspace/:id/invite` | POST | Invite user to workspace |

#### Bases

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/base/create` | POST | Create new base |
| `/api/v1/base/list` | GET | List bases in workspace |
| `/api/v1/base/:id` | GET | Get base details |
| `/api/v1/base/:id` | PATCH | Update base |
| `/api/v1/base/:id` | DELETE | Delete base |
| `/api/v1/base/:id/tables` | GET | List tables in base |
| `/api/v1/base/:id/duplicate` | POST | Duplicate base |

#### Tables

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/table/create` | POST | Create new table |
| `/api/v1/table/list` | GET | List tables |
| `/api/v1/table/:id` | GET | Get table metadata |
| `/api/v1/table/:id` | PATCH | Update table |
| `/api/v1/table/:id` | DELETE | Delete table |
| `/api/v1/table/:id/columns` | GET | List table columns |
| `/api/v1/table/:id/column` | POST | Add column |
| `/api/v1/table/:id/column/:cid` | PATCH | Update column |
| `/api/v1/table/:id/column/:cid` | DELETE | Delete column |

#### Records (Rows)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/table/:id/records` | GET | Get all records (with filters) |
| `/api/v1/table/:id/record` | POST | Create record |
| `/api/v1/table/:id/record/:rid` | GET | Get single record |
| `/api/v1/table/:id/record/:rid` | PATCH | Update record |
| `/api/v1/table/:id/record/:rid` | DELETE | Delete record |
| `/api/v1/table/:id/records/bulk` | POST | Bulk create records |
| `/api/v1/table/:id/records/bulk` | PATCH | Bulk update records |
| `/api/v1/table/:id/records/bulk` | DELETE | Bulk delete records |

#### Users

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/user/profile/:id` | GET | Get user profile |
| `/api/v1/user/profile/:id` | PATCH | Update user profile |
| `/api/v1/user/profile/:id/avatar` | POST | Upload avatar |
| `/api/v1/user/profile/:id/avatar` | DELETE | Remove avatar |
| `/api/v1/user/workspaces` | GET | Get user workspaces |
| `/api/v1/user/access-details` | GET | Get user access details |
| `/api/v1/user/list` | GET | List all users |
| `/api/v1/user/create` | POST | Create user |
| `/api/v1/user/assign` | POST | Assign user to workspace |

#### Assets

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/asset/upload` | POST | Upload file |
| `/api/v1/asset/:id` | GET | Get file |
| `/api/v1/asset/:id` | DELETE | Delete file |
| `/api/v1/asset/list` | GET | List files |

## Usage Examples

### Authentication Flow

```bash
# 1. Register new user
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe"
  }'

# 2. Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "john@example.com",
    "password": "SecurePass123!"
  }'

# Save the access_token from response

# 3. Access protected endpoint
curl -X GET http://localhost:8080/api/v1/workspace/list \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# 4. Refresh token when access token expires
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }'
```

### Create Complete Workspace

```bash
# 1. Create workspace
curl -X POST http://localhost:8080/api/v1/workspace/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Marketing Team",
    "description": "Marketing department workspace"
  }'

# Response: {"id": "ws_123...", "name": "Marketing Team", ...}
WORKSPACE_ID="ws_123..."

# 2. Create base in workspace
curl -X POST http://localhost:8080/api/v1/base/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "workspace_id": "'$WORKSPACE_ID'",
    "name": "Content Calendar",
    "description": "Blog posts and social media planning"
  }'

# Response: {"id": "base_456...", "name": "Content Calendar", ...}
BASE_ID="base_456..."

# 3. Create table
curl -X POST http://localhost:8080/api/v1/table/create \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "base_id": "'$BASE_ID'",
    "name": "Blog Posts",
    "columns": [
      {
        "name": "Title",
        "type": "text",
        "required": true
      },
      {
        "name": "Status",
        "type": "single_select",
        "options": ["Draft", "Review", "Published"],
        "required": true
      },
      {
        "name": "Publish Date",
        "type": "date"
      },
      {
        "name": "Author",
        "type": "user"
      },
      {
        "name": "Tags",
        "type": "multi_select",
        "options": ["SEO", "Tutorial", "News", "Product"]
      }
    ]
  }'

# Response: {"id": "tbl_789...", "name": "Blog Posts", ...}
```

### CRUD Operations on Records

```bash
TABLE_ID="tbl_789..."

# CREATE record
curl -X POST http://localhost:8080/api/v1/table/$TABLE_ID/record \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "Title": "10 Tips for Better Database Design",
    "Status": "Draft",
    "Publish Date": "2026-04-15",
    "Tags": ["Tutorial", "SEO"]
  }'

# Response: {"id": "rec_abc...", "Title": "10 Tips...", ...}
RECORD_ID="rec_abc..."

# READ single record
curl -X GET http://localhost:8080/api/v1/table/$TABLE_ID/record/$RECORD_ID \
  -H "Authorization: Bearer $TOKEN"

# READ all records (with filtering)
curl -X GET "http://localhost:8080/api/v1/table/$TABLE_ID/records?filter=Status:Draft&sort=Publish%20Date:DESC&limit=50" \
  -H "Authorization: Bearer $TOKEN"

# UPDATE record
curl -X PATCH http://localhost:8080/api/v1/table/$TABLE_ID/record/$RECORD_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "Status": "Review"
  }'

# DELETE record
curl -X DELETE http://localhost:8080/api/v1/table/$TABLE_ID/record/$RECORD_ID \
  -H "Authorization: Bearer $TOKEN"
```

### Bulk Operations

```bash
# Bulk create records
curl -X POST http://localhost:8080/api/v1/table/$TABLE_ID/records/bulk \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "records": [
      {
        "Title": "Introduction to SereniBase",
        "Status": "Published",
        "Publish Date": "2026-03-01"
      },
      {
        "Title": "Advanced Query Techniques",
        "Status": "Draft",
        "Publish Date": "2026-03-15"
      },
      {
        "Title": "Building REST APIs",
        "Status": "Review",
        "Publish Date": "2026-03-20"
      }
    ]
  }'

# Bulk update records
curl -X PATCH http://localhost:8080/api/v1/table/$TABLE_ID/records/bulk \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"id": "rec_123", "Status": "Published"},
      {"id": "rec_456", "Status": "Published"},
      {"id": "rec_789", "Status": "Archived"}
    ]
  }'

# Bulk delete records
curl -X DELETE http://localhost:8080/api/v1/table/$TABLE_ID/records/bulk \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "record_ids": ["rec_123", "rec_456", "rec_789"]
  }'
```

### Advanced Filtering & Querying

```bash
# Complex filter with AND logic
# Get blog posts that are:
# - Status is "Published"
# - Publish Date after 2026-01-01
# - Tags include "Tutorial"
curl -X GET "http://localhost:8080/api/v1/table/$TABLE_ID/records" \
  -H "Authorization: Bearer $TOKEN" \
  -G \
  --data-urlencode "filter[Status][$eq]=Published" \
  --data-urlencode "filter[Publish Date][$gte]=2026-01-01" \
  --data-urlencode "filter[Tags][$contains]=Tutorial" \
  --data-urlencode "sort=Publish Date:DESC" \
  --data-urlencode "limit=20" \
  --data-urlencode "offset=0"

# Search in text fields
curl -X GET "http://localhost:8080/api/v1/table/$TABLE_ID/records" \
  -H "Authorization: Bearer $TOKEN" \
  -G \
  --data-urlencode "search=database design" \
  --data-urlencode "search_fields=Title,Content"

# Aggregations
curl -X GET "http://localhost:8080/api/v1/table/$TABLE_ID/aggregate" \
  -H "Authorization: Bearer $TOKEN" \
  -G \
  --data-urlencode "group_by=Status" \
  --data-urlencode "aggregates=count,avg(Views),sum(Shares)"
```

### File Attachments

```bash
# Upload file
curl -X POST http://localhost:8080/api/v1/asset/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@document.pdf" \
  -F "table_id=$TABLE_ID" \
  -F "record_id=$RECORD_ID" \
  -F "field_name=Attachments"

# Response includes file URL and metadata

# Attach file ID to record
curl -X PATCH http://localhost:8080/api/v1/table/$TABLE_ID/record/$RECORD_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "Attachments": ["asset_123...", "asset_456..."]
  }'
```

### User Management

```bash
# Invite user to workspace
curl -X POST http://localhost:8080/api/v1/workspace/$WORKSPACE_ID/invite \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "sarah@example.com",
    "role": "member",
    "message": "Join our marketing team workspace!"
  }'

# Assign user to workspace with specific permissions
curl -X POST http://localhost:8080/api/v1/user/assign \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "workspace_id": "'$WORKSPACE_ID'",
    "role": "admin"
  }'

# Update user permissions
curl -X PUT http://localhost:8080/api/v1/user/access/update \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "user_123",
    "workspace_id": "'$WORKSPACE_ID'",
    "base_permissions": {
      "'$BASE_ID'": "edit"
    }
  }'
```

## Microservices

SereniBase uses a microservices architecture for modularity and scalability:

### 1. JWT Provider Service

**Purpose:** Centralized JWT token management

**Responsibilities:**
- Generate access and refresh tokens
- Validate tokens
- Token revocation
- Key rotation

**Endpoints:**
- `POST /api/v1/jwt/generate` - Generate new token pair
- `POST /api/v1/jwt/validate` - Validate token
- `POST /api/v1/jwt/refresh` - Refresh access token
- `POST /api/v1/jwt/revoke` - Revoke token

**Configuration:**
```dotenv
AUTH_URL=http://jwt-provider:8081
AUTH_JWT_SECRET=your-256-bit-secret
ACCESS_TOKEN_DURATION=15m
REFRESH_TOKEN_DURATION=168h
```

### 2. Email Service

**Purpose:** SMTP email delivery

**Responsibilities:**
- Send transactional emails
- Email templates
- Queue management
- Retry logic

**Email Types:**
- Welcome emails
- Password reset
- User invitations
- Workspace notifications
- Security alerts

**Configuration:**
```dotenv
EMAIL_URL=http://email-service:8082/api/v1/email
EMAIL_SMTP_HOST=smtp.gmail.com
EMAIL_SMTP_PORT=587
EMAIL_SMTP_USERNAME=your-email@gmail.com
EMAIL_SMTP_PASSWORD=your-app-password
```

**Supported SMTP Providers:**
- Gmail
- SendGrid
- Mailgun
- Amazon SES
- Custom SMTP servers

### 3. Storage Provider Service

**Purpose:** File storage abstraction

**Responsibilities:**
- File upload/download
- Multiple backend support (Local, MinIO, S3)
- File metadata management
- Pre-signed URLs
- File cleanup

**Storage Backends:**

**Local Storage** (Development):
```dotenv
STORAGE_DRIVER=local
STORAGE_DEV_PATH=./uploads
```

**MinIO** (Self-hosted S3):
```dotenv
STORAGE_DRIVER=minio
STORAGE_MINIO_ENDPOINT=minio:9000
STORAGE_MINIO_ACCESS_KEY=minioadmin
STORAGE_MINIO_SECRET_KEY=minioadmin
STORAGE_MINIO_BUCKET=serenibase
```

**AWS S3** (Production):
```dotenv
STORAGE_DRIVER=s3
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=serenibase-files
STORAGE_AWS_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE
STORAGE_AWS_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

### 4. Antivirus Service

**Purpose:** File scanning with ClamAV

**Responsibilities:**
- Scan uploaded files for malware
- Quarantine infected files
- Scan result logging
- Signature updates

**Configuration:**
```dotenv
ANTIVIRUS_URL=http://antivirus-service:8084/api/v1
ANTIVIRUS_ENABLED=true
ANTIVIRUS_CLAMAV_HOST=clamav
ANTIVIRUS_CLAMAV_PORT=3310
```

**Features:**
- Real-time file scanning on upload
- Virus definition updates
- Scan result caching
- Configurable scan policies

## Security

### Authentication & Authorization

**JWT-Based Authentication:**
- Access tokens (short-lived: 15 minutes)
- Refresh tokens (long-lived: 7 days)
- Secure token storage
- Automatic token rotation

**Password Security:**
- Bcrypt hashing (cost factor 10)
- Password complexity requirements
- Password reset via email
- Account lockout after failed attempts

**Role-Based Access Control (RBAC):**
- Workspace-level roles
- Base-level permissions
- Table-level permissions
- Row-level permissions (coming soon)

### API Security

**HTTPS/TLS:**
```nginx
server {
    listen 443 ssl http2;
    server_name serenibase.example.com;
    
    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
}
```

**CORS Configuration:**
```dotenv
CORS_ALLOWED_ORIGINS=https://serenibase.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Content-Type,Authorization,X-Request-ID
CORS_ALLOW_CREDENTIALS=true
```

**Rate Limiting:**
```go
// Example rate limiting middleware
rateLimiter := middleware.RateLimit(100, time.Minute) // 100 req/min
r.Use(rateLimiter)
```

### Input Validation

**Validation on all inputs:**
- SQL injection prevention
- XSS protection
- CSRF protection
- Request size limits
- File type validation

**Example validation:**
```go
type CreateTableRequest struct {
    Name        string `json:"name" validate:"required,min=1,max=100"`
    Description string `json:"description" validate:"max=500"`
    BaseID      string `json:"base_id" validate:"required,uuid4"`
}
```

### Security Best Practices

**1. Change Default Credentials:**
```bash
# Never use these in production!
OWNER_EMAIL=admin@example.com
OWNER_PASSWORD=Admin@123
```

**2. Use Strong JWT Secrets:**
```bash
# Generate random 256-bit secret
openssl rand -base64 32

# Set in .env
AUTH_JWT_SECRET=generated_random_secret_here
```

**3. Enable HTTPS:**
```dotenv
SERVER_SCHEME=https
```

**4. Configure PostgreSQL SSL:**
```dotenv
DATABASE_SSL_MODE=verify-full
```

**5. Regular Security Updates:**
```bash
# Update Docker images
docker compose pull
docker compose up -d

# Update dependencies
go get -u ./...
go mod tidy
```

**6. Monitor Logs:**
```bash
# Watch for suspicious activity
docker compose logs -f serenibase | grep -i "error\|fail\|unauthorized"
```

## Deployment

### Docker Production Deployment

**1. Prepare environment:**

```bash
# Clone repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Create production .env
cp .env.example .env
nano .env  # Configure for production
```

**2. Update docker-compose for production:**

Create `docker-compose.prod.yaml`:

```yaml
version: "3.9"

services:
  serenibase:
    image: serenibase/serenibase:latest
    restart: always
    environment:
      - SERVER_ENV=production
      - SERVER_SCHEME=https
    env_file:
      - .env
    depends_on:
      - postgres
    networks:
      - serenibase-network
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2'
          memory: 2G

  postgres:
    image: postgres:15-alpine
    restart: always
    environment:
      - POSTGRES_USER=${DATABASE_USER}
      - POSTGRES_PASSWORD=${DATABASE_PASSWORD}
      - POSTGRES_DB=${DATABASE_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - serenibase-network
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G

  nginx:
    image: nginx:alpine
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - serenibase
    networks:
      - serenibase-network

volumes:
  postgres_data:

networks:
  serenibase-network:
    driver: bridge
```

**3. Deploy:**

```bash
# Build and start
docker compose -f docker-compose.prod.yaml up -d

# Check status
docker compose ps

# View logs
docker compose logs -f serenibase
```

### Kubernetes Deployment

**Example Kubernetes manifests:**

**Deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: serenibase
spec:
  replicas: 3
  selector:
    matchLabels:
      app: serenibase
  template:
    metadata:
      labels:
        app: serenibase
    spec:
      containers:
      - name: serenibase
        image: serenibase/serenibase:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_HOST
          value: postgres-service
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

**Service:**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: serenibase-service
spec:
  selector:
    app: serenibase
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

**ConfigMap:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: serenibase-config
data:
  SERVER_ENV: "production"
  DATABASE_HOST: "postgres-service"
  DATABASE_PORT: "5432"
```

**Secret:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: serenibase-secret
type: Opaque
stringData:
  database-password: "your-secure-password"
  jwt-secret: "your-jwt-secret"
```

### Cloud Platform Deployment

**AWS:**
- **ECS/Fargate**: Container orchestration
- **RDS PostgreSQL**: Managed database
- **S3**: File storage
- **S3**: File storage
- **ALB**: Load balancer
- **Route53**: DNS

**Google Cloud Platform:**
- **Cloud Run**: Serverless containers
- **Cloud SQL**: Managed PostgreSQL
- **Cloud Storage**: File storage
- **Cloud Load Balancing**: Load balancer
- **Cloud DNS**: DNS

**Azure:**
- **Container Instances**: Container hosting
- **Azure Database for PostgreSQL**: Managed database
- **Blob Storage**: File storage
- **Application Gateway**: Load balancer
- **Azure DNS**: DNS

### Reverse Proxy Configuration

**Nginx configuration** (`nginx.conf`):

```nginx
upstream serenibase_backend {
    least_conn;
    server serenibase:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name serenibase.example.com;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name serenibase.example.com;

    ssl_certificate /etc/nginx/ssl/cert.pem;
    ssl_certificate_key /etc/nginx/ssl/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req zone=api_limit burst=20 nodelay;

    # API routes
    location /api/ {
        proxy_pass http://serenibase_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        
        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Frontend
    location / {
        proxy_pass http://base-ui:5050;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }

    # File uploads - larger body size
    location /api/v1/asset/upload {
        client_max_body_size 100M;
        proxy_pass http://serenibase_backend;
        proxy_request_buffering off;
    }
}
```

## Development

### Local Development Setup

```bash
# Clone repository
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base

# Install dependencies
go mod download

# Setup PostgreSQL
createdb serenibase_dev
psql serenibase_dev < init.sql

# Create .env for development
cp .env.example .env.development
# Edit with development settings

# Run in development mode
go run cmd/server/main.go

# Or use air for hot reload
go install github.com/cosmtrek/air@latest
air
```

### Project Structure

```
sereni-base/
├── cmd/
│   └── server/
│       └── main.go                 # Application entry point
├── internal/
│   ├── app/                        # Application initialization
│   ├── config/                     # Configuration management
│   ├── handlers/                   # HTTP handlers (controllers)
│   │   ├── auth.go
│   │   ├── base.go
│   │   ├── table.go
│   │   ├── user.go
│   │   └── workspace.go
│   ├── middleware/                 # HTTP middleware
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── logger.go
│   │   └── rate_limiter.go
│   ├── models/                     # Data models
│   ├── dto/                        # Data transfer objects
│   ├── services/                   # Business logic
│   │   ├── auth/
│   │   ├── base/
│   │   ├── table/
│   │   └── workspace/
│   ├── providers/                  # External service providers
│   ├── router/                     # Route definitions
│   └── utils/                      # Utility functions
├── tests/                          # Integration tests
├── docs/                           # Additional documentation
├── build/
│   ├── config/                     # Config templates
│   └── scripts/                    # Build and setup scripts
├── go-postgres-rest/               # Database abstraction library
├── docker-compose.yaml             # Docker Compose configuration
├── Dockerfile                      # Docker image definition
├── Makefile                        # Build tasks
├── .env.example                    # Example environment file
└── README.md                       # This file
```

### Adding a New Feature

**Example: Adding a "Comments" feature**

**1. Create model** (`internal/models/comment.go`):
```go
package models

import "time"

type Comment struct {
    ID        string    `json:"id"`
    TableID   string    `json:"table_id"`
    RecordID  string    `json:"record_id"`
    UserID    string    `json:"user_id"`
    Content   string    `json:"content"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

**2. Create DTO** (`internal/dto/comment.go`):
```go
package dto

type CreateCommentRequest struct {
    TableID  string `json:"table_id" validate:"required,uuid4"`
    RecordID string `json:"record_id" validate:"required,uuid4"`
    Content  string `json:"content" validate:"required,min=1,max=1000"`
}

type CommentResponse struct {
    ID        string `json:"id"`
    Content   string `json:"content"`
    Author    string `json:"author"`
    CreatedAt string `json:"created_at"`
}
```

**3. Create service interface** (`internal/services/interfaces/comment.go`):
```go
package interfaces

type CommentService interface {
    CreateComment(req dto.CreateCommentRequest, userID string) (*models.Comment, error)
    GetComments(tableID, recordID string) ([]models.Comment, error)
    DeleteComment(commentID, userID string) error
}
```

**4. Implement service** (`internal/services/comment/comment.go`):
```go
package comment

import "serenibase/internal/services/interfaces"

type commentService struct {
    repo interfaces.CommentRepository
}

func NewCommentService(repo interfaces.CommentRepository) interfaces.CommentService {
    return &commentService{repo: repo}
}

func (s *commentService) CreateComment(req dto.CreateCommentRequest, userID string) (*models.Comment, error) {
    // Implementation
}
```

**5. Create handler** (`internal/handlers/comment.go`):
```go
package handlers

type CommentHandler struct {
    commentService interfaces.CommentService
}

// @Summary Create comment
// @Tags Comments
// @Accept json
// @Produce json
// @Param request body dto.CreateCommentRequest true "Comment data"
// @Success 201 {object} dto.CommentResponse
// @Router /comment/create [post]
func (h *CommentHandler) CreateComment(c *gin.Context) {
    // Implementation
}
```

**6. Add routes** (`internal/router/router.go`):
```go
comment := r.Group("/comment")
{
    comment.POST("/create", handlers.Comment.CreateComment)
    comment.GET("/:table_id/:record_id", handlers.Comment.GetComments)
    comment.DELETE("/:id", handlers.Comment.DeleteComment)
}
```

**7. Update Swagger docs:**
```bash
swag init -g cmd/server/main.go -o ./docs
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/services/table/...

# Run tests with race detector
go test -race ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Code Quality

**Run linters:**
```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run

# Auto-fix issues
golangci-lint run --fix
```

**Format code:**
```bash
# Format all Go files
go fmt ./...

# Or use goimports (adds missing imports)
goimports -w .
```

### Debugging

**VS Code launch.json:**
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch SereniBase",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/server",
            "env": {
                "ENV_FILE": ".env.development"
            },
            "preLaunchTask": "go: build"
        }
    ]
}
```

## Testing

### Unit Tests

**Example unit test** (`internal/services/table/table_test.go`):

```go
package table_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
    "serenibase/internal/services/table"
)

func TestCreateTable(t *testing.T) {
    // Setup mock repository
    mockRepo := new(MockTableRepository)
    mockRepo.On("CreateTable", mock.Anything).Return(&models.Table{
        ID:   "tbl_123",
        Name: "Test Table",
    }, nil)

    // Create service with mock
    service := table.NewTableService(mockRepo)

    // Execute test
    result, err := service.CreateTable(dto.CreateTableRequest{
        Name:   "Test Table",
        BaseID: "base_123",
    })

    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, result)
    assert.Equal(t, "Test Table", result.Name)
    mockRepo.AssertExpectations(t)
}
```

### Integration Tests

**Example integration test** (`tests/integration/workspace_test.go`):

```go
package integration_test

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "github.com/stretchr/testify/assert"
)

func TestCreateWorkspaceFlow(t *testing.T) {
    // Setup test server
    router := setupTestRouter()
    
    // Login to get token
    token := login(t, router, "admin@example.com", "Admin@123")
    
    // Create workspace
    reqBody := map[string]interface{}{
        "name": "Test Workspace",
        "description": "Integration test workspace",
    }
    body, _ := json.Marshal(reqBody)
    
    req, _ := http.NewRequest("POST", "/api/v1/workspace/create", bytes.NewBuffer(body))
    req.Header.Set("Authorization", "Bearer "+token)
    req.Header.Set("Content-Type", "application/json")
    
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    
    // Assertions
    assert.Equal(t, http.StatusCreated, w.Code)
    
    var response map[string]interface{}
    json.Unmarshal(w.Body.Bytes(), &response)
    assert.Equal(t, "Test Workspace", response["name"])
}
```

### API Testing with Postman

**Import Postman collection:**
```bash
# Collection is included in repository
open serenibase.postman_collection.json
```

**Collection includes:**
- All API endpoints
- Pre-configured environments (development, staging, production)
- Automated tests for each endpoint
- Token management

**Run collection:**
```bash
# Install newman (Postman CLI)
npm install -g newman

# Run collection
newman run serenibase.postman_collection.json \
  --environment development.postman_environment.json
```

### Load Testing

**Example with k6:**

```javascript
// load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  stages: [
    { duration: '30s', target: 50 },  // Ramp-up to 50 users
    { duration: '1m', target: 50 },   // Stay at 50 users
    { duration: '30s', target: 0 },   // Ramp-down to 0 users
  ],
};

export default function () {
  // Login
  let loginRes = http.post('http://localhost:8080/api/v1/auth/login', JSON.stringify({
    email: 'admin@example.com',
    password: 'Admin@123'
  }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(loginRes, {
    'login successful': (r) => r.status === 200,
  });

  let token = loginRes.json('access_token');

  // List workspaces
  let wsRes = http.get('http://localhost:8080/api/v1/workspace/list', {
    headers: { 'Authorization': `Bearer ${token}` },
  });

  check(wsRes, {
    'workspaces retrieved': (r) => r.status === 200,
  });

  sleep(1);
}
```

**Run load test:**
```bash
k6 run load_test.js
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Refused

**Error:**
```
failed to connect to database: dial tcp 127.0.0.1:5432: connect: connection refused
```

**Solution:**
```bash
# Check if PostgreSQL is running
docker compose ps postgres

# If not running, start it
docker compose up -d postgres

# Check PostgreSQL logs
docker compose logs postgres

# Verify connection settings in .env
DATABASE_HOST=postgres  # Use "postgres" for Docker, "localhost" for local
DATABASE_PORT=5432
```

#### 2. JWT Token Validation Failed

**Error:**
```json
{
  "error": "Unauthorized",
  "message": "Invalid token"
}
```

**Solution:**
```bash
# Ensure JWT secret matches between services
# Check .env:
AUTH_JWT_SECRET=same_secret_everywhere

# Restart services after changing
docker compose restart serenibase jwt-provider
```

#### 3. File Upload Failed

**Error:**
```json
{
  "error": "File upload failed",
  "message": "Connection refused to storage service"
}
```

**Solution:**
```bash
# Check storage service is running
docker compose ps sereni-storage-provider

# Verify storage configuration
STORAGE_URL=http://sereni-storage-provider:8083/api/v1
STORAGE_DRIVER=minio  # or local, s3

# Check storage service logs
docker compose logs sereni-storage-provider
```

#### 4. CORS Errors in Browser

**Error in browser console:**
```
Access to fetch at 'http://localhost:8080' from origin 'http://localhost:5050'
has been blocked by CORS policy
```

**Solution:**
```dotenv
# Update .env with correct frontend URL
CORS_ALLOWED_ORIGINS=http://localhost:5050,http://localhost:8080
AUTH_ALLOWED_ORIGINS=http://localhost:5050,http://localhost:8080
EMAIL_ALLOWED_ORIGIN=http://localhost:5050,http://localhost:8080

# Restart services
docker compose restart
```

#### 5. Migration Table Already Exists

**Error:**
```
relation "migrations" already exists
```

**Solution:**
```bash
# This is usually safe to ignore - table already created

# If you need to reset migrations:
psql -h localhost -U postgres -d serenibase
DROP TABLE IF EXISTS migrations;
\q

# Restart application
docker compose restart serenibase
```

#### 6. Port Already in Use

**Error:**
```
Bind for 0.0.0.0:8080 failed: port is already allocated
```

**Solution:**
```bash
# Find process using port
lsof -i :8080  # Linux/macOS
netstat -ano | findstr :8080  # Windows

# Kill process or change port in .env
SERVER_PORT=8081

# Restart services
docker compose up -d
```

### Debugging Tips

**1. Enable debug logging:**
```dotenv
LOG_LEVEL=debug
SERVER_ENV=dev
```

**2. View real-time logs:**
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f serenibase

# With timestamps
docker compose logs -f --timestamps
```

**3. Access container shell:**
```bash
# Access SereniBase container
docker compose exec serenibase sh

# Check environment variables
env | grep DATABASE

# Test database connection
nc -zv postgres 5432
```

**4. Database debugging:**
```bash
# Connect to PostgreSQL
docker compose exec postgres psql -U postgres -d serenibase

# Common queries
\dt  -- List tables
\d users  -- Describe users table
SELECT * FROM workspaces;
SELECT * FROM bases;
```

**5. Network debugging:**
```bash
# Check Docker network
docker network ls
docker network inspect sereni-base_serenibase-network

# Test service connectivity
docker compose exec serenibase ping postgres
docker compose exec serenibase curl http://jwt-provider:8081/health
```

### Health Checks

**Check service health:**
```bash
# SereniBase API
curl http://localhost:8080/health

# JWT Provider
curl http://localhost:8081/health

# Email Service
curl http://localhost:8082/health

# Storage Provider
curl http://localhost:8083/health

# Antivirus Service
curl http://localhost:8084/health
```

## Performance

### Optimization Tips

**1. Database Connection Pooling:**
```dotenv
# Adjust based on expected concurrent users
DATABASE_MAX_OPEN_CONNS=100      # Max concurrent connections
DATABASE_MAX_IDLE_CONNS=25       # Idle connections to maintain
DATABASE_CONN_MAX_LIFETIME=1h    # Recycle connections
```

**2. Database Connection Pooling:**
```sql
-- Analyze query performance
EXPLAIN ANALYZE SELECT * FROM records WHERE table_id = 'tbl_123';

-- Create indexes on frequently queried columns
CREATE INDEX idx_records_table_id ON records(table_id);
CREATE INDEX idx_records_created_at ON records(created_at);

-- Update statistics
ANALYZE records;
```

**4. Nginx Caching:**
```nginx
# Add to nginx.conf
proxy_cache_path /var/cache/nginx levels=1:2 keys_zone=api_cache:10m max_size=1g inactive=60m;

location /api/v1/table/ {
    proxy_cache api_cache;
    proxy_cache_valid 200 5m;
    proxy_cache_key "$request_uri";
    proxy_pass http://serenibase_backend;
}
```

**5. Scale Horizontally:**
```yaml
# docker-compose.prod.yaml
services:
  serenibase:
    deploy:
      replicas: 5  # Run 5 instances
```

### Performance Monitoring

**Metrics to monitor:**
- API response times
- Database connection pool usage
- Memory usage
- CPU usage
- Request rate
- Error rate
- Cache hit ratio

**Monitoring tools:**
- **Prometheus + Grafana**: Metrics collection and visualization
- **ELK Stack**: Log aggregation and analysis
- **New Relic / DataDog**: APM (Application Performance Monitoring)
- **PostgreSQL pg_stat_statements**: Query performance

## Best Practices

### 1. Security

- ✅ Always use HTTPS in production
- ✅ Change default passwords immediately
- ✅ Use strong JWT secrets (256-bit random)
- ✅ Enable database SSL in production
- ✅ Regularly update dependencies
- ✅ Enable antivirus scanning for uploads
- ✅ Implement rate limiting
- ✅ Use environment variables for secrets (never commit .env)

### 2. Database

- ✅ Regular backups (automated)
- ✅ Use transactions for multi-step operations
- ✅ Create indexes on frequently queried columns
- ✅ Monitor connection pool utilization
- ✅ Use database migrations for schema changes

### 3. API Design

- ✅ Use consistent error responses
- ✅ Implement pagination for list endpoints
- ✅ Use appropriate HTTP status codes
- ✅ Version your API (/api/v1, /api/v2)
- ✅ Provide request tracing (X-Request-ID header)

### 4. Deployment

- ✅ Use Docker for consistent environments
- ✅ Separate configuration per environment (dev/staging/prod)
- ✅ Implement health checks
- ✅ Use blue-green deployments for zero downtime
- ✅ Monitor logs and metrics

### 5. Development

- ✅ Write unit tests for business logic
- ✅ Write integration tests for API endpoints
- ✅ Use linters and formatters
- ✅ Document API with Swagger
- ✅ Follow Go best practices and idioms

## Migration Guide

### Migrating from Airtable

**1. Export data from Airtable:**
- Export each table as CSV
- Download attachments

**2. Create workspace and base in SereniBase:**
```bash
# Create workspace
curl -X POST http://localhost:8080/api/v1/workspace/create \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Migrated from Airtable"}'

# Create base
curl -X POST http://localhost:8080/api/v1/base/create \
  -d '{"workspace_id": "ws_123", "name": "My Base"}'
```

**3. Create tables with matching schema:**
```bash
# For each Airtable table, create equivalent SereniBase table
curl -X POST http://localhost:8080/api/v1/table/create \
  -d '{
    "base_id": "base_123",
    "name": "Contacts",
    "columns": [...]
  }'
```

**4. Import data:**
```bash
# Use bulk insert API
curl -X POST http://localhost:8080/api/v1/table/$TABLE_ID/records/bulk \
  -d '{"records": [...]}'
```

**5. Migrate attachments:**
```bash
# Upload files to SereniBase storage
# Update records with file references
```

### Migrating from NocoDB

**1. Export NocoDB database:**
```bash
# NocoDB uses PostgreSQL - export with pg_dump
pg_dump nocodb_db > nocodb_export.sql
```

**2. Transform schema:**
- Map NocoDB meta tables to SereniBase structure
- Convert custom field types

**3. Migrate using SereniBase API:**
Similar to Airtable migration steps above.

## FAQ

**Q: Can I self-host SereniBase?**
A: Yes! SereniBase is 100% self-hosted and open-source. Deploy on your own infrastructure with Docker.

**Q: What's the difference between SereniBase and Airtable?**
A: SereniBase is self-hosted and open-source, giving you complete control over your data and infrastructure. No per-user costs, unlimited customization, and full data ownership.

**Q: Can I use SereniBase for production applications?**
A: Yes! SereniBase is production-ready with RBAC, audit logging, comprehensive testing, and enterprise features.

**Q: How do I backup my data?**
A: Use PostgreSQL backup tools (pg_dump) for database backups. For files, backup the storage volume (MinIO/S3).

```bash
# Database backup
docker compose exec postgres pg_dump -U postgres serenibase > backup.sql

# Restore
docker compose exec -T postgres psql -U postgres serenibase < backup.sql
```

**Q: How many users can SereniBase handle?**
A: With proper configuration and horizontal scaling, SereniBase can handle thousands of concurrent users. Performance depends on your infrastructure.

**Q: Can I customize the frontend UI?**
A: Yes! The base-ui frontend is open-source React. Fork and customize as needed.

**Q: Does SereniBase support real-time collaboration?**
A: Real-time collaboration via WebSockets is in development. Current version supports multi-user access with optimistic locking.

**Q: Can I integrate SereniBase with external services?**
A: Yes! Use the REST API and webhooks (coming soon) to integrate with any external service.

**Q: Is there a managed/cloud version?**
A: Currently, SereniBase is self-hosted only. A managed cloud version may be offered in the future.

**Q: What's the pricing?**
A: SereniBase is MIT licensed and completely free. You only pay for your infrastructure costs.

**Q: How do I get support?**
A: Check documentation, open GitHub issues, or join our community Discord.

**Q: Can I contribute to SereniBase?**
A: Absolutely! We welcome contributions. See [Contributing](#contributing) section.

## Contributing

We welcome contributions from the community! Here's how you can help:

### Ways to Contribute

- 🐛 **Report bugs**: Open an issue with reproduction steps
- 💡 **Suggest features**: Share your ideas for improvements
- 📖 **Improve documentation**: Fix typos, add examples, clarify instructions
- 🔧 **Submit code**: Fix bugs or implement new features
- ✅ **Write tests**: Increase test coverage
- 🌍 **Translate**: Help localize SereniBase

### Contribution Workflow

**1. Fork and clone:**
```bash
git clone https://github.com/YOUR_USERNAME/sereni-base.git
cd sereni-base
```

**2. Create feature branch:**
```bash
git checkout -b feature/my-new-feature
```

**3. Make changes and test:**
```bash
# Make your changes
# Add tests
go test ./...
```

**4. Commit with clear message:**
```bash
git add .
git commit -m "Add feature: brief description"
```

**5. Push and create pull request:**
```bash
git push origin feature/my-new-feature
# Open pull request on GitHub
```

### Code Guidelines

- Follow Go best practices and idioms
- Write tests for new features
- Update documentation
- Use meaningful variable and function names
- Add Swagger comments for API endpoints
- Format code with `go fmt` and `goimports`

### Pull Request Checklist

- [ ] Code follows project conventions
- [ ] Tests pass (`go test ./...`)
- [ ] New tests added for new features
- [ ] Documentation updated
- [ ] Swagger docs regenerated if API changed
- [ ] No merge conflicts
- [ ] PR description explains changes clearly

## License

SereniBase is licensed under the **MIT License**.

```
MIT License

Copyright (c) 2026 SereniBase Contributors

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

See [LICENSE](LICENSE) file for full license text.

---

