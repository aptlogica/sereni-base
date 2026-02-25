# SereniBase Complete Setup Guide (Windows, macOS, Linux)

This guide is for a fresh machine and first-time users.
Follow exactly in order.

## 1. Minimum Requirements

- Docker Desktop (or Docker Engine + Docker Compose plugin)
- Git
- Make (optional but recommended)
- At least 8 GB RAM available to Docker
- At least 20 GB free disk

Check tools:

```bash
docker --version
docker compose version
git --version
```

If `docker compose` fails, install/enable Docker Compose plugin first.

## 2. Clone Repository

```bash
git clone https://github.com/aptlogica/sereni-base.git
cd sereni-base
```

## 3. Run Setup (Recommended)

### Windows (PowerShell)

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup.ps1
```

### macOS / Linux

```bash
chmod +x build/scripts/setup.sh build/scripts/setup-y.sh
./build/scripts/setup.sh
```

### Non-interactive defaults

Windows:

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup-y.ps1
```

macOS/Linux:

```bash
./build/scripts/setup-y.sh
```

## 4. Access URLs

- Frontend: `http://localhost:5050`
- API: `http://localhost:8080`
- Health: `http://localhost:8080/api/v1/health`
- MinIO Console: `http://localhost:9001`

## 5. Verify Containers

```bash
docker compose -f docker-compose.all.yaml ps
```

Expected: all services should be `Up` (some may show `healthy` after a short delay).

If any service is `unhealthy`, `exited`, or restarting:

```bash
docker compose -f docker-compose.all.yaml logs --tail=200 <service-name>
```

## 6. Day-2 Commands

Start:

```bash
docker compose -f docker-compose.all.yaml up -d
```

Stop:

```bash
docker compose -f docker-compose.all.yaml down
```

Rebuild after code changes:

```bash
docker compose -f docker-compose.all.yaml up --build -d
```

Hard reset (delete local volumes/data):

```bash
docker compose -f docker-compose.all.yaml down -v
```

## 7. Troubleshooting

### A) `dockerfile parse error ... unknown instruction: server`

Cause:
- Old Docker parser does not support heredoc syntax used in some Dockerfiles.

Status in this repo:
- Fixed by using `services/base-ui/nginx.default.conf` + `COPY` in `services/base-ui/Dockerfile`.

What to do:
1. Pull latest repo changes.
2. Rebuild:
   ```bash
   docker compose -f docker-compose.all.yaml build --no-cache base-ui
   docker compose -f docker-compose.all.yaml up -d
   ```

### B) Login fails with JWT/auth errors

Common causes:
- `AUTH_JWT_SECRET` changed between runs.
- Auth container not healthy.
- App is using stale token from old setup.

Fix:
1. Check auth container:
   ```bash
   docker compose -f docker-compose.all.yaml ps jwt-provider
   docker compose -f docker-compose.all.yaml logs --tail=200 jwt-provider
   ```
2. Check `.env` has one stable value for `AUTH_JWT_SECRET`.
3. If you changed secret, stop containers and restart:
   ```bash
   docker compose -f docker-compose.all.yaml down
   docker compose -f docker-compose.all.yaml up -d
   ```
4. Clear browser local storage/session for base-ui and login again.

### C) Some containers start, some don’t

Common causes:
- Port conflicts (5050, 8080, 8081, 8082, 8083, 8084, 5432, 9000, 9001, 3310).
- Low Docker memory/CPU.
- Stale volumes or broken previous state.

Fix order:
1. Check container states:
   ```bash
   docker compose -f docker-compose.all.yaml ps
   ```
2. Check logs for failing service:
   ```bash
   docker compose -f docker-compose.all.yaml logs --tail=200 <service-name>
   ```
3. Resolve port conflicts, then restart.
4. If still broken:
   ```bash
   docker compose -f docker-compose.all.yaml down -v
   docker compose -f docker-compose.all.yaml up --build -d
   ```

### D) Windows-specific `.env` weird behavior

Cause:
- UTF-8 BOM in `.env` can break first variable parsing on some tools.

Status in this repo:
- Setup scripts now write `.env` as UTF-8 **without BOM**.

If you already have a bad `.env`, regenerate:

```powershell
Remove-Item .env -Force
powershell -NoProfile -ExecutionPolicy Bypass -File .\build\scripts\setup.ps1
```

### E) macOS/Linux script not executable

```bash
chmod +x build/scripts/*.sh
```

## 8. Clean From-Scratch Reinstall

Use this when moving between machines or fixing unknown state.

```bash
docker compose -f docker-compose.all.yaml down -v
docker system prune -f
git pull
```

Then rerun setup script from section 3.

## 9. Files to Check for Setup Problems

- `build/scripts/setup.sh`
- `build/scripts/setup.ps1`
- `build/scripts/append-env-vars.ps1`
- `docker-compose.all.yaml`
- `services/base-ui/Dockerfile`
- `services/base-ui/nginx.default.conf`

