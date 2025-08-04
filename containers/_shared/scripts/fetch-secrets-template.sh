#!/bin/bash
# Smart auto-activating secrets script for {SERVICE}
# This script is always present and handles all secret scenarios:
# - No secrets needed: exits successfully  
# - Local testing: creates mock secrets
# - Real 1Password: fetches actual secrets (when customized)
set -euo pipefail

SERVICE_NAME="{SERVICE}"
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] [${SERVICE_NAME}-secrets]"
ONEPASSWORD_TOKEN_FILE="/etc/iago/secrets/1password-token.txt"
SECRETS_DIR="/etc/${SERVICE_NAME}/secrets"

echo "$LOG_PREFIX Starting smart secret management for ${SERVICE_NAME}..."

# ================================
# LOCAL TESTING MODE
# ================================
if [[ "${LOCAL_TESTING:-false}" == "true" ]] || [[ "${ENVIRONMENT:-}" == "development" ]]; then
    echo "$LOG_PREFIX Local testing mode detected - creating mock secrets"
    mkdir -p "$SECRETS_DIR"
    
    # Create common mock secrets (customize as needed for your service)
    echo "mock-api-token-for-${SERVICE_NAME}" > "$SECRETS_DIR/api-token"
    echo "mock-password-for-${SERVICE_NAME}" > "$SECRETS_DIR/password"
    echo "mock-database-url-for-${SERVICE_NAME}" > "$SECRETS_DIR/database-url"
    
    # Create mock environment file
    cat > "$SECRETS_DIR/environment" << EOF
# Mock environment variables for local testing
API_TOKEN=mock-api-token-for-${SERVICE_NAME}
DB_PASSWORD=mock-password-for-${SERVICE_NAME}
DATABASE_URL=mock-database-url-for-${SERVICE_NAME}
EOF
    
    chmod -R 0600 "$SECRETS_DIR"/*
    echo "$LOG_PREFIX Mock secrets created for local testing"
    exit 0
fi

# ================================
# TOKEN VALIDATION
# ================================
# Check if token file exists
if [[ ! -f "$ONEPASSWORD_TOKEN_FILE" ]]; then
    echo "$LOG_PREFIX No 1Password token file found - no secrets configured"
    exit 0
fi

# Check if file contains actual token (not just comments/placeholder)
if ! grep -v '^#' "$ONEPASSWORD_TOKEN_FILE" | grep -q .; then
    echo "$LOG_PREFIX No secrets configured (token file contains only comments)"
    exit 0
fi

# Read potential token
POTENTIAL_TOKEN=$(grep -v '^#' "$ONEPASSWORD_TOKEN_FILE" | head -n1 | tr -d '\n\r')

# Check for mock token
if [[ "$POTENTIAL_TOKEN" == "MOCK_TOKEN_FOR_LOCAL_TESTING" ]]; then
    echo "$LOG_PREFIX Mock token detected - creating mock secrets"
    mkdir -p "$SECRETS_DIR"
    
    # Create mock secrets (same as local testing mode)
    echo "mock-api-token-for-${SERVICE_NAME}" > "$SECRETS_DIR/api-token"
    echo "mock-password-for-${SERVICE_NAME}" > "$SECRETS_DIR/password"
    
    cat > "$SECRETS_DIR/environment" << EOF
API_TOKEN=mock-api-token-for-${SERVICE_NAME}
DB_PASSWORD=mock-password-for-${SERVICE_NAME}
EOF
    
    chmod -R 0600 "$SECRETS_DIR"/*
    echo "$LOG_PREFIX Mock secrets created"
    exit 0
fi

# ================================
# REAL 1PASSWORD TOKEN DETECTED
# ================================
echo "$LOG_PREFIX Real 1Password token detected"

# Check if this script has been customized with actual 1Password references
if ! grep -q "op://" "$0"; then
    echo "$LOG_PREFIX Real token present but no 1Password references configured in this script"
    echo "$LOG_PREFIX To activate real secrets:"
    echo "$LOG_PREFIX   1. Edit $0"
    echo "$LOG_PREFIX   2. Add your op:// references in the CUSTOMIZE section below"
    echo "$LOG_PREFIX   3. Remove the template check"
    exit 0
fi

echo "$LOG_PREFIX Fetching real secrets from 1Password..."

# Set the service account token as environment variable for op CLI
export OP_SERVICE_ACCOUNT_TOKEN="$POTENTIAL_TOKEN"

# Create secrets directory
mkdir -p "$SECRETS_DIR"

# ================================
# CUSTOMIZE THIS SECTION FOR YOUR SERVICE
# ================================
# Add your actual 1Password references here
# Examples:

# Example 1: Fetch API token
# if ! API_TOKEN=$(op read "op://iago/your-service-item/api-token" 2>/dev/null); then
#     echo "$LOG_PREFIX ERROR: Failed to fetch API token from 1Password"
#     exit 1
# fi
# echo "$API_TOKEN" > "$SECRETS_DIR/api-token"
# chmod 0600 "$SECRETS_DIR/api-token"

# Example 2: Fetch database password  
# if ! DB_PASSWORD=$(op read "op://iago/your-service-item/database-password" 2>/dev/null); then
#     echo "$LOG_PREFIX ERROR: Failed to fetch database password from 1Password"
#     exit 1
# fi
# echo "$DB_PASSWORD" > "$SECRETS_DIR/db-password"
# chmod 0600 "$SECRETS_DIR/db-password"

# Example 3: Create environment file for systemd
# cat > "$SECRETS_DIR/environment" << EOF
# API_TOKEN=$API_TOKEN
# DB_PASSWORD=$DB_PASSWORD
# EOF
# chmod 0600 "$SECRETS_DIR/environment"

# ================================
# TEMPLATE CHECK (REMOVE AFTER CUSTOMIZING)
# ================================
echo "$LOG_PREFIX This script template needs customization!"
echo "$LOG_PREFIX Add your actual op:// references above and remove this check"
exit 0

# ================================
# END CUSTOMIZATION SECTION
# ================================

echo "$LOG_PREFIX Successfully fetched and saved all secrets for ${SERVICE_NAME}"
echo "$LOG_PREFIX Secrets saved to $SECRETS_DIR/"

# Auto-Activation Usage:
# This script is automatically included in every iago container
# It handles all scenarios without manual intervention:
#
# No secrets needed:
#   - Just works, no action required
#
# Local testing:  
#   - LOCAL_TESTING=true podman run ...
#   - ENVIRONMENT=development podman run ...
#   - Creates mock secrets automatically
#
# Real secrets:
#   - Customize this script with your op:// references  
#   - Add real token to /etc/iago/secrets/1password-token.txt
#   - Secrets automatically fetched