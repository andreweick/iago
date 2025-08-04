#!/bin/bash
set -euo pipefail

# Bootc Container Runner
# Generic script to run containers based on environment configuration

CONTAINER_NAME="$1"
CONTAINER_CONFIG_DIR="/etc/iago/containers"
ENV_FILE="$CONTAINER_CONFIG_DIR/$CONTAINER_NAME.env"

# Check if container config exists
if [ ! -f "$ENV_FILE" ]; then
    echo "Error: Container configuration file $ENV_FILE does not exist"
    exit 1
fi

# Source the environment file
source "$ENV_FILE"

# Validate required environment variables
if [ -z "${CONTAINER_IMAGE:-}" ]; then
    echo "Error: CONTAINER_IMAGE not defined in $ENV_FILE"
    exit 1
fi

# Set defaults for optional variables
CONTAINER_NAME_ACTUAL="${CONTAINER_NAME:-bootc-$CONTAINER_NAME}"
HEALTH_CHECK_WAIT="${HEALTH_CHECK_WAIT:-30}"
RESTART_POLICY="${RESTART_POLICY:-always}"

echo "[$(date)] Starting container: $CONTAINER_IMAGE"
echo "[$(date)] Container name: $CONTAINER_NAME_ACTUAL"

# Pull the latest image first
echo "[$(date)] Pulling container image..."
/usr/bin/podman pull "$CONTAINER_IMAGE"

# Run the container with standard bootc configuration
exec /usr/bin/podman run \
    --rm \
    --name "$CONTAINER_NAME_ACTUAL" \
    --net host \
    --pid host \
    --privileged \
    --security-opt label=disable \
    --volume /etc:/etc \
    --volume /var:/var \
    --volume /run:/run \
    --env "MACHINE_NAME=$CONTAINER_NAME" \
    --sdnotify=conmon \
    --health-cmd "/usr/local/bin/health.sh" \
    --health-interval=30s \
    --health-retries=3 \
    --health-start-period=60s \
    "$CONTAINER_IMAGE"