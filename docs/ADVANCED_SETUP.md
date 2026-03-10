# Advanced Setup Configuration Guide

## Overview

The SereniBase setup script now supports advanced configuration options for database, storage, and antivirus services. Users can choose between using containerized services or connecting to existing external services.

---

## Configuration Options

### 1. Network Configuration
**Required for all setups**

- **PUBLIC_HOST**: Your server's IP address or domain name
  - Examples: `localhost`, `192.168.1.100`, `serenibase.example.com`
  - Used for: Frontend access, backend API, CORS configuration

---

### 2. Database Configuration
**Choose how to run PostgreSQL**

#### Option 1: Create New PostgreSQL Instance (Default, Recommended)
- A PostgreSQL 15 container will be created
- Default credentials: `postgres/postgres`
- Database name: `serenibase`
- No additional configuration needed
- **Use Case**: Development, testing, quick start

#### Option 2: Use Existing PostgreSQL Database
- Connect to your own PostgreSQL server
- **Required Information:**
  - Database Host (IP or hostname)
  - Database Port (default: 5432)
  - Database Name
  - Database User
  - Database Password
- **Use Case**: Production deployments, existing infrastructure

**Configuration Variables:**
```env
DATABASE_HOST=<your-db-host>
DATABASE_PORT=5432
DATABASE_USER=<your-username>
DATABASE_PASSWORD=<your-password>
DATABASE_NAME=<your-database>
```

---

### 3. Storage Configuration
**Choose where to store uploaded files**

#### Option 1: Local Filesystem (Default)
- Files stored inside the container at `/app/uploads`
- No additional services needed
- **Pros**: Simple, no external dependencies
- **Cons**: Data lost if container is removed (use volumes in production)
- **Use Case**: Development, testing, single-server deployments

#### Option 2: Create New MinIO Instance
- MinIO S3-compatible storage container will be created
- Default credentials: `minioadmin/minioadmin`
- Console accessible at: `http://PUBLIC_HOST:9001`
- **Pros**: S3-compatible, scalable, web interface
- **Cons**: Additional container to manage
- **Use Case**: Production with multiple servers, cloud-like storage

#### Option 3: Use Existing MinIO Server
- Connect to your own MinIO deployment
- **Required Information:**
  - MinIO Endpoint (host:port)
  - Access Key
  - Secret Key
  - Bucket Name
  - Use SSL (yes/no)
- **Use Case**: Existing MinIO infrastructure, shared storage

#### Option 4: AWS S3
- Use Amazon S3 for file storage
- **Required Information:**
  - AWS Region (e.g., `us-east-1`)
  - S3 Bucket Name
  - AWS Access Key ID
  - AWS Secret Access Key
- **Use Case**: Cloud deployments, AWS infrastructure

**Configuration Variables:**
```env
# For Local Storage
STORAGE_DRIVER=local
STORAGE_DEV_PATH=./uploads

# For MinIO
STORAGE_DRIVER=minio
STORAGE_MINIO_ENDPOINT=minio:9000
STORAGE_MINIO_ACCESS_KEY=minioadmin
STORAGE_MINIO_SECRET_KEY=minioadmin
STORAGE_MINIO_BUCKET=serenibase
STORAGE_MINIO_USE_SSL=false

# For AWS S3
STORAGE_DRIVER=aws
STORAGE_AWS_REGION=us-east-1
STORAGE_AWS_BUCKET=my-bucket
STORAGE_AWS_ACCESS_KEY=<your-access-key>
STORAGE_AWS_SECRET_KEY=<your-secret-key>
```

**Storage Service Host:**
The storage service itself always runs at `http://sereni-storage-provider:8083` within the Docker network. The `STORAGE_DRIVER` determines where files are actually stored.

---

### 4. Antivirus Configuration
**Choose how to scan uploaded files**

#### Option 1: Create New ClamAV Instance (Default, Recommended)
- ClamAV container will be created for malware scanning
- Automatic virus definition updates
- No additional configuration needed
- **Pros**: Security, automatic updates
- **Cons**: Additional resource usage (~1GB RAM)
- **Use Case**: Production, security-conscious deployments

#### Option 2: Use Existing ClamAV Server
- Connect to your own ClamAV daemon
- **Required Information:**
  - ClamAV Host (IP or hostname)
  - ClamAV Port (default: 3310)
- **Use Case**: Existing antivirus infrastructure, shared ClamAV

#### Option 3: Disable Antivirus Scanning
- No antivirus scanning will be performed
- ⚠️ **WARNING**: Files will NOT be scanned for malware!
- **Use Case**: Development only, trusted environments

**Configuration Variables:**
```env
ANTIVIRUS_DRIVER=clamav
ANTIVIRUS_CLAMAV_ADDRESS=clamav:3310

# If using external ClamAV
ANTIVIRUS_CLAMAV_ADDRESS=<your-host>:3310
```

