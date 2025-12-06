# LCP Server - Production Backup Strategy

This document outlines various backup strategies for the LCP Server MySQL database in production.

## Quick Start

### 1. Manual Backup
```bash
# Make scripts executable
chmod +x scripts/*.sh

# Create a backup
./scripts/backup-mysql.sh

# Restore from backup
./scripts/restore-mysql.sh /opt/lcp-backups/mysql_backup_20241128_143000.sql.gz
```

### 2. Automated Backups
```bash
# Add to crontab for daily backups at 2 AM
crontab -e
# Add this line:
0 2 * * * /path/to/lcp-server/scripts/automated-backup.sh
```

## Backup Strategies

### Strategy 1: Local File System Backups

**Pros:**
- Simple to implement
- Fast backup and restore
- No external dependencies

**Cons:**
- Single point of failure
- Limited by local disk space

**Implementation:**
- Use `scripts/backup-mysql.sh` for manual backups
- Use `scripts/automated-backup.sh` with cron for automation

### Strategy 2: Docker Volume Backups

Add volume backup to your docker-compose:

```yaml
services:
  backup:
    image: alpine:latest
    volumes:
      - db-data:/data/mysql:ro
      - ./backups:/backup
    command: >
      sh -c "
        apk add --no-cache tar gzip &&
        tar -czf /backup/mysql-volume-backup-$$(date +%Y%m%d_%H%M%S).tar.gz -C /data mysql &&
        find /backup -name 'mysql-volume-backup-*.tar.gz' -mtime +30 -delete
      "
    profiles:
      - backup
```

Run with: `docker compose --profile backup run --rm backup`

### Strategy 3: Cloud Storage Integration

#### AWS S3 Integration

1. Install AWS CLI in your backup script:
```bash
# In automated-backup.sh, uncomment and configure:
aws s3 cp "$LATEST_BACKUP" s3://your-backup-bucket/lcp-backups/
```

2. Set up AWS credentials:
```bash
aws configure
# or use IAM roles for EC2 instances
```

#### Google Cloud Storage
```bash
# Upload to GCS
gsutil cp "$LATEST_BACKUP" gs://your-backup-bucket/lcp-backups/
```

### Strategy 4: Remote Server Backup

Using rsync to sync backups to a remote server:

```bash
# In automated-backup.sh
rsync -avz --delete /opt/lcp-backups/ user@backup-server:/backups/lcp/
```

### Strategy 5: Database Replication

For high availability, set up MySQL replication:

```yaml
# In compose.yaml - add a replica
mysql-replica:
  image: mysql:8.4.7
  restart: always
  environment:
    - MYSQL_ROOT_PASSWORD_FILE=/run/secrets/mysql-root-password
  command: >
    --server-id=2
    --log-bin=mysql-bin
    --binlog-do-db=lcpserver
    --read-only=1
  volumes:
    - replica-data:/var/lib/mysql
  secrets:
    - mysql-root-password
```

## Production Setup

### 1. Backup Schedule

Recommended cron schedule:
```bash
# Daily backup at 2 AM
0 2 * * * /opt/lcp-server/scripts/automated-backup.sh

# Weekly full backup on Sunday at 1 AM
0 1 * * 0 /opt/lcp-server/scripts/backup-mysql.sh weekly_$(date +\%Y\%m\%d)

# Monthly archive on 1st day at midnight
0 0 1 * * /opt/lcp-server/scripts/backup-mysql.sh monthly_$(date +\%Y\%m)
```

### 2. Monitoring and Alerting

Add monitoring to your backup script:

```bash
# Check backup file size
MIN_SIZE=1024  # 1KB minimum
BACKUP_SIZE=$(stat -c%s "$COMPRESSED_BACKUP")
if [ "$BACKUP_SIZE" -lt "$MIN_SIZE" ]; then
    echo "WARNING: Backup file is suspiciously small"
    # Send alert
fi

# Send success notification
curl -X POST https://hooks.slack.com/... \
  -H 'Content-type: application/json' \
  --data '{"text":"LCP MySQL backup completed successfully"}'
```

### 3. Security Considerations

1. **Encrypt backups:**
```bash
# Encrypt backup file
gpg --symmetric --cipher-algo AES256 "$BACKUP_FILE"
```

2. **Secure backup location:**
```bash
# Set proper permissions
chmod 600 /opt/lcp-backups/*.sql.gz
chown backup-user:backup-group /opt/lcp-backups/
```

3. **Rotate passwords:**
- Regularly update MySQL passwords
- Use different credentials for backup operations

### 4. Testing Backups

Regular backup testing is crucial:

```bash
# Test script
#!/bin/bash
TEST_DB="lcpserver_test_restore"

# Create test database and restore
docker compose exec mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "
CREATE DATABASE $TEST_DB;
"

# Restore backup to test database
zcat "$LATEST_BACKUP" | docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" "$TEST_DB"

# Verify data integrity
TABLES_COUNT=$(docker compose exec -T mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$TEST_DB';
" | tail -1)

echo "Test restore completed. Tables: $TABLES_COUNT"

# Cleanup
docker compose exec mysql mysql -u root -p"$MYSQL_ROOT_PASSWORD" -e "DROP DATABASE $TEST_DB;"
```

## Disaster Recovery Plan

### 1. Full System Recovery

```bash
# 1. Deploy fresh system
git clone <your-repo>
cd lcp-server
docker compose up -d mysql

# 2. Restore database
./scripts/restore-mysql.sh /path/to/latest/backup.sql.gz

# 3. Start all services
docker compose up -d
```

### 2. Point-in-Time Recovery

For point-in-time recovery, enable binary logging:

```yaml
# In mysql service
command: >
  --log-bin=mysql-bin
  --binlog-expire-logs-seconds=604800
  --max_binlog_size=100M
```

### 3. Recovery Time Objectives (RTO/RPO)

- **RPO (Recovery Point Objective)**: Maximum 24 hours (daily backups)
- **RTO (Recovery Time Objective)**: Target 1 hour for full system recovery

## Backup Verification

Implement automated backup verification:

```bash
#!/bin/bash
# verify-backup.sh
BACKUP_FILE="$1"

# Extract first 100 lines and check for valid SQL
zcat "$BACKUP_FILE" | head -100 | grep -q "CREATE TABLE"
if [ $? -eq 0 ]; then
    echo "Backup verification: PASSED"
else
    echo "Backup verification: FAILED"
    exit 1
fi
```

## Storage Requirements

Estimate storage needs:
- Database size: ~100MB typical
- Compressed backup: ~20MB typical
- 30 days retention: ~600MB
- Weekly archives: +80MB/month
- Monthly archives: +20MB/month

Plan for 2-3x growth factor.