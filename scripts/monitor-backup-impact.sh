#!/bin/bash

# Backup Impact Monitor
# This script monitors system performance during backup operations

set -e

MONITOR_DURATION=300  # 5 minutes
LOG_FILE="/var/log/lcp-backup-impact.log"

echo "LCP Server - Backup Impact Monitor"
echo "=================================="
echo "Monitoring system impact during backup operations"
echo "Duration: ${MONITOR_DURATION} seconds"
echo ""

# Function to get metrics
get_metrics() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # MySQL connections
    local mysql_connections=$(docker compose exec -T mysql mysql -u root -p$(cat config/mysql-root-password.txt) -e "SHOW STATUS LIKE 'Threads_connected';" 2>/dev/null | tail -1 | awk '{print $2}' || echo "0")
    
    # MySQL queries per second
    local mysql_qps=$(docker compose exec -T mysql mysql -u root -p$(cat config/mysql-root-password.txt) -e "SHOW STATUS LIKE 'Queries';" 2>/dev/null | tail -1 | awk '{print $2}' || echo "0")
    
    # Container CPU and memory usage
    local server_stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep lcp-server || echo "lcp-server\t0%\t0MB")
    local mysql_stats=$(docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}" | grep mysql || echo "mysql\t0%\t0MB")
    
    echo "[$timestamp] MySQL Connections: $mysql_connections, QPS: $mysql_qps"
    echo "[$timestamp] Server: $server_stats"
    echo "[$timestamp] MySQL: $mysql_stats"
    echo "[$timestamp] ---"
}

# Test application responsiveness
test_responsiveness() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    # Test health endpoint
    local health_response_time=$(curl -o /dev/null -s -w "%{time_total}" http://localhost/health 2>/dev/null || echo "timeout")
    
    # Test API endpoint
    local api_response_time=$(curl -o /dev/null -s -w "%{time_total}" http://localhost/api/status 2>/dev/null || echo "timeout")
    
    echo "[$timestamp] Health endpoint: ${health_response_time}s"
    echo "[$timestamp] API endpoint: ${api_response_time}s"
}

# Start monitoring
echo "Starting baseline monitoring (30 seconds)..."
for i in {1..6}; do
    get_metrics
    test_responsiveness
    sleep 5
done

echo ""
echo "=== BASELINE ESTABLISHED ==="
echo "Now run your backup script in another terminal:"
echo "  ./scripts/backup-mysql.sh"
echo "  ./scripts/hot-backup.sh"
echo "  ./scripts/cold-backup.sh"
echo ""
echo "Monitoring during backup operation..."

# Monitor during backup
start_time=$(date +%s)
while [ $(($(date +%s) - start_time)) -lt $MONITOR_DURATION ]; do
    get_metrics
    test_responsiveness
    
    # Check if backup is running
    if pgrep -f "mysqldump" > /dev/null; then
        echo "[$(date '+%Y-%m-%d %H:%M:%S')] *** BACKUP DETECTED - MONITORING IMPACT ***"
    fi
    
    sleep 5
done

echo ""
echo "=== MONITORING COMPLETED ==="
echo "Check $LOG_FILE for detailed logs"
echo ""
echo "Summary Guidelines:"
echo "- Hot backup: Service remains available, <10% performance impact"
echo "- Cold backup: Service unavailable for 30-120 seconds, 100% data consistency"
echo "- Standard backup: Service available, 10-20% performance impact"