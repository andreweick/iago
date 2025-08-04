#!/bin/bash
set -euo pipefail

# Bootc Container Manager
# Automatically starts containers based on configuration files in /etc/iago/containers/

CONTAINER_CONFIG_DIR="/etc/iago/containers"
MACHINE_INFO_FILE="/etc/iago/machine-info"

echo "[$(date)] Starting bootc container manager"

# Check if container config directory exists
if [ ! -d "$CONTAINER_CONFIG_DIR" ]; then
    echo "[$(date)] Container config directory $CONTAINER_CONFIG_DIR does not exist"
    exit 0
fi

# Get machine name for default behavior
MACHINE_NAME=""
if [ -f "$MACHINE_INFO_FILE" ]; then
    MACHINE_NAME=$(grep "^MACHINE_NAME=" "$MACHINE_INFO_FILE" | cut -d= -f2)
    echo "[$(date)] Machine name: $MACHINE_NAME"
fi

# Function to start a container service
start_container_service() {
    local container_name="$1"
    local env_file="$2"
    
    echo "[$(date)] Found container config: $container_name"
    
    # Validate env file has required variables
    if ! grep -q "^CONTAINER_IMAGE=" "$env_file"; then
        echo "[$(date)] Warning: $env_file missing CONTAINER_IMAGE, skipping"
        return
    fi
    
    # Enable and start the service
    systemctl enable "bootc@${container_name}.service" || true
    
    # Check if service is already running
    if systemctl is-active --quiet "bootc@${container_name}.service"; then
        echo "[$(date)] Service bootc@${container_name}.service is already running"
    else
        echo "[$(date)] Starting bootc@${container_name}.service"
        systemctl start "bootc@${container_name}.service"
    fi
}

# Primary strategy: Look for machine-specific config first
if [ -n "$MACHINE_NAME" ] && [ -f "$CONTAINER_CONFIG_DIR/$MACHINE_NAME.env" ]; then
    start_container_service "$MACHINE_NAME" "$CONTAINER_CONFIG_DIR/$MACHINE_NAME.env"
else
    # Fallback strategy: Scan for any container config files
    echo "[$(date)] No machine-specific config found, scanning for all container configs"
    
    config_found=false
    for env_file in "$CONTAINER_CONFIG_DIR"/*.env; do
        # Check if glob matched any files
        [ -f "$env_file" ] || continue
        
        config_found=true
        container_name=$(basename "$env_file" .env)
        start_container_service "$container_name" "$env_file"
    done
    
    if [ "$config_found" = false ]; then
        echo "[$(date)] No container configuration files found in $CONTAINER_CONFIG_DIR"
    fi
fi

echo "[$(date)] Bootc container manager completed"