#!/bin/bash

# Automated MySQL Backup with Cron
# This script performs automated backups and can be added to crontab

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BACKUP_SCRIPT="$SCRIPT_DIR/hot-backup.sh"
LOG_FILE="$PROJECT_DIR/logs/hot-backup.log"

# Change to project directory
cd "$PROJECT_DIR"

# Create log directory if it doesn't exist
mkdir -p "$(dirname "$LOG_FILE")"

# Function to log with timestamp
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

log "Starting automated MySQL backup"

# Check if backup script exists
if [ ! -f "$BACKUP_SCRIPT" ]; then
    log "ERROR: Backup script not found: $BACKUP_SCRIPT"
    exit 1
fi

# Make sure backup script is executable
chmod +x "$BACKUP_SCRIPT"

# Run backup
if "$BACKUP_SCRIPT" >> "$LOG_FILE" 2>&1; then
    log "Automated backup completed successfully"
else
    log "ERROR: Automated backup failed"
    exit 1
fi

# Optional: Send backup to remote storage (S3, rsync, etc.)
# Uncomment and configure as needed:

# # Upload to S3
# BACKUP_DIR="/opt/lcp-backups"
# LATEST_BACKUP=$(ls -t "$BACKUP_DIR"/hot_backup_*.sql.gz | head -1)
# if [ -n "$LATEST_BACKUP" ]; then
#     log "Uploading backup to S3..."
#     aws s3 cp "$LATEST_BACKUP" s3://your-backup-bucket/lcp-backups/
# fi

# # Sync to remote server
# log "Syncing backups to remote server..."
# rsync -avz "$BACKUP_DIR"/ user@backup-server:/backups/lcp/

log "Automated backup process completed"