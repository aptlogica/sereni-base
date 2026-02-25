# Build Scripts

Main setup docs:
- `build/SETUP_COMPLETE_GUIDE.md` (full beginner guide)
- `build/SETUP.md` (existing reference guide)

Primary scripts:
- `build/scripts/setup.sh` (interactive Linux/macOS)
- `build/scripts/setup.ps1` (interactive Windows PowerShell)
- `build/scripts/setup-y.sh` (auto defaults Linux/macOS)
- `build/scripts/setup-y.ps1` (auto defaults Windows PowerShell)
- `build/scripts/clone-services.sh`
- `build/scripts/clone-services.ps1`

Typical usage:

Linux/macOS:

```bash
./build/scripts/setup.sh
```

Windows PowerShell:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup.ps1
```
