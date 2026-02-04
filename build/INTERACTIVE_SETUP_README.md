# 🚀 Interactive Setup Scripts

Welcome to SereniBase! This directory contains interactive setup scripts that guide you through configuring your application with a user-friendly interface.

## 📋 Available Scripts

### Windows (PowerShell)
```powershell
.\build\scripts\setup-interactive.ps1
```

### Linux/macOS (Bash)
```bash
chmod +x build/scripts/setup-interactive.sh
./build/scripts/setup-interactive.sh
```

## ✨ Features

- **🎯 Interactive Prompts** - Shows default values in brackets like `[localhost]`
- **✅ Input Validation** - Email format, password confirmation
- **🌐 Auto-Detection** - Finds your local IP address
- **🐳 Docker Check** - Verifies Docker installation
- **💾 Auto-Generate .env** - Creates complete configuration file
- **🎨 Color Output** - Beautiful, easy-to-read interface
- **📝 Summary Report** - Shows what was configured and next steps

## 🎮 Usage

### Basic Usage

Just run the script and follow the prompts:

**Windows:**
```powershell
.\build\scripts\setup-interactive.ps1
```

**Linux/macOS:**
```bash
./build/scripts/setup-interactive.sh
```

### Advanced Options

**Skip Docker Check:**
```powershell
# Windows
.\build\scripts\setup-interactive.ps1 -SkipDocker

# Linux/macOS
./build/scripts/setup-interactive.sh --skip-docker
```

**Show Help:**
```powershell
# Windows
.\build\scripts\setup-interactive.ps1 -Help

# Linux/macOS
./build/scripts/setup-interactive.sh --help
```

## 📸 Example Session

```
╔══════════════════════════════════════════════════════════════════════════════╗
║                         🚀 SERENIBASE SETUP                                   ║
║                     Interactive Configuration Wizard                          ║
╚══════════════════════════════════════════════════════════════════════════════╝

📡 Detected System Information:
   OS: Windows 10
   Local IP: 192.168.1.100

🐳 Checking Docker...
   ✓ Docker is available: Docker version 24.0.0

═══════════════════════════════════════════════════════════════
                    📋 CONFIGURATION                            
═══════════════════════════════════════════════════════════════

Press Enter to accept default values shown in [brackets]

┌─────────────────────────────────────────────────────────┐
│           🌐 NETWORK CONFIGURATION                      │
└─────────────────────────────────────────────────────────┘

This is how users will access your application.
Examples:
  - localhost (for testing on this machine)
  - 192.168.1.100 (for LAN access)
  - yourdomain.com (for production)

IP Address or Domain [localhost]: ▌
```

## 🎯 What Gets Configured

### Required Settings (You'll Be Prompted)

1. **Network Configuration**
   - `PUBLIC_HOST` - Your domain or IP address

2. **Admin Account**
   - `OWNER_FIRST_NAME` - First name
   - `OWNER_LAST_NAME` - Last name
   - `OWNER_EMAIL` - Login email
   - `OWNER_PASSWORD` - Password (with confirmation)

3. **Security**
   - `AUTH_JWT_SECRET` - JWT signing secret

4. **Database**
   - Docker PostgreSQL (recommended) or external database

5. **Optional Features**
   - Email configuration (for password reset)
   - Storage driver (local/minio/aws)

### Auto-Configured Settings (No Input Needed)

All these are automatically set based on your `PUBLIC_HOST`:

- Server configuration (port, host, timeouts)
- Internal service URLs (auth, email, storage, antivirus)
- CORS origins
- Password reset URLs
- Frontend API URLs
- All service-to-service communication URLs

See [`docs/ENVIRONMENT_SETUP_GUIDE.md`](./docs/ENVIRONMENT_SETUP_GUIDE.md) for complete details.

## 📖 Default Values

When you see `[value]` in a prompt, that's the default. Press **Enter** to use it:

```
IP Address or Domain [localhost]:     ← Press Enter to use "localhost"
IP Address or Domain [localhost]: 192.168.1.100  ← Or type a new value
```

### Common Defaults

| Prompt | Default | When to Change |
|--------|---------|----------------|
| IP Address or Domain | `localhost` | LAN/production access |
| Admin First Name | `Admin` | Personalization |
| Admin Last Name | `User` | Personalization |
| Admin Email | `admin@example.com` | Always (use real email) |
| Admin Password | `Admin@123` | Always (use strong password) |
| JWT Secret | Long random string | Production (use stronger) |
| Database Password | `postgres` | Production (use strong password) |
| Storage Driver | `local` | Cloud storage needed |

