#!/bin/bash
# Automated backup script for {SERVICE}
# Performs nightly rsync backups to NAS with retention management
set -euo pipefail

SERVICE_NAME="{SERVICE}"
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] [${SERVICE_NAME}-backup]"
BACKUP_CONFIG="/etc/iago/backup/config.env"
SERVICE_BACKUP_CONFIG="/etc/iago/backup/${SERVICE_NAME}.conf"

# Source configurations
[[ -f "$BACKUP_CONFIG" ]] && source "$BACKUP_CONFIG"
[[ -f "$SERVICE_BACKUP_CONFIG" ]] && source "$SERVICE_BACKUP_CONFIG"

# Default values
NAS_HOST="${NAS_HOST:-nas.local}"
BACKUP_BASE="${BACKUP_BASE:-/volume1/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
BACKUP_PATHS="${BACKUP_PATHS:-/var/lib/${SERVICE_NAME}}"
SSH_KEY="${SSH_KEY:-/etc/iago/secrets/backup-ssh-key}"
BANDWIDTH_LIMIT="${BANDWIDTH_LIMIT:-5000}" # KB/s

echo "$LOG_PREFIX Starting backup for ${SERVICE_NAME}"

# Verify SSH key exists
if [[ ! -f "$SSH_KEY" ]]; then
    echo "$LOG_PREFIX ERROR: SSH key not found at $SSH_KEY"
    echo "$LOG_PREFIX Backup not configured - exiting successfully"
    exit 0
fi

# Run pre-backup hook if exists
if [[ -x "/usr/local/bin/pre-backup-${SERVICE_NAME}.sh" ]]; then
    echo "$LOG_PREFIX Running pre-backup hook"
    /usr/local/bin/pre-backup-${SERVICE_NAME}.sh || {
        echo "$LOG_PREFIX WARNING: Pre-backup hook failed"
    }
fi

# Perform backup
BACKUP_DEST="${BACKUP_BASE}/$(hostname)/${SERVICE_NAME}/$(date +%Y%m%d_%H%M%S)"
echo "$LOG_PREFIX Backing up to ${NAS_HOST}:${BACKUP_DEST}"

for path in $BACKUP_PATHS; do
    if [[ -d "$path" ]]; then
        echo "$LOG_PREFIX Backing up $path"
        rsync -avz --delete \
            --bwlimit="${BANDWIDTH_LIMIT}" \
            -e "ssh -i $SSH_KEY -o StrictHostKeyChecking=no" \
            "$path/" \
            "${NAS_HOST}:${BACKUP_DEST}${path}/" || {
            echo "$LOG_PREFIX ERROR: Backup failed for $path"
            exit 1
        }
    fi
done

# Clean old backups
echo "$LOG_PREFIX Cleaning backups older than ${RETENTION_DAYS} days"
ssh -i "$SSH_KEY" -o StrictHostKeyChecking=no "$NAS_HOST" \
    "find ${BACKUP_BASE}/$(hostname)/${SERVICE_NAME} -type d -mtime +${RETENTION_DAYS} -exec rm -rf {} +" || {
    echo "$LOG_PREFIX WARNING: Cleanup failed"
}

echo "$LOG_PREFIX Backup completed successfully"