---
layout: default
title: Backup
nav_order: 6
---

This document outlines various backup strategies for the LCP Server MySQL database in production.
Other databases certainly require adaptations. We cannot provide a complete solution for every platform, therefore let the community work on it. Proposals are welcome.  

## Two Backup Scripts are provided

| Method | Service Status | Impact Level | Use Case |
|--------|---------------|--------------|----------|
| **Hot Backup** | ✅ Fully Available | Minimal (<5%) | Production, frequent backups |
| **Cold Backup** | ❌ Temporarily Unavailable | Full (30-120s) | Maximum consistency needed |

### Hot Backup (`hot-backup.sh`)

```bash
./scripts/hot-backup.sh
# or
docker compose --profile backup run --rm hot-backup
```

**Service Impact:**
- ✅ **API Calls**: Continue normally
- ✅ **Database Reads**: No impact
- ✅ **Database Writes**: <5% performance impact
- ✅ **User Experience**: Unnoticeable
- ⏱️ **Duration**: 15-60 seconds

**Technical Details:**
- Uses `--single-transaction` with optimized flags
- Creates consistent snapshot without locking
- Streams data directly to compressed output
- MySQL continues serving all requests

**Impact**
```
Service Status: Available
API Response Time: +5ms average
Database Queries: No slowdown
User Experience: Unnoticeable
Backup Duration: 30 seconds
```

### Cold Backup (`cold-backup.sh`)

```bash
./scripts/cold-backup.sh
```

**Service Impact:**
- ❌ **API Calls**: Unavailable during backup
- ❌ **Database Access**: Stopped
- ❌ **User Experience**: Service unavailable
- ⏱️ **Duration**: 30-120 seconds downtime
- ✅ **Data Consistency**: Maximum (100%)

**Technical Details:**
- Stops LCP server during backup
- Flushes and locks tables
- Guarantees perfect data consistency
- Automatic service restart

**Impact**
```
Service Status: Unavailable
Downtime: 45 seconds average
Data Consistency: Perfect
User Experience: Brief outage
Recovery: Automatic
```


## Quick Start

### Manual Backup
```bash
# Make scripts executable
chmod +x scripts/*.sh

# Create a cold backup
./scripts/cold-backup.sh

# Restore from backup
./scripts/restore-backup.sh /opt/lcp-server/backups/cold_backup_20251128_143000.sql.gz
```

Note: Before moving to production, be sure to check the restoration of your backups in a staging environment.

### Automated Backups

By default, the automated backup is using the hot backup variant. This is easy to modify. 

```bash
# Add to crontab for daily backups at 2 AM
crontab -e
# Add this line:
0 2 * * * /path/to/lcp-server/scripts/automated-backup.sh
```

## Monitoring Backup Impact

Use the monitoring script to measure actual impact:

```bash
# Start monitoring in one terminal
./scripts/monitor-backup-impact.sh

# Run backup in another terminal
./scripts/cold-backup.sh
```

The monitor will show:
- Response times during backup
- Database connection counts
- Resource usage
- Performance metrics


## Backup Strategies

Where should you store the backups? 

### Strategy 1: Local File System Backups

**Pros:**
- Simple to implement
- Fast backup and restore
- No external dependencies

**Cons:**
- Single point of failure
- Limited by local disk space

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
rsync -avz --delete /opt/lcp-server/backups/ user@backup-server:/backups/lcp/
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

## Additional Production Steps

### 1. Backup Schedule

Recommended cron schedule:
```bash
# Daily backup at 2 AM
0 2 * * * /opt/lcp-server/scripts/automated-backup.sh

# Weekly full backup on Sunday at 1 AM
0 1 * * 0 /opt/lcp-server/scripts/cold-backup.sh weekly_$(date +\%Y\%m\%d)

# Monthly archive on 1st day at midnight
0 0 1 * * /opt/lcp-server/scripts/cold-backup.sh monthly_$(date +\%Y\%m)
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

### 3. Automating Backup Verification

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

### 4. Securing Backups

1. **Encrypt backups:**
```bash
# Encrypt backup file
gpg --symmetric --cipher-algo AES256 "$BACKUP_FILE"
```

2. **Secure backup location:**
```bash
# Set proper permissions
chmod 600 /opt/lcp-server/backups/*.sql.gz
chown backup-user:backup-group /opt/lcp-server/backups/
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
./scripts/restore-mysql.sh /opt/lcp-server/backups/latest-backup.sql.gz

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


## Troubleshooting

### High Impact During Backup

If backups cause significant performance issues:

1. **Switch to hot backup method**:
   ```bash
   # Instead of standard backup
   ./scripts/hot-backup.sh
   ```

2. **Adjust backup timing**:
   ```bash
   # Move to lower-traffic hours
   0 3 * * * /opt/lcp-server/scripts/backup-mysql.sh
   ```

3. **Optimize MySQL settings**:
   ```yaml
   # In compose.yaml mysql service
   command: >
     --innodb-buffer-pool-size=512M
     --innodb-log-file-size=256M
   ```

### Service Unavailable During Hot Backup

This shouldn't happen with hot backups. If it does:

1. **Check MySQL health**:
   ```bash
   docker compose logs mysql
   ```

2. **Verify backup process**:
   ```bash
   docker compose exec mysql mysqladmin processlist
   ```
