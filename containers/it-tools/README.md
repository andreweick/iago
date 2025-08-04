# IT Tools Container

A bootc container for [it-tools](https://github.com/CorentinTh/it-tools), a collection of useful web-based developer tools.

## Version Information

- **Base Image**: fedora-bootc:42
- **Node.js**: 20.x
- **pnpm**: 8.15.6
- **serve**: 14.2.1
- **it-tools**: main branch (latest)

## Service Information

This container uses the `bootc@it-tools.service` template for service management. The application is served using Node.js `serve` package for production-ready static file serving.

## Port Information

**Exposed Port**: 80 (HTTP)

The container exposes port 80 for reverse proxy configuration. The external reverse proxy handles TLS termination and routing.

## Container Auto-Updates

This container participates in the automated bootc update system:
- **Daily Updates**: Pulls new container images at 02:00 AM via `bootc-update.timer`
- **Health Monitoring**: Automatic rollback on health check failures
- **Zero Downtime**: Seamless updates without service interruption

## Service Management

Use standard systemd commands with the bootc service template:

```bash
# Check service status
systemctl status bootc@it-tools.service

# Start/stop/restart service
systemctl start bootc@it-tools.service
systemctl stop bootc@it-tools.service
systemctl restart bootc@it-tools.service

# View logs
journalctl -xeu bootc@it-tools.service

# Check container logs directly
podman logs bootc-it-tools
```

## Reverse Proxy Integration

This container is designed to work with an external reverse proxy:
- **TLS/SSL**: Handled by reverse proxy
- **Routing**: Reverse proxy forwards requests to port 80
- **Headers**: Container serves plain HTTP, proxy adds security headers

Example reverse proxy configuration (port 80 backend):
```
upstream it-tools {
    server machine.example.com:80;
}
```

## Secrets Management

This container includes 1Password CLI for future secret management:
- **Current Status**: No secrets are configured
- **1Password Ready**: Complete infrastructure is installed and ready
- **Auto-Activation**: Will automatically handle secrets when configured

To add secrets in the future:
1. SSH to the deployed machine
2. Replace `/etc/iago/secrets/1password-token.txt` with real service account token
3. Customize `/usr/local/bin/fetch-it-tools-secrets.sh` with your specific secret references
4. Secrets will be automatically fetched when token file is updated

## Health Checks

The container includes comprehensive health monitoring:
- **Service Status**: Verifies `bootc@it-tools.service` is running
- **Port Check**: Confirms port 80 is responding to HTTP requests
- **Automatic Recovery**: Health failures trigger service restart and potential rollback

Health check script: `/usr/local/bin/health-it-tools.sh`

## Update Strategies

### Version Updates
To update it-tools or dependencies:
1. Edit `Containerfile` with new version numbers
2. Rebuild container: `just build-container it-tools`
3. Push to registry: Container will auto-update at next cycle

### Disable Auto-Updates
Create `/etc/iago/containers/it-tools.env`:
```bash
BOOTC_UPDATE_DISABLE=true
```

### Manual Updates
```bash
# Force immediate update
systemctl start bootc-update.service

# Check update logs
journalctl -xeu bootc-update.service
```

## Troubleshooting

### Service Won't Start
```bash
# Check systemd service
systemctl status bootc@it-tools.service

# Check container status
podman ps -a | grep it-tools

# View detailed logs
journalctl -xeu bootc@it-tools.service
```

### Port Not Responding
```bash
# Check if port 80 is bound
ss -tlnp | grep :80

# Test local connectivity
curl -v http://localhost:80

# Check application logs
podman logs bootc-it-tools
```

### Update Failures
```bash
# Check bootc update logs
journalctl -xeu bootc-update.service

# Verify container registry connectivity
podman pull registry.example.com/it-tools:latest

# Force rollback if needed
systemctl stop bootc@it-tools.service
podman rm -f bootc-it-tools
systemctl start bootc@it-tools.service
```

### Health Check Failures
```bash
# Run health check manually
/usr/local/bin/health-it-tools.sh

# Check service dependencies
systemctl list-dependencies bootc@it-tools.service

# Verify network connectivity
curl -I http://localhost:80
```

## Development and Testing

### Local Testing
```bash
# Build container locally
podman build -t localhost:5000/it-tools:test .

# Test with no secrets (default)
podman run --rm --privileged --network host localhost:5000/it-tools:test

# Test with mock secrets for development
LOCAL_TESTING=true podman run --rm --privileged --network host localhost:5000/it-tools:test
```

### Build Process
The container build process:
1. Installs Node.js, pnpm, and serve with pinned versions
2. Clones it-tools repository
3. Runs `pnpm install --frozen-lockfile`
4. Builds with `pnpm build`
5. Installs to `/usr/share/it-tools`
6. Sets up service user and permissions
7. Configures systemd service and health checks

### Security Features
- **Non-root execution**: Runs as `it-tools` system user
- **Minimal privileges**: `NoNewPrivileges=true`
- **Filesystem protection**: Read-only system with specific write paths
- **Network isolation**: Only necessary ports exposed
- **Secret security**: 0600 permissions on all secret files

## Architecture Notes

### Bootc Container Design
- **Immutable Base**: Application and dependencies in `/usr`
- **Mutable State**: Logs and runtime data in `/var`
- **Service Integration**: Native systemd service management
- **Update Model**: Entire container updated atomically

### Static File Serving
- **serve Package**: Production-ready Node.js static server
- **SPA Support**: Configured with `-s` flag for client-side routing
- **Performance**: Optimized caching and compression
- **Simplicity**: No nginx complexity, direct Node.js serving

This container provides a robust, auto-updating deployment of it-tools with comprehensive monitoring, security, and operational features.