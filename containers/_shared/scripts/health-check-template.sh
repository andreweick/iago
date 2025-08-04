#!/bin/bash
# Generic health check script template
# Replace {SERVICE} with the actual service name when copying
set -euo pipefail

SERVICE_NAME="{SERVICE}"
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] [${SERVICE_NAME}-health]"

# Simple health check - verify service is running
if systemctl is-active --quiet "bootc@${SERVICE_NAME}.service"; then
    echo "$LOG_PREFIX ${SERVICE_NAME} service is healthy"
    exit 0
else
    echo "$LOG_PREFIX bootc@${SERVICE_NAME}.service is not running"
    exit 1
fi

# Additional health checks can be added here based on service needs:
# - Port connectivity tests
# - Application-specific endpoints
# - File system checks
# - Resource utilization checks