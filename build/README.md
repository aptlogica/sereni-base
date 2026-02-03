# 🔧 Build Directory

This directory contains all build-related files, scripts, and configurations for SereniBase.

## 📁 Structure

```
build/
├── config/                    # Configuration templates
│   └── .env.example          # Environment variables template
│
├── scripts/                   # Setup and utility scripts
│   ├── setup.sh              # Interactive setup (Linux/macOS)
│   ├── setup.bat             # Interactive setup (Windows CMD)
│   ├── setup-all.ps1         # Full setup (Windows PowerShell)
│   ├── clone-services.sh     # Clone microservices (Bash)
│   ├── clone-services.ps1    # Clone microservices (PowerShell)
│   ├── clone-go-postgres-rest.sh   # Clone go-postgres-rest (Bash)
│   └── clone-go-postgres-rest.ps1  # Clone go-postgres-rest (PowerShell)
│
└── README.md                  # This file
```

## 🚀 Usage

### From Project Root

```bash
# Linux/macOS - Interactive setup
make setup

# Linux/macOS - Automated setup
make setup-all

# Windows CMD
build\scripts\setup.bat

# Windows PowerShell
.\build\scripts\setup-all.ps1
```

### Running Scripts Directly

#### Linux/macOS (Bash)
```bash
# Make scripts executable
chmod +x build/scripts/*.sh

# Run setup
./build/scripts/setup.sh

# Clone services only
./build/scripts/clone-services.sh
```

#### Windows (PowerShell)
```powershell
# Run setup
.\build\scripts\setup-all.ps1

# Clone services only
.\build\scripts\clone-services.ps1
```

## 📋 Scripts Description

| Script | Platform | Description |
|--------|----------|-------------|
| `setup.sh` | Linux/macOS | Interactive setup wizard with prompts |
| `setup.bat` | Windows CMD | Interactive setup wizard for Command Prompt |
| `setup-all.ps1` | Windows PS | Automated full setup |
| `clone-services.sh` | Linux/macOS | Clones all microservices from services.list |
| `clone-services.ps1` | Windows | Clones all microservices from services.list |
| `clone-go-postgres-rest.sh` | Linux/macOS | Clones go-postgres-rest repository |
| `clone-go-postgres-rest.ps1` | Windows | Clones go-postgres-rest repository |

## ⚙️ Configuration

The `config/.env.example` file contains all environment variables with documentation.

To create your configuration:
```bash
cp build/config/.env.example .env
```

See [docs/ENV_CONFIGURATION.md](../docs/ENV_CONFIGURATION.md) for detailed documentation.
