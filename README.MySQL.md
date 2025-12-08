# MySQL Configuration for LCP Server

## Files in this directory

- `access.txt` - Basic authentication credentials for LCP server API
- `mysql-root-password.txt` - MySQL root password
- `mysql-password.txt` - Password for the `lcp_user` database user

## MySQL Setup

The Docker Compose configuration creates:
- A MySQL 8.4.7 Community server
- Database: `lcpserver`
- User: `lcp_user` with access to the `lcpserver` database
- Data stored in Docker volume: `db-data`, mapped to /var/lib/mysql

## Security Notes

**IMPORTANT**: Change the default passwords in the `.txt` files before running in production!

1. Edit `mysql-root-password.txt` with a strong root password (the default is `your_secure_root_password_change_this`)
2. Edit `mysql-password.txt` with a strong password for the lcp_user (the default is `lcp_user_password_change_this`)
3. Make sure these files have restricted permissions (600)

## Connection String Example

For the LCP server configuration, use a DSN like:
```
dsn: "lcp_user:password@tcp(mysql:3306)/lcpserver"
```

Where `password` matches the content of `mysql-password.txt`.