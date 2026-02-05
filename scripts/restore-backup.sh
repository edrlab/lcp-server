#!/bin/bash

# MySQL Restore Script for LCP Server
# Usage: ./restore-mysql.sh <backup_file.sql.gz>

set -e

# Configuration
CONTAINER_NAME="lcp-mysql-1"
MYSQL_ROOT_PASSWORD_FILE="config/mysql-root-password.txt"
DATABASE_NAME="lcpserver"

# Check arguments
if [ $# -ne 1 ]; then
    echo "Usage: $0 <backup_file.sql.gz>"
    echo "Example: $0 ./backups/mysql_backup_20241128_143000.sql.gz"
    exit 1
fi

BACKUP_FILE="$1"

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo "Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

# Check MySQL container
if ! docker compose ps mysql | grep -qE "running|healthy"; then
    echo "Error: MySQL container is not running or not healthy"
    exit 1
fi

# Get MySQL root password
if [ ! -f "$MYSQL_ROOT_PASSWORD_FILE" ]; then
    echo "Error: MySQL root password file not found: $MYSQL_ROOT_PASSWORD_FILE"
    exit 1
fi

MYSQL_ROOT_PASSWORD=$(cat "$MYSQL_ROOT_PASSWORD_FILE")

echo "Starting MySQL restore..."
echo "Backup file: $BACKUP_FILE"
echo "Target database: $DATABASE_NAME"
echo ""

# Warning message
read -p "WARNING: This will replace all data in database '$DATABASE_NAME'. Are you sure? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Restore cancelled"
    exit 1
fi

# Stop LCP server temporarily
echo "Stopping LCP server..."
docker compose stop server

# Drop and recreate database
echo "Recreating database..."
docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "
DROP DATABASE IF EXISTS $DATABASE_NAME;
CREATE DATABASE $DATABASE_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
"

# Restore from backup
echo "Restoring database from backup..."
if [[ "$BACKUP_FILE" == *.gz ]]; then
    gunzip -c "$BACKUP_FILE" | docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" "$DATABASE_NAME"
else
    cat "$BACKUP_FILE" | docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" "$DATABASE_NAME"
fi

# Restart LCP server
echo "Restarting LCP server..."
docker compose start server

# Wait for server to be ready
echo "Waiting for services to be ready..."
sleep 10

# Verify restore
echo "Verifying restore..."
TABLES_COUNT=$(docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "
USE $DATABASE_NAME;
SELECT COUNT(*) AS table_count FROM information_schema.tables WHERE table_schema = '$DATABASE_NAME';
" | tail -1)

echo "Database restored successfully!"
echo "Tables found: $TABLES_COUNT"
echo ""
echo "You can verify the restore by checking:"
echo "- Application logs: docker compose logs server"
echo "- Database content: docker compose exec mysql mysql -u root -p\$(cat $MYSQL_ROOT_PASSWORD_FILE) $DATABASE_NAME"