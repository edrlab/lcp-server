# Configuration Secrets

This directory contains sensitive configuration files used as Docker secrets.

## Required Files

### `access.txt`
Basic authentication credentials for LCP API access and JWT dashboard accounts.

**Format:**
```
username:password
dashboard_user1:password1
dashboard_user2:password2
```

First line: Basic auth credentials for API access
Following lines: Dashboard admin accounts (username:password pairs)

### `mysql-root-password.txt`
MySQL root password.

**Example:**
```
your_strong_root_password_here
```

### `mysql-password.txt`
Password for the MySQL user defined in `MYSQL_USER` (typically `lcp_user`).

**Example:**
```
your_strong_user_password_here
```

### `jwt-secretkey.txt`
Secret key used to sign JWT tokens for dashboard authentication.

**Example:**
```
your_strong_random_jwt_secret_key_minimum_32_characters
```

**⚠️ Important:** Generate a strong random key of at least 32 characters.

## Security Best Practices

1. **Never commit these files to version control** (already ignored in `.gitignore`)
2. **Use strong, random passwords** - minimum 16 characters
3. **Different passwords for each secret**
4. **Generate JWT secret key with:**
   ```bash
   openssl rand -base64 48
   ```
5. **Restrict file permissions:**
   ```bash
   chmod 600 config/*.txt
   ```
6. **Rotate secrets regularly** in production
7. **Backup secrets securely** (encrypted storage)

## Initial Setup

1. Copy example files or create new ones:
   ```bash
   cd config/
   echo "your_mysql_root_password" > mysql-root-password.txt
   echo "your_mysql_user_password" > mysql-password.txt
   echo "apiuser:apipassword" > access.txt
   openssl rand -base64 48 > jwt-secretkey.txt
   chmod 600 *.txt
   ```

2. Update `config-vm/` directory with production values if needed

## How Secrets Are Used

- **Docker Compose** mounts these files to `/run/secrets/` inside containers
- **LCP Server** reads secrets at startup via environment variables:
  - `LCPSERVER_ACCESS_FILE` → `/run/secrets/access`
  - `MYSQL_PASSWORD_FILE` → `/run/secrets/mysql-password`
  - `JWT_SECRETKEY_FILE` → `/run/secrets/jwt-secretkey`
- Passwords are **never** stored in environment variables or code
