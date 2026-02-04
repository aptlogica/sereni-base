# 🔧 Build Directory

This directory contains all build-related files, scripts, and configurations for SereniBase.

## 📁 Structure

```
build/
├── config/                    # Configuration templates
│   └── .env.example          # Environment variables template
│
├── scripts/                   # Setup and utility scripts
│   ├── setup.sh              # Basic setup (Linux/macOS)
│   ├── setup.bat             # Basic setup (Windows CMD)
│   ├── setup-interactive.ps1 # Interactive setup wizard (Windows PowerShell)
│   ├── setup-interactive.sh  # Interactive setup wizard (Linux/macOS)
│   ├── setup-all.ps1         # Full setup (Windows PowerShell)
│   ├── clone-services.sh     # Clone microservices (Bash)
│   ├── clone-services.ps1    # Clone microservices (PowerShell)
│   ├── clone-go-postgres-rest.sh   # Clone go-postgres-rest (Bash)
│   ├── clone-go-postgres-rest.ps1  # Clone go-postgres-rest (PowerShell)
│   ├── services.list         # List of microservices to clone
│   ├── test-ctrl-c.ps1       # Test script for Ctrl+C handling
│   └── test-ui-preview.ps1   # Test script for UI preview
│
├── INTERACTIVE_SETUP_README.md  # Documentation for interactive setup
├── CTRL_C_FIX_NOTES.md          # Notes on Ctrl+C handling fixes
├── UI_IMPROVEMENTS.md           # Documentation for UI improvements
└── README.md                    # This file
```

## 🚀 Usage

### From Project Root

```bash
# Linux/macOS - Interactive setup
./build/scripts/setup-interactive.sh

# Windows PowerShell - Interactive setup
.\build\scripts\setup-interactive.ps1

# Windows CMD - Basic setup
build\scripts\setup.bat

# Windows PowerShell - Automated full setup
.\build\scripts\setup-all.ps1
```

### Running Scripts Directly

#### Linux/macOS (Bash)
```bash
# Make scripts executable
chmod +x build/scripts/*.sh

# Run interactive setup
./build/scripts/setup-interactive.sh

# Run basic setup
./build/scripts/setup.sh

# Clone services only
./build/scripts/clone-services.sh
```

#### Windows (PowerShell)
```powershell
# Run interactive setup
.\build\scripts\setup-interactive.ps1

# Run automated setup
.\build\scripts\setup-all.ps1

# Clone services only
.\build\scripts\clone-services.ps1
```

## 📋 Scripts Description

| Script | Platform | Description |
|--------|----------|-------------|
| `setup-interactive.ps1` | Windows PS | Interactive setup wizard with prompts |
| `setup-interactive.sh` | Linux/macOS | Interactive setup wizard with prompts |
| `setup.sh` | Linux/macOS | Basic setup script |
| `setup.bat` | Windows CMD | Basic setup for Command Prompt |
| `setup-all.ps1` | Windows PS | Automated full setup |
| `clone-services.sh` | Linux/macOS | Clones all microservices from services.list |
| `clone-services.ps1` | Windows | Clones all microservices from services.list |
| `clone-go-postgres-rest.sh` | Linux/macOS | Clones go-postgres-rest repository |
| `clone-go-postgres-rest.ps1` | Windows | Clones go-postgres-rest repository |
| `services.list` | All | List of microservices to clone |

## ⚙️ Configuration

The `config/.env.example` file contains all environment variables with documentation.

To create your configuration:
```bash
cp build/config/.env.example .env
```

See [docs/ENV_CONFIGURATION.md](../docs/ENV_CONFIGURATION.md) for detailed documentation.
