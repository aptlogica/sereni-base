# 📚 Documentation Index

## Welcome to SereniBase Documentation!

This index helps you find the right documentation for your needs.

---

## 🎯 Quick Navigation

### I want to...

| Goal | Document | Time |
|------|----------|------|
| **Get started quickly** | [Interactive Setup README](../INTERACTIVE_SETUP_README.md) | 5 min |
| **See all required variables** | [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md) | 2 min |
| **Understand defaults vs overrides** | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md) | 15 min |
| **Look up specific variable** | [Environment Variables](./ENVIRONMENT_VARIABLES.md) | Reference |
| **Check API error codes** | [API Response Codes](./API_RESPONSE_CODES.md) | Reference |
| **Production deployment** | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment) | 30 min |

---

## 📖 Documentation Structure

```
docs/
│
├── 🚀 INTERACTIVE_SETUP_README.md
│   └─ Start here! Interactive setup script guide
│      • How to run setup script (Windows/Linux/macOS)
│      • What gets configured
│      • Default values explained
│      • Troubleshooting
│
├── 📊 ENV_QUICK_REFERENCE_CARD.md
│   └─ Print and keep handy!
│      • Required vs optional variables (color-coded)
│      • Quick decision tree
│      • Security checklist
│      • Configuration scenarios
│
├── 📘 ENVIRONMENT_SETUP_GUIDE.md
│   └─ Complete configuration guide
│      • Interactive setup instructions
│      • All variables explained
│      • Required vs optional details
│      • Application defaults reference
│      • Configuration by scenario
│      • Production checklist
│
├── 📙 ENVIRONMENT_VARIABLES.md
│   └─ Detailed variable reference
│      • Every variable documented
│      • Default values
│      • Usage examples
│      • When to change
│
└── 📕 API_RESPONSE_CODES.md
    └─ Complete API reference
       • All HTTP status codes
       • Error codes
       • UI messages
       • Developer messages
```

---

## 🎓 Learning Paths

### Path 1: Beginner (First Time Setup)

1. **Start:** [Interactive Setup README](../INTERACTIVE_SETUP_README.md)
   - Run the setup script
   - Follow prompts
   - Get running quickly

2. **Learn:** [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md)
   - Understand what was configured
   - See what can be changed
   - Security checklist

3. **Explore:** [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md)
   - Learn about scenarios
   - Understand defaults
   - Plan for production

### Path 2: Developer (Deep Understanding)

1. **Reference:** [Environment Variables](./ENVIRONMENT_VARIABLES.md)
   - Detailed documentation
   - All options explained
   - Examples for each variable

2. **Configure:** [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md)
   - Manual configuration
   - Advanced scenarios
   - Production optimization

3. **API:** [API Response Codes](./API_RESPONSE_CODES.md)
   - Error handling
   - Status codes
   - User messages

### Path 3: Operations (Production Deployment)

1. **Plan:** [Environment Setup Guide - Production](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment)
   - Production requirements
   - Security checklist
   - Infrastructure setup

