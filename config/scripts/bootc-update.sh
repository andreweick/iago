#!/bin/bash
set -euo pipefail

# Bootc Container Update Script
# Updates all configured containers based on their update strategy

CONTAINER_CONFIG_DIR="/etc/iago/containers"
MACHINE_INFO_FILE="/etc/iago/machine-info"
HEALTH_CHECK_WAIT="{{ .Bootc.HealthCheckWait }}"

echo "[$(date)] Starting bootc container update process"

# Check if container config directory exists
if [ ! -d "$CONTAINER_CONFIG_DIR" ]; then
    echo "[$(date)] Container config directory $CONTAINER_CONFIG_DIR does not exist"
    exit 0
fi

# Function to update a single container
update_container() {
    local container_name="$1"
    local env_file="$2"
    
    echo "[$(date)] Updating container: $container_name"
    
    # Source the environment file
    source "$env_file"
    
    # Validate required variables
    if [ -z "${CONTAINER_IMAGE:-}" ]; then
        echo "[$(date)] Warning: CONTAINER_IMAGE not defined in $env_file, skipping"
        return
    fi
    
    # Set defaults
    UPDATE_STRATEGY="${UPDATE_STRATEGY:-latest}"
    HEALTH_CHECK_WAIT_OVERRIDE="${HEALTH_CHECK_WAIT_OVERRIDE:-$HEALTH_CHECK_WAIT}"
    SERVICE="bootc@${container_name}.service"
    
    echo "[$(date)] Checking for updates to ${CONTAINER_IMAGE} (strategy: $UPDATE_STRATEGY)"
    
    # Skip update if strategy is 'pinned'
    if [ "$UPDATE_STRATEGY" = "pinned" ]; then
        echo "[$(date)] Container $container_name is pinned, skipping update"
        return
    fi
    
    # Save current image as :previous for rollback
    podman tag "${CONTAINER_IMAGE}" "${CONTAINER_IMAGE%:*}:previous" 2>/dev/null || true
    
    # Pull latest image
    if ! podman pull "${CONTAINER_IMAGE}"; then
        echo "[$(date)] Failed to pull ${CONTAINER_IMAGE}"
        return
    fi
    
    # Check if service is running and restart if needed
    if systemctl is-active --quiet "${SERVICE}"; then
        echo "[$(date)] Restarting ${SERVICE} with new image"
        systemctl restart "${SERVICE}"
        
        # Wait for service to stabilize
        sleep "$HEALTH_CHECK_WAIT_OVERRIDE"
        
        # Verify service is healthy
        if ! systemctl is-active --quiet "${SERVICE}"; then
            echo "[$(date)] Service $SERVICE failed to start with new image, rolling back"
            podman tag "${CONTAINER_IMAGE%:*}:previous" "${CONTAINER_IMAGE}"
            systemctl restart "${SERVICE}"
            echo "[$(date)] Rollback completed for $container_name"
        else
            echo "[$(date)] Successfully updated $container_name"
        fi
    else
        echo "[$(date)] Service $SERVICE is not active, skipping update"
    fi
}

# Get machine name for primary container
MACHINE_NAME=""
if [ -f "$MACHINE_INFO_FILE" ]; then
    MACHINE_NAME=$(grep "^MACHINE_NAME=" "$MACHINE_INFO_FILE" | cut -d= -f2)
fi

# Update machine-specific container first if it exists
if [ -n "$MACHINE_NAME" ] && [ -f "$CONTAINER_CONFIG_DIR/$MACHINE_NAME.env" ]; then
    update_container "$MACHINE_NAME" "$CONTAINER_CONFIG_DIR/$MACHINE_NAME.env"
fi

# Update any additional containers
for env_file in "$CONTAINER_CONFIG_DIR"/*.env; do
    # Check if glob matched any files
    [ -f "$env_file" ] || continue
    
    container_name=$(basename "$env_file" .env)
    
    # Skip if this is the machine-specific container we already updated
    if [ "$container_name" = "$MACHINE_NAME" ]; then
        continue
    fi
    
    update_container "$container_name" "$env_file"
done

echo "[$(date)] Bootc container update process completed"