# Bootc Container Creation Prompt for Iago

Use this prompt when creating a new bootc container in the iago project. Fill in the required information below before proceeding:

---

## Application Information (REQUIRED)

**Application Name**: [FILL_IN_NAME]
**GitHub Repository**: [FILL_IN_REPO_URL]
**Purpose/Description**: [FILL_IN_DESCRIPTION]

**Reference Dockerfile** (if available: [FILL_IN_URL]


---

I need to create a bootc container for **[APPLICATION_NAME]** in my iago project.

Once you have read and processed the prompt, present the plan to the user for feedback.

This project uses a specific bootc auto-update system with **STRICT PATTERNS** that must be followed.

## CRITICAL: Pattern Compliance Requirements

**MANDATORY PATTERN ENFORCEMENT:**
1. **MUST use `bootc@{name}.service` template** - NO custom systemd services allowed
2. **MUST copy from `containers/_shared/` templates** - NO creating scripts from scratch
3. **MUST follow shared template patterns** - NO deviations without explicit planning session
4. **MUST copy from `containers/_shared/` scripts** - NO creating scripts from scratch
3. **MUST follow shared scripts patterns** - NO deviations without explicit planning session


**IF TEMPLATES CANNOT BE FOLLOWED:**
- STOP immediately and explain why the established patterns won't work
- Request a planning session with user to discuss alternatives
- Get explicit approval before any deviation from templates
- Document the technical necessity that prevents template usage

## Existing Infrastructure Context:

**Reverse Proxy**: An external reverse proxy **ALREADY EXISTS** - you must:
- ❌ **NEVER add nginx, apache, caddy, or any reverse proxy**
- ✅ **ONLY report what port the application exposes**
- ✅ **Design for direct HTTP backend service**

**Bootc System**: Containers are managed by:
- Auto-updates daily at 02:00 AM via `bootc-update.timer`
- Automatic rollback on failure via health checks
- Runtime configuration via `/etc/iago/containers/{name}.env` files
- Service management via `bootc@{container-name}.service` template

## MANDATORY: Planning Phase

**Before creating ANY files, you MUST:**

1. **Review External Services**: Check the application's requirements against existing infrastructure:
   - Does it need PostgreSQL? (Check if already available in architecture)
   - Does it need Redis/Valkey? (Check if already available)
   - Does it need a web server? (FORBIDDEN - reverse proxy exists)
   - Any message queues, databases, or caches?

2. **Validate Template Compatibility**: Confirm the application can work with:
   - `bootc@{name}.service` template (single process startup)
   - Standard environment variable configuration
   - Shared init/health script patterns

3. **Present Implementation Plan**: Show user your approach and get approval before proceeding

4. **Report Port Information**: Clearly state what port(s) the application will expose

**If templates cannot be used, STOP and explain why. Do not proceed without user approval.**

---

## MANDATORY: Use Shared Templates

Before creating any files from scratch, check `containers/_shared/` for reusable templates:
- **Scripts**: `basic-init-template.sh`, `health-check-template.sh`, `fetch-secrets-template.sh`
- **Systemd**: `secrets-service.template`, `secrets-watcher.template`
- **Documentation**: Always read `_shared/README.md` for detailed usage instructions

The _shared templates provide consistent patterns across all containers. Copy and customize them rather than creating files from scratch.

## Initial Setup

First, create the container directory structure:
```bash
# From the iago root directory
mkdir -p containers/[CONTAINER_NAME]/scripts
```

## Container Structure to Create:

Create the following files in `containers/[CONTAINER_NAME]/`:

### 1. Containerfile
- Use base image: `FROM quay.io/fedora/fedora-bootc:42`
- **Always install 1Password CLI** (standard in all iago containers):
  ```dockerfile
  # Install 1Password CLI (standard in all iago containers)
  RUN curl -sSfL https://downloads.1password.com/linux/tar/stable/x86_64/1password-cli-latest-linux_amd64.tar.gz | \
      tar -xzO op > /usr/local/bin/op && \
      chmod +x /usr/local/bin/op
  ```
- Create a system user named after the application
- **Pin application and dependency versions** for reproducible builds
- **Always create 1Password token placeholder**:
  ```dockerfile
  # Create 1Password token placeholder (auto-activating)
  RUN mkdir -p /etc/iago/secrets && \
      echo "# No secrets configured for this container" > /etc/iago/secrets/1password-token.txt && \
      echo "# To enable: replace with your service account token" >> /etc/iago/secrets/1password-token.txt && \
      chmod 0600 /etc/iago/secrets/1password-token.txt
  ```
- **Copy 1Password infrastructure from _shared** (don't create from scratch):
  ```dockerfile
  # Copy the shared 1Password templates - DO NOT create these files yourself
  COPY containers/_shared/systemd/secrets-service.template /etc/systemd/system/[name]-secrets.service
  COPY containers/_shared/systemd/secrets-watcher.template /etc/systemd/system/[name]-secrets-watcher.path

  # Copy the fetch secrets script template
  COPY containers/_shared/scripts/fetch-secrets-template.sh /usr/local/bin/fetch-[name]-secrets.sh

  # Replace {SERVICE} placeholders with your actual service name
  RUN sed -i 's/{SERVICE}/[name]/g' /etc/systemd/system/[name]-secrets.service && \
      sed -i 's/{SERVICE}/[name]/g' /etc/systemd/system/[name]-secrets-watcher.path && \
      sed -i 's/{SERVICE}/[name]/g' /usr/local/bin/fetch-[name]-secrets.sh && \
      chmod +x /usr/local/bin/fetch-[name]-secrets.sh

  # Enable 1Password infrastructure
  RUN systemctl enable [name]-secrets.service [name]-secrets-watcher.path
  ```

  **Note**: The fetch script template will need customization after deployment if you need real secrets. See the 1Password section below.
- Copy scripts from `scripts/*` to `/usr/local/bin/`
- Make scripts executable: `RUN chmod +x /usr/local/bin/*`
- **DO NOT include ENTRYPOINT or CMD**: Bootc containers boot via systemd, not container entrypoints. The `bootc@{name}.service` template handles application startup, not the container itself.
- Include health check command setup if applicable

### 2. scripts/init.sh
**Copy from the shared template:**
```bash
# From the iago root directory
cp containers/_shared/scripts/basic-init-template.sh containers/[CONTAINER_NAME]/scripts/init.sh
sed -i 's/{SERVICE}/[CONTAINER_NAME]/g' containers/[CONTAINER_NAME]/scripts/init.sh
```
- This template creates `/var/lib/[name]` and `/var/log/[name]` directories
- Sets proper permissions for the service user
- Add any service-specific initialization after the template content

### 3. scripts/health.sh
**Copy from the shared template:**
```bash
cp containers/_shared/scripts/health-check-template.sh containers/[CONTAINER_NAME]/scripts/health.sh
sed -i 's/{SERVICE}/[CONTAINER_NAME]/g' containers/[CONTAINER_NAME]/scripts/health.sh
```
- Provides consistent health check format
- Add service-specific health checks (e.g., port checks, API endpoints)
- Should verify the application is actually running, not just systemd status

### 4. README.md (REQUIRED)
**Must include these exact sections:**

#### Required Sections:
- **Version Information**: Document all pinned versions (base image, packages, application)
- **Service Information**: **MUST state it uses `bootc@[name].service`** (NOT a custom service)
- **Port Information**: **MUST clearly state exposed port(s) for reverse proxy configuration**
- **Container Auto-Updates**: Explain the daily 02:00 AM auto-update pulls new images (doesn't rebuild)
- **Service Management**: Use `systemctl` commands with `bootc@[name].service`
- **Reverse Proxy Integration**: State that external reverse proxy handles TLS/routing
- **Secrets Management**: Always document 1Password capability (even if unused):
  - "This container includes 1Password CLI for future secret management"
  - "Currently no secrets are configured"
  - "To add secrets: see iago 1Password integration documentation"
- **Update Strategies**:
  - How to update versions: Edit Containerfile, rebuild, push to registry
  - How to disable auto-updates via `/etc/iago/containers/[name].env`
- **Troubleshooting**: Include checking `bootc-update.service` logs

## Customizing the Templates

After copying templates from `_shared/`:

1. **For init.sh**: Add any service-specific initialization after the basic template content
2. **For health.sh**: Add health checks specific to your application (port checks, process checks, etc.)
3. **For fetch-secrets script**: This will need customization when you actually need secrets:
   - Edit the script to add your specific 1Password references
   - Remove the template error block
   - See `_shared/README.md` for examples

**Important**: The templates use `{SERVICE}` as a placeholder. Always replace it with your actual service name using sed or manual editing.

## 1Password Secrets Management (Always Ready):

All iago containers include complete 1Password infrastructure by default. The system automatically handles all scenarios without manual setup:

### Auto-Activation Scenarios

**No secrets needed** (default):
- Container runs normally with no secret-related activity
- Services run successfully and log "No secrets configured"

**Local testing**:
- `LOCAL_TESTING=true podman run ...`
- `ENVIRONMENT=development podman run ...`
- Automatically creates mock secrets for testing

**Real secrets**:
- Customize the fetch script with your 1Password references
- Add real token to activate automatic secret fetching

### When You Need Real Secrets

#### 1. Customize the fetch script:
The container includes `/usr/local/bin/fetch-[name]-secrets.sh` copied from the shared template.
**After deployment**, edit this script to:
- Add your specific 1Password vault/item references (replacing the examples)
- Remove the template error block at the end
- Follow the examples in the template for different secret types

#### 2. Activate with real token:
```bash
# SSH to the deployed machine
ssh user@machine

# Replace placeholder with real token
sudo nano /etc/iago/secrets/1password-token.txt
# (Replace content with your 1Password service account token)

# Secrets are automatically fetched when file is saved
```

#### 3. Optional: Make main service depend on secrets:
```dockerfile
# In your main service file, add:
# After=[name]-secrets.service
# Wants=[name]-secrets.service
```

### Local Testing Made Easy

Test containers locally without any 1Password setup:

```bash
# Build container
podman build -t localhost:5000/myservice:test .

# Test with no secrets (works perfectly)
podman run --rm --privileged localhost:5000/myservice:test

# Test with mock secrets for development
LOCAL_TESTING=true podman run --rm --privileged localhost:5000/myservice:test

# Alternative development mode
ENVIRONMENT=development podman run --rm --privileged localhost:5000/myservice:test

# Mock token testing (creates same mock secrets as LOCAL_TESTING)
echo "MOCK_TOKEN_FOR_LOCAL_TESTING" > /tmp/mock-token.txt
podman run --rm --privileged \
  -v /tmp/mock-token.txt:/etc/iago/secrets/1password-token.txt:ro \
  localhost:5000/myservice:test
```

### How the Auto-Activation Works

1. **Smart Detection**: The fetch script automatically detects:
   - Local testing mode (creates mock secrets)
   - No token file or placeholder content (exits successfully)
   - Real 1Password token (fetches actual secrets if script is customized)

2. **File Watcher**: Always monitors `/etc/iago/secrets/1password-token.txt` for changes

3. **Auto-Trigger**: Any change to the token file triggers the secrets service

4. **Zero Configuration**: Works perfectly with no secrets, no manual setup required

### Security Notes
- All secret files have 0600 permissions (root only)
- 1Password service account tokens should have minimal vault access
- No secrets ever stored in public repos or ignition files
- Mock secrets only used in local testing environments

### 5. FORBIDDEN FILES (DO NOT CREATE):
- ❌ Individual systemd service files (like `[name]-app.service`)
- ❌ Any systemd timer files
- ❌ Update scripts (handled by bootc system)
- ❌ Reverse proxy configurations (nginx, apache, caddy config files)
- ❌ TLS certificates or SSL configurations
- ❌ Any files not copied from `containers/_shared/` templates

## Containerfile Best Practices:

When creating your Containerfile, follow these modern best practices:

### Security and User Management
- **Run as non-root**: Create a system user for the application and use `USER` directive
- **Use minimal base images**: Start with `fedora-bootc:42` and only install what's needed
- **Don't expose unnecessary ports**: Only EXPOSE ports your application actually uses

### Build Optimization
- **Layer optimization**: Combine related RUN commands to reduce layers
- **Order by change frequency**: Put least-changing commands first to maximize cache
- **Use heredocs for readability**: For complex RUN commands:
  ```dockerfile
  RUN <<EOF
  set -xeuo pipefail
  dnf install -y package1 package2
  dnf clean all
  EOF
  ```

### Best Practices Specific to Bootc
- **Version pinning**: Pin application and dependency versions for reproducible builds:
  ```dockerfile
  # Base image uses stable fedora-bootc:42
  FROM quay.io/fedora/fedora-bootc:42

  # Pin package versions when critical
  RUN dnf install -y \
      postgresql-16.1-1.fc42 \
      nginx-1.24.0-2.fc42 \
      && dnf clean all

  # Pin binary releases
  ARG APP_VERSION=v1.2.3
  RUN curl -L https://github.com/org/app/releases/download/${APP_VERSION}/app-linux-amd64 \
      -o /usr/local/bin/app && chmod +x /usr/local/bin/app

  # Pin git commits for source builds
  RUN git clone https://github.com/org/app.git && \
      cd app && \
      git checkout abc123def456 && \
      make install
  ```
- **COPY vs ADD**: Always use COPY unless you need ADD's auto-extraction features
- **Filesystem layout**: Remember /var is mutable, /usr is immutable in bootc
- **Always run bootc lint**: Include `RUN bootc container lint` as your final command. This validates:
  - Systemd is properly installed and configured
  - Filesystem layout follows bootc conventions (/usr immutable, /var mutable)
  - No conflicting container patterns (ENTRYPOINT/CMD vs systemd)
  - Service files are properly formatted
  - Container is compatible with bootc update mechanisms
  - Prevents deployment issues by catching problems at build time

### Multi-Stage Builds
For applications that require compilation:
```dockerfile
# Build stage
FROM quay.io/fedora/fedora-bootc:42 AS builder
# Install build dependencies
# Compile application

# Runtime stage
FROM quay.io/fedora/fedora-bootc:42
# Copy only necessary artifacts from builder
COPY --from=builder /path/to/binary /usr/local/bin/
```

## Version Management Strategy:

### Why Version Pinning Matters
- **Reproducible builds**: Same Containerfile produces identical images months later
- **Controlled updates**: Version changes are intentional, not accidental
- **Debugging**: Know exactly what versions are running in production
- **Security**: Review version changes before applying them

### How to Manage Versions
1. **Document version choices** in your README.md:
   ```markdown
   ## Version Information
   - Base Image: fedora-bootc:42
   - PostgreSQL: 16.1-1.fc42
   - Application: v1.2.3
   ```

2. **Update process**:
   - Test new versions in development first
   - Update Containerfile with new pinned versions
   - Build and test container locally
   - Commit changes with clear message about version updates
   - Deploy to production

3. **Finding available versions**:
   - Fedora packages: `dnf info <package>` or https://packages.fedoraproject.org/
   - Container images: Check registry tags (e.g., Quay.io, Docker Hub)
   - GitHub releases: Check releases page of the project

### Relationship to Auto-Updates
- **Build time**: Versions are pinned in Containerfile (reproducible)
- **Run time**: bootc auto-update pulls new container images you've built
- **Key insight**: Auto-updates deploy your pre-built containers, they don't rebuild them
- Version updates happen when YOU rebuild with updated Containerfile

## Bootc System Architecture Clarification:

### Pattern Compliance: bootc@ Template is MANDATORY

**The `bootc@{name}.service` template MUST be used unless technically impossible.**

**Template works for:**
- Single process applications ✅
- Applications configurable via environment variables ✅
- Standard HTTP services ✅
- Applications with simple startup requirements ✅

**IF template cannot work (RARE), you must:**
1. **STOP implementation immediately**
2. **Explain the specific technical limitation**
3. **Request planning session with user**
4. **Get explicit approval for deviation**
5. **Document why templates are insufficient**

**Valid reasons for deviation (must be proven):**
- Multiple coordinated processes required
- Special systemd features needed (socket activation, etc.)
- Complex startup sequences that cannot be handled via init scripts
- Technical requirements incompatible with container-based execution

### Filesystem Considerations in Bootc

**Immutable (/usr)** - Versioned with container:
- Application binaries and libraries
- Static configuration files
- Documentation and assets
- Default application data

**Mutable (/var, /etc)** - Machine-local state:
- Logs (/var/log)
- Runtime data (/var/lib)
- Machine-specific configuration (/etc)
- Temporary files (/var/tmp)

Example: If your app installs to `/var/www`, move it to `/usr/share/www`:
```dockerfile
# Move static content from /var to /usr
RUN mv /var/www/* /usr/share/www/ && \
    rmdir /var/www && \
    ln -s /usr/share/www /var/www
```

### State Management Best Practices

- **Persistent data**: Store in /var/lib/[appname]
- **Configuration**: Machine-specific in /etc, defaults in /usr/share
- **Logs**: Always write to /var/log/[appname]
- **Secrets**: Use /etc/iago/secrets/[name]-[service]

## Key Points to Remember:

1. The container will be started by `bootc@[name].service` using the generic template (unless you need custom systemd services)
2. Runtime configuration comes from `/etc/iago/containers/[name].env`
3. Auto-updates are handled by the existing `bootc-update.sh` script
4. The container should be designed to run with podman in privileged mode with host networking
5. Health checks should be built into the container for the auto-update rollback feature

## Application Reference (Recommended):

If you have a non-bootc Dockerfile for this application, include it here:
```dockerfile
[Paste the standard Dockerfile]
```

This helps me understand:
- Required packages and dependencies
- Build steps and compilation needs
- Runtime configuration patterns
- Port exposures and user permissions

If no Dockerfile is available, provide:
- Application name and purpose
- Required packages/dependencies
- Build/compilation steps if needed
- Runtime requirements (ports, volumes, environment variables)
- Configuration needs
- External services or databases required

## Troubleshooting Common Issues:

### Example Containerfile Structure
```dockerfile
# ... build steps and installation ...

# Copy scripts from scripts/* to /usr/local/bin/
COPY containers/[name]/scripts/* /usr/local/bin/
RUN chmod +x /usr/local/bin/*

# No ENTRYPOINT or CMD needed - bootc containers boot via systemd
# The bootc@{name}.service template handles application startup

# Always run bootc container lint as final step
RUN bootc container lint
```

### bootc container lint failures
- **"No systemd in container"**: Ensure `systemd` package is installed
- **"Missing /usr/lib/systemd"**: Base image might not be bootc-compatible
- **"Invalid filesystem layout"**: Check you're not modifying immutable paths incorrectly

### Service not starting
```bash
# Check service status
systemctl status bootc@[name].service

# View detailed logs
journalctl -xeu bootc@[name].service

# Check podman directly
podman logs bootc-[name]
```

### Permission issues
- Ensure files in /usr are world-readable (0644 not 0640)
- User directories should be created with proper ownership
- Secrets in /etc/iago/secrets need 0600 permissions

### Update failures
```bash
# Check update service logs
journalctl -xeu bootc-update.service

# Manually test container pull
podman pull [registry]/[name]:latest

# Force rollback if needed
systemctl stop bootc@[name].service
podman rm bootc-[name]
```

### 1Password secrets not working
All containers have 1Password CLI available. If secrets aren't being fetched:
```bash
# Check if token file exists and has content (exists in all containers)
ls -la /etc/iago/secrets/1password-token.txt
cat /etc/iago/secrets/1password-token.txt

# Check if secrets systemd services are enabled (only if you activated them)
systemctl list-unit-files | grep [name]-secrets

# Check secrets service logs (if activated)
journalctl -xeu [name]-secrets.service

# Manually trigger secret fetch (if activated)
systemctl start [name]-secrets.service

# Verify 1Password CLI works (available in all containers)
op --version
op whoami
```

### Testing locally before deployment
```bash
# Build the container
podman build -t localhost:5000/[name]:latest .

# Test run with bootc-like privileges
podman run --privileged --network host \
  -v /etc/iago/containers:/etc/iago/containers:ro \
  localhost:5000/[name]:latest

# Validate with bootc lint during build
podman build --no-cache -t test-[name] .
```

## Best Practices When Using Templates

1. **Always check _shared first**: Don't create scripts from scratch if a template exists
2. **Copy, don't symlink**: Templates should be copied to your container directory
3. **Maintain consistency**: Use the same service name throughout all files
4. **Read the template comments**: Templates include usage instructions and examples
5. **Test after customization**: Verify your customizations work before deployment

For complex services that don't fit the template patterns, create custom scripts but follow the naming conventions established by the templates.

---

Note: This follows the conventions established in the iago project where containers are managed by a sophisticated bootc runtime system rather than individual systemd services.
