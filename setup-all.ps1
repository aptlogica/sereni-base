# Clone all services and go-postgres-rest, then run docker compose up-all

# Clone main services
Write-Host "Cloning all service repos..."
powershell -NoProfile -ExecutionPolicy Bypass -File clone-services.ps1

# Clone or pull go-postgres-rest
Write-Host "Cloning or updating go-postgres-rest..."
powershell -NoProfile -ExecutionPolicy Bypass -File clone-go-postgres-rest.ps1

# Run docker compose up-all
Write-Host "Running all services with Docker Compose..."
docker compose -f docker-compose.all.yaml up --build
