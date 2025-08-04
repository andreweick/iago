#!/bin/bash
# Generic health check script template
# Replace it-tools with the actual service name when copying
set -euo pipefail

SERVICE_NAME="it-tools"
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] [${SERVICE_NAME}-health]"

# Health check for it-tools - verify service and port
if systemctl is-active --quiet "bootc@${SERVICE_NAME}.service"; then
    echo "$LOG_PREFIX systemd service is running"
    
    # Check if port 80 is responding
    if curl -f -s http://localhost:80 > /dev/null 2>&1; then
        echo "$LOG_PREFIX port 80 is responding"
        echo "$LOG_PREFIX ${SERVICE_NAME} service is healthy"
        exit 0
    else
        echo "$LOG_PREFIX port 80 is not responding"
        exit 1
    fi
else
    echo "$LOG_PREFIX bootc@${SERVICE_NAME}.service is not running"
    exit 1
fi