## ✅ After Setup

Once the script completes, you'll see:

1. **Configuration Summary** - What was set
2. **Next Steps** - Commands to run
3. **Access Information** - URLs and credentials
4. **Documentation Links** - Where to learn more

### Start Your Application

```bash
# Start all services with Docker
docker-compose up -d

# Access the application
# URL will be shown in the summary
```

## 🔧 Manual Configuration Alternative

If you prefer manual configuration:

1. **Copy the template:**
   ```bash
   cp build/config/.env.example .env
   ```

2. **Edit the file:**
   ```bash
   # Windows
   notepad .env
   
   # Linux/macOS
   nano .env
   # or
   vim .env
   ```

3. **Update required values:**
   - `PUBLIC_HOST`
   - `OWNER_EMAIL`
   - `OWNER_PASSWORD`
   - `AUTH_JWT_SECRET`

## 📚 Complete Documentation

For detailed information about all environment variables:

- **[Environment Setup Guide](./docs/ENVIRONMENT_SETUP_GUIDE.md)** - Complete variable reference
- **[Environment Variables](./docs/ENVIRONMENT_VARIABLES.md)** - Detailed documentation
- **[API Response Codes](./docs/API_RESPONSE_CODES.md)** - API error codes

## 🐛 Troubleshooting

### Script Won't Run (Windows)

**Error:** "Running scripts is disabled on this system"

**Solution:**
```powershell
# Allow script execution (run PowerShell as Administrator)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser

# Then run the script
.\build\scripts\setup-interactive.ps1
```

### Script Won't Run (Linux/macOS)

**Error:** "Permission denied"

**Solution:**
```bash
# Make script executable
chmod +x build/scripts/setup-interactive.sh

# Then run the script
./build/scripts/setup-interactive.sh
```

### Docker Not Found

If Docker check fails:

1. Install Docker:
   - Windows: [Docker Desktop](https://www.docker.com/products/docker-desktop)
   - Linux: `curl -fsSL https://get.docker.com -o get-docker.sh && sh get-docker.sh`
   - macOS: [Docker Desktop](https://www.docker.com/products/docker-desktop)

2. Or skip the check:
   ```bash
   # Windows
   .\build\scripts\setup-interactive.ps1 -SkipDocker
   
   # Linux/macOS
   ./build/scripts/setup-interactive.sh --skip-docker
   ```

### Invalid Email Format

Make sure your email follows the format: `user@domain.com`

Examples:
- ✅ `admin@company.com`
- ✅ `john.doe@example.org`
- ❌ `admin@` (missing domain)
- ❌ `@example.com` (missing user)
- ❌ `invalid.email` (missing @)

### Password Mismatch

The script asks you to confirm your password. If they don't match:
1. Script will notify you
2. You'll be prompted to enter both again

### Generated .env Not Working

1. Check file location - should be in project root:
   ```
   sereni-base/
   ├── .env          ← Should be here
   ├── docker-compose.yaml
   ├── README.md
   └── ...
   ```

2. Verify file contents:
   ```bash
   # Windows
   type .env
   
   # Linux/macOS
   cat .env
   ```

3. Restart services:
   ```bash
   docker-compose down
   docker-compose up -d
   ```

## 🎓 Learning Path

1. **Start Here:** Run the interactive setup script
2. **Understand Basics:** Read [Environment Setup Guide](./docs/ENVIRONMENT_SETUP_GUIDE.md)
3. **Deep Dive:** Check [Environment Variables](./docs/ENVIRONMENT_VARIABLES.md)
4. **Production:** Follow security checklist in documentation

## 💡 Tips

### Quick Testing
```bash
# Accept all defaults for fast testing
# Just press Enter for every prompt
```

### LAN Access
```bash
# When prompted for IP, enter your local IP
# Find your IP:
# - Windows: ipconfig
# - Linux/macOS: ip addr or ifconfig
```

### Production Deployment
- Use strong passwords (16+ characters)
- Use 64-character JWT secret
- Configure email for password reset
- Use external database with backups
- Use cloud storage (AWS S3 or MinIO)
- Restrict CORS origins

### Regenerate Configuration
```bash
# Delete old .env
rm .env

# Run script again
.\build\scripts\setup-interactive.ps1  # Windows
./build/scripts/setup-interactive.sh   # Linux/macOS
```

## 🤝 Need Help?

- 📖 Check [Environment Setup Guide](../docs/ENVIRONMENT_SETUP_GUIDE.md)
- 📚 Read [main README](../README.md)
- 🔍 See example configurations in documentation

---

**Happy Configuring! 🎉**