---

## Example Setup Flows

### Quick Start (All Defaults)
```
Network Configuration
  IP/Domain: localhost

Owner Account
  Name: Admin User
  Email: admin@example.com
  Password: Admin@123

Email: Skip (N)
Database: Create new instance (1)
Storage: Local filesystem (1)
Antivirus: Create ClamAV (1)

✅ Setup complete in < 5 minutes
```

### Production with External Database
```
Network Configuration
  IP/Domain: serenibase.company.com

Owner Account
  Name: John Doe
  Email: john@company.com
  Password: <secure-password>

Email: Configure SMTP (Y)
  SMTP details...

Database: Use existing (2)
  Host: db.company.com
  Port: 5432
  Name: serenibase_prod
  User: serenibase_user
  Password: <db-password>

Storage: Create MinIO (2)
  (Uses default MinIO container)

Antivirus: Create ClamAV (1)
  (Uses default ClamAV container)

✅ Production-ready setup
```

### AWS Cloud Deployment
```
Network Configuration
  IP/Domain: app.example.com

Owner Account
  Details...

Email: Configure (Y)
  SMTP details...

Database: Use existing RDS (2)
  Host: mydb.abc123.us-east-1.rds.amazonaws.com
  Details...

Storage: AWS S3 (4)
  Region: us-east-1
  Bucket: myapp-uploads
  Access Key: AKIA...
  Secret Key: ...

Antivirus: Create ClamAV (1)
  (Still needs container for AV service)

✅ Cloud-optimized setup
```

---

## Service Dependencies

### What Gets Deployed Based on Your Choices

| Configuration | Containers Deployed |
|--------------|-------------------|
| **Minimal** (All external) | base-ui, serenibase, jwt-provider, email-service, sereni-storage-provider, antivirus-service |
| **+ New Database** | + postgres |
| **+ New MinIO** | + minio |
| **+ New ClamAV** | + clamav |
| **Full Stack** | All containers |

### Container Scaling

The setup script automatically scales services to 0 (disabled) based on your choices:
- External database → postgres=0
- External MinIO or non-MinIO storage → minio=0
- External ClamAV → clamav=0
- Disabled antivirus → clamav=0 + antivirus-service=0

---

## Port Requirements

### Required Ports (Always Needed)
- **5050**: Frontend (base-ui)
- **8080**: Backend API (serenibase)
- **8081**: Auth service (jwt-provider)
- **8082**: Email service
- **8083**: Storage service
- **8084**: Antivirus service

### Optional Ports (Based on Configuration)
- **5432**: PostgreSQL (if using new database)
- **9000**: MinIO API (if using new MinIO)
- **9001**: MinIO Console (if using new MinIO)
- **3310**: ClamAV (if using new ClamAV)

---

## Security Best Practices

### Development/Testing
✅ Use all default containers
✅ Use localhost for PUBLIC_HOST
✅ Default passwords are acceptable

### Production
✅ Use strong, unique passwords for all services
✅ Use proper domain name with SSL/TLS
✅ Change default database credentials
✅ Change MinIO credentials if used
✅ Generate strong JWT secret (32+ characters)
✅ Use external managed database (RDS, Cloud SQL, etc.)
✅ Use cloud storage (S3, Google Cloud Storage)
✅ Keep ClamAV virus definitions updated
✅ Regular backups
✅ Firewall configuration
✅ Container resource limits

---

## Troubleshooting

### Issue: Can't connect to external database
**Solution:** 
- Verify host/port are correct
- Check firewall rules allow connection
- Ensure database exists and user has permissions
- Test connection: `psql -h HOST -p PORT -U USER -d DATABASE`

### Issue: MinIO connection fails
**Solution:**
- Verify endpoint format (host:port, no http://)
- Check SSL setting matches your MinIO setup
- Verify access/secret keys are correct
- Ensure bucket exists

### Issue: ClamAV not scanning files
**Solution:**
- Check ClamAV container is running: `docker ps | grep clamav`
- Verify virus definitions are updated
- Check antivirus service logs: `docker logs antivirus-service`

### Issue: Storage service can't write files
**Solution:**
- For local: Check volume permissions
- For MinIO: Verify credentials and bucket exists
- For S3: Verify IAM permissions for bucket operations

---

## Advanced: Manual Configuration

All these settings can be manually configured by editing `.env` file:

```bash
# Edit .env file
vi .env

# Update any configuration
PUBLIC_HOST=your-domain.com
DATABASE_HOST=your-db-host
STORAGE_DRIVER=minio
# ... etc

# Restart services
make down-all
make up-all
```

---

## Support

For issues or questions:
- Check logs: `make logs`
- View running services: `docker ps`
- Review `.env` configuration
- Consult documentation at: `docs/`
