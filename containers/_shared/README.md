# Shared Container Resources

This directory contains reusable templates and common patterns for container development across all iago workloads.

## Directory Structure

```
_shared/
├── scripts/           # Script templates
└── systemd/           # Systemd unit templates
```

## Scripts Templates

### `scripts/health-check-template.sh`

Generic health check script for services.

**Usage:**
1. Copy to your container's `scripts/` directory as `health.sh`
2. Replace `{SERVICE}` with your service name (e.g., `postgres`, `caddy-work`)
3. Add service-specific health checks as needed

**Example:**
```bash
# For postgres container
cp _shared/scripts/health-check-template.sh postgres/scripts/health.sh
sed -i 's/{SERVICE}/postgres/g' postgres/scripts/health.sh
```

**Provides:**
- Service status checking via systemd
- Consistent logging format
- Error handling
- Extension points for service-specific checks

### `scripts/basic-init-template.sh`

Simple initialization script for basic services.

**Usage:**
1. Copy to your container's `scripts/` directory as `init.sh`
2. Replace `{SERVICE}` with your service name
3. Add service-specific initialization as needed

**Example:**
```bash
# For a new web service
cp _shared/scripts/basic-init-template.sh web/scripts/init.sh
sed -i 's/{SERVICE}/web/g' web/scripts/init.sh
```

**Provides:**
- Directory creation (`/var/lib/{service}`, `/var/log/{service}`)
- Permission setting
- Consistent logging
- Extension points for complex initialization

**Note:** For complex services like PostgreSQL or Immich, create custom init scripts instead of using this template.

### `scripts/fetch-secrets-template.sh`

Generic 1Password secret fetching script for services requiring external secrets.

**Usage:**
1. Copy to your container's `scripts/` directory as `fetch-{SERVICE}-secrets.sh`
2. Replace `{SERVICE}` with your service name
3. Customize the secret fetching logic for your specific 1Password items
4. Remove the template error block after customization

**Example:**
```bash
# For caddy container needing Cloudflare token
cp _shared/scripts/fetch-secrets-template.sh caddy-work/scripts/fetch-caddy-secrets.sh
sed -i 's/{SERVICE}/caddy/g' caddy-work/scripts/fetch-caddy-secrets.sh
# Then edit the script to fetch actual Cloudflare token from 1Password
```

**Provides:**
- 1Password service account token validation
- Generic secret fetching framework using `op` CLI
- Proper file permissions and ownership
- Consistent error handling and logging
- Examples for common secret types (API tokens, passwords, environment files)

**Prerequisites:**
- 1Password CLI (`op`) installed in container
- 1Password service account token manually placed in `/etc/iago/secrets/1password-token.txt`
- Corresponding systemd service and watcher units (use templates)

### `scripts/backup-template.sh`

Automated backup script for nightly rsync to NAS.

**Usage:**
1. Copy to your container's `scripts/` directory as `backup-{SERVICE}.sh`
2. Replace `{SERVICE}` with your service name
3. Configure backup paths in `/etc/iago/backup/{SERVICE}.conf`
4. Add SSH key via 1Password integration

**Example:**
```bash
# For postgres container
cp _shared/scripts/backup-template.sh postgres/scripts/backup-postgres.sh
sed -i 's/{SERVICE}/postgres/g' postgres/scripts/backup-postgres.sh
```

**Provides:**
- Rsync-based efficient backups to NAS
- Retention management (default 7 days)
- Pre-backup hooks for application consistency
- Bandwidth limiting to prevent network saturation
- Automatic SSH key detection (exits gracefully if not configured)

**Configuration:**
- Global settings: `/etc/iago/backup/config.env`
- Per-service overrides: `/etc/iago/backup/{SERVICE}.conf`

**Pre-backup hooks:**
Create `/usr/local/bin/pre-backup-{SERVICE}.sh` for database dumps or other consistency measures.

## Systemd Templates

### `systemd/secrets-service.template`

Generic secrets fetching service for 1Password integration.

**Usage:**
1. Copy to your container's `systemd/` directory 
2. Rename to `{SERVICE}-secrets.service`
3. Replace `{SERVICE}` placeholders with your service name
4. Adjust `ReadWritePaths` for your service's secret locations
5. Create corresponding `fetch-{SERVICE}-secrets.sh` script

**Example:**
```bash
# For caddy container
cp _shared/systemd/secrets-service.template caddy-work/systemd/caddy-secrets.service
sed -i 's/{SERVICE}/caddy/g' caddy-work/systemd/caddy-secrets.service
```

**Provides:**
- 1Password secret fetching integration
- Proper dependency ordering (runs before main service)
- Security settings and isolation
- Failure handling and retry logic

### `systemd/secrets-watcher.template`

Path unit for watching secret file changes.

**Usage:**
1. Copy to your container's `systemd/` directory
2. Rename to `{SERVICE}-secrets-watcher.path`
3. Replace `{SERVICE}` placeholders
4. Adjust `PathModified` path if needed

