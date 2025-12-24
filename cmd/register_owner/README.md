# Owner Registration Script

This script allows you to pre-register an owner user for your Sereni-Base application without requiring OTP verification or going through the standard registration API flow.

## Overview

The owner registration script performs the following steps:
1. Creates a user in the master schema
2. Adds the user to the auth provider
3. Creates a tenant with a free subscription plan
4. Initializes the RBAC system for the tenant
5. Creates the user in the tenant schema
6. Assigns admin role to the user
7. Creates a default workspace
8. Marks the email as verified

This replicates the functionality of the `VerifyEmail` function from the authentication service, but without requiring OTP validation.

## Configuration

### 1. Update config.yaml

Add the following section to your `config.yaml` file:

```yaml
owner_registration:
  first_name: "Admin"
  last_name: "User"
  email: "admin@example.com"
  password: "Admin@123"
```

**Note:** Make sure to use a strong password and change these values to your actual owner details.

### 2. Set Environment Variables (Optional)

The script reads the country from the `COUNTRY` environment variable. If not set, it defaults to "US".

To set the country:

**Windows PowerShell:**
```powershell
$env:COUNTRY = "US"
```

**Linux/Mac:**
```bash
export COUNTRY="US"
```

## Running the Script

### Prerequisites

1. Ensure PostgreSQL database is running and accessible
2. Ensure the auth provider service is running
3. Ensure your `config.yaml` is properly configured with database and auth settings

### Execute the Script

**From the project root directory:**

```bash
go run cmd/register_owner/main.go
```

Or build and run:

```bash
go build -o register_owner cmd/register_owner/main.go
./register_owner
```

## Expected Output

When successful, you should see output similar to:

```
=== Owner Registration Script ===
Registering owner: Admin User (admin@example.com)

Step 1: Checking if user already exists...
Step 2: Creating user in master schema...
✓ User created in master schema with ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

Step 3: Adding user to auth provider...
✓ User added to auth provider

Step 4: Getting subscription plan...
✓ Found subscription plan: Free

Step 5: Getting admin role...
✓ Found admin role: Admin

Step 6: Initializing tenant...
✓ Tenant created with ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx, Schema: tenant_xxxxxxxx

Step 7: Initializing RBAC system...
✓ RBAC system initialized

Step 8: Creating user in tenant schema...
✓ User created in tenant schema

Step 9: Updating user status in master schema...
✓ User updated: Status=active, EmailVerified=true

Step 10: Assigning admin role to user...
✓ Admin role assigned to user

Step 11: Creating default workspace...
✓ Default workspace created with ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx

Step 12: Setting email as verified in auth provider...
✓ Email verified in auth provider

==================================================
✓ Owner registration completed successfully!
==================================================

Owner Details:
  Name:      Admin User
  Email:     admin@example.com
  User ID:   xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  Tenant ID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
  Schema:    tenant_xxxxxxxx
  Role:      Admin

You can now login with:
  Email:    admin@example.com
  Password: <configured password>
```

## Troubleshooting

### User Already Exists

If you see:
```
User with email admin@example.com already exists
```

The user has already been registered. You can either:
1. Use a different email in the config
2. Delete the existing user from the database
3. Use the existing credentials to login

### Database Connection Error

If you see database connection errors:
```
Failed to connect to database: ...
```

Ensure:
1. PostgreSQL is running
2. Database credentials in `config.yaml` are correct
3. The database exists

### Auth Provider Error

If you see auth provider errors:
```
Failed to connect to auth provider: ...
```

Ensure:
1. The auth provider service is running
2. Auth provider URL in `config.yaml` is correct

## Security Notes

1. **Never commit `config.yaml` with real credentials to version control**
2. Use environment variables or secrets management for production deployments
3. Change the default password immediately after first login
4. Use strong passwords for owner accounts
5. Keep the `config.yaml` file permissions restricted

## Config File Reference

The owner registration configuration in `config.yaml`:

| Field | Required | Description | Example |
|-------|----------|-------------|---------|
| `first_name` | Yes | Owner's first name | "Admin" |
| `last_name` | Yes | Owner's last name | "User" |
| `email` | Yes | Owner's email (used for login) | "admin@example.com" |
| `password` | Yes | Owner's password (use strong password) | "Admin@123" |

## Related Files

- `cmd/register_owner/main.go` - Registration script
- `internal/config/config.go` - Configuration structure
- `config.yaml` - Main configuration file
- `config.yaml.example` - Example configuration template
- `internal/services/auth_management.go` - Authentication service (VerifyEmail function)

## Notes

- This script should only be run once for initial setup
- Running it multiple times with the same email will result in an error
- The script automatically creates all necessary database schemas and tables
- The country code is obtained from the local machine's environment variable
- The timezone is automatically set based on the server's timezone