2. **Configure:** [ENV Quick Reference Card - Security](./ENV_QUICK_REFERENCE_CARD.md#-security-checklist)
   - Required changes
   - Security variables
   - Validation

3. **Monitor:** [API Response Codes](./API_RESPONSE_CODES.md)
   - Error monitoring
   - Alert configuration
   - Troubleshooting

---

## 🔍 Find Information By...

### By Topic

| Topic | Primary Document | Secondary Document |
|-------|------------------|-------------------|
| **Setup & Installation** | [Interactive Setup README](../INTERACTIVE_SETUP_README.md) | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md) |
| **Environment Variables** | [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md) | [Environment Variables](./ENVIRONMENT_VARIABLES.md) |
| **Configuration Defaults** | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md#optional-variables-with-defaults) | [Environment Variables](./ENVIRONMENT_VARIABLES.md) |
| **Security** | [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md#-security-checklist) | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment) |
| **Production** | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment) | [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md) |
| **API Errors** | [API Response Codes](./API_RESPONSE_CODES.md) | - |
| **Docker** | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md#scenario-4-docker-compose-all-services) | [Interactive Setup README](../INTERACTIVE_SETUP_README.md) |
| **Troubleshooting** | [Interactive Setup README](../INTERACTIVE_SETUP_README.md#-troubleshooting) | [Environment Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md#common-questions) |

### By Question

<details>
<summary><b>How do I get started?</b></summary>

1. Run the interactive setup script:
   ```bash
   # Windows
   .\setup-interactive.ps1
   
   # Linux/macOS
   ./setup-interactive.sh
   ```

2. See [Interactive Setup README](../INTERACTIVE_SETUP_README.md) for details

</details>

<details>
<summary><b>What environment variables are required?</b></summary>

**Minimum required:**
- `PUBLIC_HOST`
- `OWNER_EMAIL`
- `OWNER_PASSWORD`
- `AUTH_JWT_SECRET`

See [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md#-tldr---minimum-required-for-production) for complete list.

</details>

<details>
<summary><b>What are the default values?</b></summary>

All defaults are listed in:
- [Environment Setup Guide - Defaults Reference](./ENVIRONMENT_SETUP_GUIDE.md#complete-defaults-reference)
- [ENV Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md)

Application code: `internal/config/config.go`

</details>

<details>
<summary><b>How do I configure for production?</b></summary>

Follow:
1. [Environment Setup Guide - Production Scenario](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment)
2. [ENV Quick Reference Card - Security Checklist](./ENV_QUICK_REFERENCE_CARD.md#-security-checklist)

</details>

<details>
<summary><b>What do API error codes mean?</b></summary>

See [API Response Codes](./API_RESPONSE_CODES.md) for complete reference with:
- HTTP status codes
- Application error codes
- UI messages
- Developer messages

</details>

<details>
<summary><b>How do I configure email?</b></summary>

Email configuration is optional but needed for:
- Password reset
- User notifications

See:
- [Environment Variables - Email](./ENVIRONMENT_VARIABLES.md#-email-service-configuration)
- [Environment Setup Guide - Email](./ENVIRONMENT_SETUP_GUIDE.md#conditionally-required)

</details>

<details>
<summary><b>How do I use Docker?</b></summary>

For Docker deployment:
1. Use `DATABASE_HOST=postgres` (service name)
2. Use service names for all internal URLs
3. See [Environment Setup Guide - Docker Scenario](./ENVIRONMENT_SETUP_GUIDE.md#scenario-4-docker-compose-all-services)

</details>

<details>
<summary><b>Can I change configuration after deployment?</b></summary>

Yes! Edit `.env` and restart:
```bash
docker-compose restart
```

See [Environment Setup Guide - Common Questions](./ENVIRONMENT_SETUP_GUIDE.md#common-questions)

</details>

---

## 📋 Cheat Sheets

### Interactive Setup

```bash
# Windows
.\setup-interactive.ps1

# Linux/macOS
chmod +x setup-interactive.sh
./setup-interactive.sh

# Skip Docker check
.\setup-interactive.ps1 -SkipDocker  # Windows
./setup-interactive.sh --skip-docker # Linux/macOS
```

### Manual Setup

```bash
# Copy template
cp build/config/.env.example .env

# Edit required variables
# - PUBLIC_HOST
# - OWNER_EMAIL
# - OWNER_PASSWORD
# - AUTH_JWT_SECRET

# Start services
docker-compose up -d
```

### Check Configuration

```bash
# View .env
cat .env

# Find specific variable
grep "PUBLIC_HOST" .env

# Check logs
docker-compose logs serenibase
```

### Generate Secure Secrets

```bash
# JWT Secret (Linux/macOS)
openssl rand -base64 64

# JWT Secret (PowerShell)
[Convert]::ToBase64String((1..64 | ForEach-Object { Get-Random -Maximum 256 }))
```

---

## 🎯 Quick Reference Matrix

| Task | Beginner | Developer | DevOps |
|------|----------|-----------|--------|
| **Getting Started** | ⭐ [Interactive Setup](../INTERACTIVE_SETUP_README.md) | [Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md) | [Setup Guide](./ENVIRONMENT_SETUP_GUIDE.md) |
| **Configuration** | ⭐ [Quick Reference Card](./ENV_QUICK_REFERENCE_CARD.md) | [Environment Variables](./ENVIRONMENT_VARIABLES.md) | [Environment Variables](./ENVIRONMENT_VARIABLES.md) |
| **Troubleshooting** | ⭐ [Interactive Setup](../INTERACTIVE_SETUP_README.md#-troubleshooting) | [Setup Guide - FAQ](./ENVIRONMENT_SETUP_GUIDE.md#common-questions) | [Setup Guide - FAQ](./ENVIRONMENT_SETUP_GUIDE.md#common-questions) |
| **API Reference** | [Response Codes](./API_RESPONSE_CODES.md) | ⭐ [Response Codes](./API_RESPONSE_CODES.md) | [Response Codes](./API_RESPONSE_CODES.md) |
| **Production** | [Security Checklist](./ENV_QUICK_REFERENCE_CARD.md#-security-checklist) | [Production Scenario](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment) | ⭐ [Production Scenario](./ENVIRONMENT_SETUP_GUIDE.md#scenario-3-production-deployment) |

⭐ = Recommended starting point for this role

---

## 🔗 Related Resources

### Internal

- [Main README](../README.md) - Project overview
- [Build Scripts](../build/scripts/) - Setup automation
- [Docker Compose](../docker-compose.yaml) - Container orchestration
- [Source Code](../internal/config/config.go) - Configuration implementation

### External

- [Docker Documentation](https://docs.docker.com/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Go Environment Variables](https://golang.org/doc/install/source#environment)

---

## 📝 Documentation Versions

| Document | Last Updated | Version |
|----------|-------------|---------|
| Interactive Setup README | 2026-02-04 | 1.0 |
| ENV Quick Reference Card | 2026-02-04 | 1.0 |
| Environment Setup Guide | 2026-02-04 | 1.0 |
| Environment Variables | 2024-XX-XX | 1.0 |
| API Response Codes | 2024-XX-XX | 1.0 |

---

## 🤝 Contributing to Documentation

Found an issue or want to improve the docs?

1. Check which document needs updating (see structure above)
2. Make changes maintaining the current format
3. Update "Last Updated" date
4. Submit a pull request

---

## 💡 Documentation Tips

### For New Users

1. **Don't read everything!** Start with [Interactive Setup](../INTERACTIVE_SETUP_README.md)
2. **Use the search** - Ctrl+F is your friend
3. **Follow the learning paths** - They're designed for progressive learning
4. **Bookmark this index** - Come back when you need something specific

### For Developers

1. **Keep Quick Reference Card handy** - It has the decision tree
2. **Use Environment Variables doc** - Complete reference for all options
3. **Check API Response Codes** - When implementing error handling
4. **Read source code** - `internal/config/config.go` has all defaults

### For Operations

1. **Production checklist first** - Don't miss security items
2. **Understand all required variables** - Use Quick Reference Card
3. **Plan monitoring** - API Response Codes help with alerts
4. **Document your overrides** - Keep track of what you changed and why

---

## 🎯 Success Criteria

You've successfully learned SereniBase configuration when you can:

- [ ] Run the interactive setup script
- [ ] Identify required vs optional variables
- [ ] Explain what `PUBLIC_HOST` does
- [ ] Generate a secure JWT secret
- [ ] Configure email for password reset
- [ ] Deploy to production securely
- [ ] Troubleshoot common issues
- [ ] Read API error codes

---

**Welcome to SereniBase! 🚀**

*Start with [Interactive Setup](../INTERACTIVE_SETUP_README.md) and you'll be running in minutes!*

---

**Last Updated:** February 4, 2026