**Example:**
```bash
# For postgres container
cp _shared/systemd/secrets-watcher.template postgres/systemd/postgres-secrets-watcher.path
sed -i 's/{SERVICE}/postgres/g' postgres/systemd/postgres-secrets-watcher.path
```

**Provides:**
- Automatic secret reloading when files change
- Integration with systemd path units
- Triggers secrets service on file modifications

### `systemd/backup-service.template`

Systemd service unit for running backups.

**Usage:**
1. Copy to your container's `systemd/` directory
2. Rename to `{SERVICE}-backup.service`
3. Replace `{SERVICE}` placeholders with your service name
4. Adjust `ReadWritePaths` if your service backs up additional paths

**Example:**
```bash
# For postgres container
cp _shared/systemd/backup-service.template postgres/systemd/postgres-backup.service
sed -i 's/{SERVICE}/postgres/g' postgres/systemd/postgres-backup.service
```

**Provides:**
- One-shot service for backup execution
- Security isolation and resource limits
- Proper dependency ordering (after network and secrets)
- Journal integration for logging

### `systemd/backup-timer.template`

Systemd timer unit for scheduled backups.

**Usage:**
1. Copy to your container's `systemd/` directory
2. Rename to `{SERVICE}-backup.timer`
3. Replace `{SERVICE}` placeholders
4. Timer runs daily with 1-hour random delay

**Example:**
```bash
# For photos container
cp _shared/systemd/backup-timer.template photos/systemd/photos-backup.timer
sed -i 's/{SERVICE}/photos/g' photos/systemd/photos-backup.timer
```

**Provides:**
- Daily backup schedule (midnight with random delay)
- Persistent timers (catches up missed runs)
- Automatic enable during container build

## When to Use Templates vs Custom Files

### Use Templates For:
- **Simple services** with standard patterns
- **New containers** that follow common patterns  
- **Basic health checks** that only verify service status
- **Standard secret fetching** from 1Password
- **Simple initialization** (directories, permissions)
- **Standard backup patterns** (rsync to NAS with retention)

### Create Custom Files For:
- **Complex initialization** (PostgreSQL database setup, Immich config generation)
- **Advanced health checks** (multi-service, endpoint testing, resource monitoring)
- **Specialized secret handling** (multiple secret sources, complex processing)
- **Service-specific systemd units** (main application services)

## Best Practices

1. **Copy, Don't Symlink:** Always copy templates to avoid accidental modifications
2. **Customize After Copy:** Adapt templates to your service's specific needs
3. **Use Consistent Naming:** Follow the `{SERVICE}-{purpose}` naming pattern
4. **Document Changes:** Add comments explaining service-specific modifications
5. **Test Thoroughly:** Verify templates work correctly after customization

## Template Variables

All templates use `{SERVICE}` as the primary placeholder:
- Replace with your actual service name (e.g., `postgres`, `caddy-work`, `photos`)
- Use consistent naming across all files for a service
- Follow existing naming conventions in the project

## Examples

### Creating a New Simple Service

```bash
# 1. Copy templates
cp _shared/scripts/health-check-template.sh myservice/scripts/health.sh
cp _shared/scripts/basic-init-template.sh myservice/scripts/init.sh
cp _shared/scripts/backup-template.sh myservice/scripts/backup-myservice.sh
cp _shared/systemd/secrets-service.template myservice/systemd/myservice-secrets.service
cp _shared/systemd/backup-service.template myservice/systemd/myservice-backup.service
cp _shared/systemd/backup-timer.template myservice/systemd/myservice-backup.timer

# 2. Replace placeholders
sed -i 's/{SERVICE}/myservice/g' myservice/scripts/health.sh
sed -i 's/{SERVICE}/myservice/g' myservice/scripts/init.sh
sed -i 's/{SERVICE}/myservice/g' myservice/scripts/backup-myservice.sh
sed -i 's/{SERVICE}/myservice/g' myservice/systemd/myservice-secrets.service
sed -i 's/{SERVICE}/myservice/g' myservice/systemd/myservice-backup.service
sed -i 's/{SERVICE}/myservice/g' myservice/systemd/myservice-backup.timer

# 3. If using 1Password secrets, copy and customize fetch script
cp _shared/scripts/fetch-secrets-template.sh myservice/scripts/fetch-myservice-secrets.sh
sed -i 's/{SERVICE}/myservice/g' myservice/scripts/fetch-myservice-secrets.sh
# Then edit the script to add your specific 1Password secret references

# 4. Customize as needed for your service
```

### Updating Templates

When updating templates:
1. Update the template file in `_shared/`
2. Consider if existing containers need updates
3. Document breaking changes
4. Test with a sample container

## Contributing

When adding new templates:
1. Ensure they solve a common pattern across multiple containers
2. Use clear placeholder variables
3. Include usage instructions in comments
4. Add documentation to this README
5. Test with at least two different services