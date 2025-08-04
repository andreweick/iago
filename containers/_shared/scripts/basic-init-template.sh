#!/bin/bash
# Basic initialization script template
# Replace {SERVICE} with the actual service name when copying
set -euo pipefail

SERVICE_NAME="{SERVICE}"
SERVICE_USER="{SERVICE}-app"
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] [${SERVICE_NAME}-init]"

echo "$LOG_PREFIX Initializing ${SERVICE_NAME} container..."

# Create required directories
echo "$LOG_PREFIX Creating directories..."
mkdir -p "/var/lib/${SERVICE_NAME}" "/var/log/${SERVICE_NAME}"

# Set proper permissions
echo "$LOG_PREFIX Setting permissions..."
chown -R "${SERVICE_USER}:${SERVICE_USER}" "/var/lib/${SERVICE_NAME}" "/var/log/${SERVICE_NAME}"

# Additional service-specific initialization can be added here:
# - Configuration file setup
# - Database initialization
# - SSL certificate preparation
# - Third-party service connectivity tests

echo "$LOG_PREFIX ${SERVICE_NAME} container initialized successfully"