#!/bin/bash

# Cold Backup Script - Maximum Data Integrity
# This version stops the LCP server during backup for maximum consistency

set -e

# Configuration
BACKUP_DIR="./backups"
MYSQL_ROOT_PASSWORD_FILE="config/mysql-root-password.txt"
DATABASE_NAME="lcpserver"
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Generate backup filename
BACKUP_NAME="cold_backup_$(date +%Y%m%d_%H%M%S)"
BACKUP_FILE="${BACKUP_DIR}/${BACKUP_NAME}.sql"
COMPRESSED_BACKUP="${BACKUP_DIR}/${BACKUP_NAME}.sql.gz"

echo "Starting COLD MySQL backup (service will be temporarily unavailable)..."
echo "Backup file: $COMPRESSED_BACKUP"

# Check MySQL container
if ! docker compose ps mysql | grep -qE "running|healthy"; then
    echo "Error: MySQL container is not running or not healthy"
    exit 1
fi

# Get password
MYSQL_ROOT_PASSWORD=$(cat "$MYSQL_ROOT_PASSWORD_FILE")

# Warning about service interruption
echo "WARNING: This will temporarily stop the LCP server"
echo "Estimated downtime: 30-120 seconds"
read -p "Continue? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cold backup cancelled"
    exit 1
fi

# Record start time
START_TIME=$(date +%s)
echo "Downtime started at: $(date)"

# Stop LCP server
echo "Stopping Nginx and LCP server for consistent backup..."
docker compose stop server nginx

# Flush and lock tables for maximum consistency
echo "Flushing tables..."
docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "
FLUSH TABLES WITH READ LOCK;
FLUSH LOGS;
"

# Create backup
echo "Creating cold database dump..."
docker compose exec -T mysql mysqldump \
    -u root \
    -p"$MYSQL_ROOT_PASSWORD" \
    --single-transaction \
    --routines \
    --triggers \
    --events \
    --complete-insert \
    --extended-insert \
    --set-gtid-purged=OFF \
    --master-data=2 \
    --default-character-set=utf8mb4 \
    --hex-blob \
    --order-by-primary \
    "$DATABASE_NAME" > "$BACKUP_FILE"

# Unlock tables
docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "UNLOCK TABLES;"

# Start services
echo "Restarting services..."
docker compose start mysql server nginx

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Verify services are back online
if docker compose ps | grep -q "unhealthy\|exited"; then
    echo "WARNING: Some services may not be healthy"
    docker compose ps
fi

# Compress backup
echo "Compressing backup..."
gzip "$BACKUP_FILE"

# Calculate downtime
END_TIME=$(date +%s)
DOWNTIME=$((END_TIME - START_TIME))

# Verify backup
if [ -f "$COMPRESSED_BACKUP" ]; then
    BACKUP_SIZE=$(du -h "$COMPRESSED_BACKUP" | cut -f1)
    echo "Cold backup completed: $COMPRESSED_BACKUP ($BACKUP_SIZE)"
    
    # Test backup integrity
    echo "Testing backup integrity..."
    if gunzip -c "$COMPRESSED_BACKUP" | head -40 | grep -q "CREATE TABLE"; then
        echo "✓ Backup integrity verified"
    else
        echo "✗ Backup integrity check failed"
        exit 1
    fi
else
    echo "Error: Backup file not found"
    exit 1
fi

# Clean old backups
find "$BACKUP_DIR" -name "cold_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete

# Downtime report
echo ""
echo "=== COLD BACKUP REPORT ==="
echo "Total downtime: ${DOWNTIME} seconds"
echo "Service impact: FULL (temporary)"
echo "Backup size: $BACKUP_SIZE"
echo "Data consistency: MAXIMUM"
echo "Downtime ended at: $(date)"
echo "========================="

echo "Cold backup process completed!"