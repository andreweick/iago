# Iago - Fedora CoreOS Machine Management

Iago is a Go CLI application that helps you create, manage, and update Fedora CoreOS machines with bootc containers for your homelab and VPS infrastructure.

## What's in a name?
Iago from Othello

Iago is perhaps the quintessential example of a character whose central motivation, once set, is almost impossible to sway. His manipulative schemes against Othello and Cassio, fueled by perceived slights and deep-seated resentment, become an absolute, unbendable obsession.

Immovable (Immutable?) in his resolve: Despite opportunities to confess or relent, Iago maintains his deceptive facade and continues his machinations with unwavering determination. His ability to appear loyal while orchestrating destruction showcases his commitment to his malevolent goals.

Never changes his mind: He consistently pursues his vengeful plot against Othello, manipulating every situation to serve his purposes, refusing to abandon his course even when it leads to tragedy.

Always stays the same (in his core nature): His character remains fundamentally duplicitous and calculating throughout the play. Even when finally exposed, he maintains his silence, staying true to his secretive and manipulative nature to the very end.

## Useful Commands

### iago init

Initialize a new machine with container scaffold, configuration, and ignition file:

```bash
# Full initialization - creates both machine config and container scaffold (default)
iago init my-app

# Create only machine configuration (config, template, ignition file)
iago init --machine-only db-server

# Create only container scaffold (directory, Containerfile, prompt)
iago init --container-only web-server

# Initialize with custom domain
iago init --domain example.com web-server

# Initialize without MAC address generation
iago init --generate-mac=false db-server

# Initialize with specific domain and no MAC
iago init --domain organmorgan.com --generate-mac=false nas-01

# List all configured machines
iago list
```

**What iago init creates:**

*Default (no flags) - Full initialization:*
- Container scaffold: `containers/{machine-name}/`
- Machine config: `machines/{machine-name}/machine.toml`
- Machine template: `machines/{machine-name}/butane.yaml.tmpl`
- Ignition file: `output/ignition/{machine-name}.ign`

*With `--machine-only` flag:*
- Machine config: `machines/{machine-name}/machine.toml`
- Machine template: `machines/{machine-name}/butane.yaml.tmpl`
- Ignition file: `output/ignition/{machine-name}.ign`

*With `--container-only` flag:*
- Container scaffold: `containers/{machine-name}/`

### iago build

Build and push containers using pure Go (no Docker/Podman dependency):

```bash
# Build single container and push to registry
iago build my-app

# Build all containers
iago build --all

# Build for local testing (pushes to localhost:5000)
iago build my-app --local

# Build only (in memory), don't push anywhere (for testing)
iago build my-app --no-push

# Build and sign with cosign keyless signing
iago build my-app --sign

# Build with explicit authentication
iago build my-app --token your-token
```

**Container build features:**
- Pure Go building (no Docker daemon required)
- Automatic registry push to configured registry
- Support for GitHub Container Registry (ghcr.io)
- Local registry support for testing
- Optional cosign signing

## Environment Variables and Secrets

Iago requires minimal configuration to get started but supports multiple authentication methods for container registry access.

### Required Configuration

**None!** Iago works out of the box with:
- Auto-generated secrets and passwords (embedded in ignition files)
- Local development without external dependencies
- Default GitHub Container Registry configuration (customizable in `config/defaults.toml`)

### Optional Container Registry Authentication

For pushing containers to registries, Iago supports multiple authentication methods (in priority order):

#### 1. CLI Flags (Highest Priority)
```bash
# Explicit authentication via command line
iago build my-app --token your-token
```

#### 2. Environment Variables

**Required Environment Variables:**

| Variable Name | Purpose | Example Value | How to Get |
|---------------|---------|---------------|------------|
| `GITHUB_TOKEN` | GitHub Container Registry authentication | `ghp_xxxxxxxxxxxxxxxxxxxx` | Create at [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens) |

**How to Set Environment Variables:**

```bash
# Bash/Zsh (Linux/macOS) - temporary for current session
export GITHUB_TOKEN="ghp_your_github_personal_access_token"

# Bash/Zsh - permanent (add to ~/.bashrc or ~/.zshrc)
echo 'export GITHUB_TOKEN="ghp_your_github_personal_access_token"' >> ~/.bashrc
source ~/.bashrc

# Fish shell
set -gx GITHUB_TOKEN "ghp_your_github_personal_access_token"

# Windows Command Prompt
set GITHUB_TOKEN=ghp_your_github_personal_access_token

# Windows PowerShell
$env:GITHUB_TOKEN="ghp_your_github_personal_access_token"

# Docker/systemd service files
Environment=GITHUB_TOKEN=ghp_your_github_personal_access_token

# Then build containers
iago build my-app
```

#### 3. 1Password Integration (Optional)

**Required Environment Variables:**

| Variable Name | Purpose | Example Value | How to Get |
|---------------|---------|---------------|------------|
| `OP_SERVICE_ACCOUNT_TOKEN` | 1Password service account authentication | `ops_xxxxxxxxxxxxxxxxxxxx` | Create at [1Password Developer > Service Accounts](https://developer.1password.com/docs/service-accounts/) |

```bash
# Set 1Password service account token
export OP_SERVICE_ACCOUNT_TOKEN="ops_your_service_account_token"

# Iago will automatically fetch GitHub tokens from 1Password vault
# Default vault location: op://github/personal-access-token/credential
iago build my-app
```

### Why These Are Needed

- **Container Registry Push**: Authentication is only required when pushing containers to registries (ghcr.io, docker.io, etc.)
- **No Push Operations**: Use `--no-push` flag for local testing without any authentication
- **Local Registry**: Use `--local` flag to push to localhost:5000 without authentication
- **GitHub SSH Keys**: Automatically fetched from GitHub API using the `github_username` in `config/defaults.toml` (no token required for public keys)

### Authentication Priority

1. **CLI `--token` flag** (overrides everything)
2. **`GITHUB_TOKEN` environment variable**
3. **1Password via `OP_SERVICE_ACCOUNT_TOKEN`** (fetches from vault: `op://github/personal-access-token/credential`)
4. **No authentication** (fails with helpful error message)

### Local Development (No Secrets Needed)

```bash
# Work completely offline
iago init my-app                           # Create machine and container
iago build my-app --no-push     # Build without pushing
iago ignite my-app                         # Generate ignition file

# Or use local registry
iago build my-app --local       # Push to localhost:5000
```

### Production Setup

For production use, set one of:
```bash
# Option 1: GitHub Personal Access Token
export GITHUB_TOKEN="ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

# Option 2: 1Password Service Account (if using 1Password)
export OP_SERVICE_ACCOUNT_TOKEN="ops_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

**GitHub Token Permissions**: Only requires `write:packages` scope for GitHub Container Registry.

## Features

- **Base Configuration**: Common settings for all machines (users, SSH keys, security)
- **Container System**: Plugin-based container workloads with bootc containers
- **Network Management**: MAC address configuration for homelab DHCP reservations
- **Auto-Updates**: Automatic CoreOS and container updates with rollback
- **Secret Management**: Secure generation and storage of passwords and keys
- **Build System**: Generate ignition files for all machines at once
- **Container Building**: Pure Go container building without Docker/Podman dependency
- **Container Signing**: Optional cosign keyless signing support
- **Lightweight CLI**: Built with urfave/cli for minimal dependencies

## Quick Start

1. **Build the application**:
   ```bash
   make build
   # or
   go build -o bin/iago ./cmd/iago
   ```

2. **Customize defaults** in `config/defaults.toml`:
   ```toml
   [user]
   username = "your-username"
   github_username = "your-github-username"
   ```

3. **Initialize your first machine**:
   ```bash
   ./bin/iago init db-01
   ```
   This creates:
   - Container scaffold: `containers/db-01/`
   - Machine config: `machines/db-01/machine.toml`
   - Machine-specific butane: `machines/db-01/butane.yaml.tmpl`
   - Ignition file: `output/ignition/db-01.ign`

4. **Customize and build your container**:
   ```bash
   # Edit container files
   cd containers/db-01/
   # Edit Containerfile, scripts, config files, etc.

   # Build and push using iago (pure Go, no Docker/Podman needed)
   ./bin/iago build db-01

   # Alternative: Use podman directly
   podman build -t ghcr.io/your-username/db-01:latest .
   podman push ghcr.io/your-username/db-01:latest
   ```

5. **Use the ignition file to boot your CoreOS machine**

## Project Structure

```
iago/
├── cmd/iago/           # CLI application entry point
├── config/                # Global configuration
│   ├── defaults.toml      # System-wide default settings
│   └── scripts/           # Embedded scripts (bootc-update.sh, motd.sh)
├── machines/              # 🖥️ CoreOS machine definitions
│   └── {machine-name}/    # Machine-specific directory
│       ├── machine.toml   # Machine configuration (name, FQDN, MAC, container image)
│       └── butane.yaml.tmpl # Complete machine-specific butane template
├── containers/            # 📦 Container workload definitions
│   ├── _shared/           # Shared container resources
│   │   ├── scripts/       # Common script templates
│   │   └── systemd/       # Common systemd patterns
│   └── {machine-name}/    # Container for specific machine
│       ├── Containerfile  # Bootc container definition
│       ├── config/        # Container configuration files
│       ├── scripts/       # Container scripts (init.sh, health.sh)
│       ├── systemd/       # Systemd service definitions
│       └── tests/         # Container-specific tests
├── internal/              # Application source code
│   ├── build/            # Ignition build system
│   ├── butane/           # Butane template rendering
│   ├── container/        # Pure Go container building (go-containerregistry)
│   ├── github/           # GitHub SSH key fetching
│   ├── ignition/         # Ignition file generation
│   ├── machine/          # Machine configuration management
│   ├── scaffold/         # Auto-scaffolding for new machines
│   ├── user/             # User configuration handling
│   └── workload/         # Workload plugin system
│       └── postgres/     # PostgreSQL workload implementation
├── Makefile              # Task automation (build, container operations, development)
└── output/              # Generated build artifacts (gitignored)
    └── ignition/        # Generated ignition files (.ign)
```

## Commands

### Core Commands

```bash
# Initialize new machine: create config, container scaffold, and ignition file
iago init --domain spouterinn.org db-02
iago init --generate-mac=false db-03    # Skip MAC generation
iago init --machine-only db-04          # Create only machine configuration
iago init --container-only web-app      # Create only container scaffold

# List all configured machines (with alias)
iago list
iago ls

# Remove a machine and all its files
iago rm db-01                     # Interactive confirmation
iago rm --force db-01             # Skip confirmation
iago remove db-01                 # Using alias
iago delete db-01                 # Another alias

# Validate configuration (with alias)
iago validate
iago val

# Generate ignition file for existing machine
iago ignite postgres-01
iago ignite --output /tmp/postgres-01.ign postgres-01
iago ignite --strict=false postgres-01  # Disable strict mode
iago gen postgres-01  # using alias with default output

# Generate ignition file for existing machine
iago ignite postgres-01
iago ignite --output /tmp/postgres-01.ign postgres-01
iago ignite --strict=false postgres-01  # Disable strict mode
```

### Container Build Commands

```bash
# Build and push container for a workload (pure Go - no Docker/Podman needed)
iago build db-01

# Build all workloads
iago build --all

# Build and push to local registry for testing
iago build db-01 --local

# Build only, don't push anywhere
iago build db-01 --no-push

# Build with custom tag
iago build db-01 --tag v1.2.3

# Build and sign with cosign keyless signing
iago build db-01 --sign
```

### Make Tasks

```bash
# Development workflow
make dev-full     # Format, lint, test, build, validate
make dev          # Build and validate
make build        # Build iago binary
make test         # Run tests
make lint         # Run linting
make fmt          # Format code
make validate     # Validate configuration

# Container management
make build-all-containers             # Build all containers using iago
make build-container-signed NAME=db-01  # Build with cosign signing
make build-container-podman NAME=db-01  # Legacy: build with podman
make push-container NAME=db-01        # Legacy: push with podman
make deploy-container NAME=db-01      # Legacy: build and push with podman

# Machine management
make deploy-machine NAME=db-01        # Complete workflow: init -> build -> ignite
make deploy-machine NAME=db-01 DOMAIN=home.lab  # With custom domain

# Ignition and build
make build-ignitions                  # Build all ignition files

# Utilities
make show-container NAME=postgres-01  # Show container directory structure
make show-machine NAME=postgres-01    # Show machine directory structure
make clean                            # Clean generated files

# Direct iago execution
make iago list                     # Run iago with arguments
make iago init db-01              # Initialize machine via make

# Release management (requires goreleaser)
make snapshot                         # Generate development snapshot
make release                          # Create tagged release
```

## Configuration

### Machine Configuration (`machines/{name}/machine.toml`)

Each machine has its own configuration file in its dedicated directory:

```toml
# machines/postgres/machine.toml
name = "postgres"                      # Machine name (matches directory)
fqdn = "postgres.organmorgan.com"      # Fully qualified domain name
container_image = "ghcr.io/andreweick/postgres"  # Container registry path
container_tag = "latest"               # Container tag (optional, defaults to "latest")
mac_address = "02:05:56:39:1b:21"      # MAC for DHCP (optional)

# machines/caddy-work/machine.toml
name = "caddy-work"
fqdn = "caddy-work.spouterinn.org"
container_image = "ghcr.io/andreweick/caddy-work"
container_tag = "latest"
mac_address = "02:05:56:54:22:07"
network_interface = "ens18"            # Network interface (optional)
```

**Machine Configuration Parameters:**

| Parameter           | Required | Description                                      | Example                     |
|---------------------|----------|--------------------------------------------------|-----------------------------|
| `name`              | ✅       | Machine identifier (must match directory name)  | `"postgres"`               |
| `fqdn`              | ✅       | Fully qualified domain name                     | `"postgres.organmorgan.com"` |
| `container_image`   | ❌       | Container registry path (defaults to `{registry}/{name}`) | `"ghcr.io/user/postgres"` |
| `container_tag`     | ❌       | Container tag (defaults to `"latest"`)          | `"v1.2.3"`                 |
| `mac_address`       | ❌       | MAC address for DHCP reservations               | `"02:05:56:39:1b:21"`      |
| `network_interface` | ❌       | Network interface name                           | `"ens18"`                  |

**Simplified Naming Convention**: The machine name is used consistently for:
- Directory structure: `machines/postgres/`, `containers/postgres/`
- Ignition filename: `postgres.ign`
- Container image: `ghcr.io/user/postgres:latest` (if not overridden)
- Systemd service: `bootc@postgres.service`
- Secret files: `/etc/iago/secrets/postgres-password`

### Default Settings (`config/defaults.toml`)

```toml
[user]
username = "maeick"                    # Primary user account
github_username = "andreweick"         # GitHub username for SSH key fetching
groups = ["sudo", "wheel"]             # User groups
password_hash = "$6$..."

[admin]
username = "admin"                     # Admin user account
groups = ["sudo", "wheel"]
password_hash = "$6$..."

[network]
timezone = "America/New_York"          # System timezone
default_network_interface = "eth0"     # Default network interface

[updates]
stream = "stable"                      # CoreOS update stream (stable/testing/next)
strategy = "periodic"                  # Update strategy
period = "daily"                       # Update frequency
reboot_time = "03:00"                  # CoreOS reboot time

[bootc]
update_time = "02:00:00"               # Container update time (HH:MM:SS)
health_check_wait = 30                 # Seconds to wait before health check

[container_registry]
url = "ghcr.io/andreweick"             # Your container registry URL
```

**Configuration Parameters:**

#### User Section
| Parameter        | Description                               | Example              |
|------------------|-------------------------------------------|----------------------|
| `username`       | Primary user account name                 | `"maeick"`           |
| `github_username`| GitHub username for SSH key fetching     | `"andreweick"`       |
| `groups`         | User groups for permissions               | `["sudo", "wheel"]`  |
| `password_hash`  | SHA-512 password hash                     | `"$6$salt$hash..."`  |

#### Admin Section
| Parameter        | Description                               | Example              |
|------------------|-------------------------------------------|----------------------|
| `username`       | Admin account name                        | `"admin"`            |
| `groups`         | Admin groups                              | `["sudo", "wheel"]`  |
| `password_hash`  | SHA-512 password hash                     | `"$6$salt$hash..."`  |

#### Network Section
| Parameter                  | Description                               | Example              |
|----------------------------|-------------------------------------------|----------------------|
| `timezone`                 | System timezone                           | `"America/New_York"` |
| `default_network_interface`| Default network interface                 | `"eth0"`             |

#### Updates Section
| Parameter     | Description                               | Values                              |
|---------------|-------------------------------------------|-------------------------------------|
| `stream`      | CoreOS update stream                      | `"stable"`, `"testing"`, `"next"` |
| `strategy`    | Update strategy                           | `"periodic"`                       |
| `period`      | Update frequency                          | `"daily"`                          |
| `reboot_time` | Reboot time for CoreOS                    | `"03:00"`                          |

#### Bootc Section
| Parameter              | Description                               | Example              |
|------------------------|-------------------------------------------|----------------------|
| `update_time`          | Container update time                     | `"02:00:00"`         |
| `health_check_wait`    | Health check delay (seconds)             | `30`                 |

#### Container Registry Section
| Parameter     | Description                               | Example              |
|---------------|-------------------------------------------|----------------------|
| `url`         | Container registry URL                    | `"ghcr.io/username"` |

### MAC Address Generation

- MAC addresses are generated by default for homelab DHCP reservations
- Uses locally administered MAC address range (starts with `02:`)
- Generated MACs follow format: `02:05:56:xx:xx:xx` (random last 3 octets)
- Use `--generate-mac=false` to disable MAC generation
- MAC prefix can be configured in defaults.toml (currently hardcoded to `02:05:56`)
- Use `iago init` to generate new machine configs with random MACs

### Machine-Specific Butane Configuration

Iago uses a per-machine butane template system where each machine has its own complete template:

- **Machine-specific template** (`machines/{machine-name}/butane.yaml.tmpl`) - Complete machine configuration
- **Template variables** - Dynamic values from machine config and defaults
- **Local script references** - Scripts embedded from `config/scripts/`

#### Auto-Generated Template

When you run `iago init {machine-name}`, a complete `butane.yaml.tmpl` file is automatically created in `machines/{machine-name}/`:

```yaml
variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      groups:
{{ range .User.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .User.PasswordHash }}"
{{ if .UserSSHKeys }}      ssh_authorized_keys:
{{ range .UserSSHKeys }}        - {{ . }}
{{ end }}{{ end }}
    - name: "{{ .Admin.Username }}"
      groups:
{{ range .Admin.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .Admin.PasswordHash }}"

storage:
  directories:
    - path: /etc/iago
      mode: "0755"
    - path: /etc/iago/secrets
      mode: "0700"
    - path: /etc/iago/containers
      mode: "0755"
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: "{{ .Machine.Name }}"
    # Container configuration
    - path: /etc/iago/containers/{{ .Machine.Name }}.env
      mode: 0644
      contents:
        inline: |
          CONTAINER_IMAGE={{ .Machine.ContainerImage }}:{{ .Machine.ContainerTag }}
          CONTAINER_NAME=bootc-{{ .Machine.Name }}
          HEALTH_CHECK_WAIT=30
          UPDATE_STRATEGY=latest
    # Management scripts (embedded from config/scripts/)
    - path: /usr/local/bin/bootc-manager.sh
      mode: 0755
      contents:
        local: bootc-manager.sh
    - path: /usr/local/bin/bootc-run.sh
      mode: 0755
      contents:
        local: bootc-run.sh

systemd:
  units:
    # Generic bootc template (handles any container)
    - name: bootc@.service
      contents: |
        [Unit]
        Description=Bootc Container %i
        After=network-online.target podman.service

        [Service]
        Type=notify
        Restart=always
        EnvironmentFile=/etc/iago/containers/%i.env
        ExecStart=/usr/local/bin/bootc-run.sh %i

        [Install]
        WantedBy=multi-user.target
```

#### Template Variables

Butane templates support the following template variables:

| Variable                             | Description                      | Example Values                    |
|--------------------------------------|----------------------------------|-----------------------------------|
| `.User.Username`                     | Primary user account             | `"maeick"`                       |
| `.User.Groups`                       | User groups array                | `["sudo", "wheel"]`              |
| `.User.PasswordHash`                 | User password hash               | `"$6$..."`                       |
| `.Admin.Username`                    | Admin account                    | `"admin"`                        |
| `.Admin.Groups`                      | Admin groups array               | `["sudo", "wheel"]`              |
| `.Admin.PasswordHash`                | Admin password hash              | `"$6$..."`                       |
| `.Network.Timezone`                  | System timezone                  | `"America/New_York"`             |
| `.Network.DefaultNetworkInterface`   | Default interface                | `"eth0"`                         |
| `.Updates.Strategy`                  | CoreOS update strategy           | `"periodic"`                     |
| `.Updates.RebootTime`                | CoreOS reboot time               | `"03:00"`                        |
| `.Updates.Stream`                    | CoreOS stream                    | `"stable"`                       |
| `.Bootc.UpdateTime`                  | Container update time            | `"02:00:00"`                     |
| `.ContainerRegistry.URL`             | Registry URL                     | `"ghcr.io/user"`                 |
| `.Machine.Name`                      | Machine name                     | `"postgres"`                     |
| `.Machine.FQDN`                      | Machine FQDN                     | `"postgres.org.com"`             |
| `.Machine.MACAddress`                | MAC address                      | `"02:05:56:39:1b:21"`            |
| `.Machine.ContainerImage`            | Container image path             | `"ghcr.io/user/postgres"`        |
| `.Machine.ContainerTag`              | Container tag                    | `"latest"`                       |
| `.GeneratedSecrets.Password`         | Generated password               | Auto-generated                   |
| `.UserSSHKeys`                       | SSH keys from GitHub             | Fetched from GitHub API          |

#### Benefits

- **Complete machine control** - Each machine has its own complete configuration template
- **Template support** - Use template variables for dynamic configuration
- **Local script embedding** - Scripts from `config/scripts/` are embedded using `local:` references
- **Clear organization** - Each machine's complete configuration is in its own directory
- **Version control friendly** - Machine configurations are isolated and trackable

## Workload System & Bootc Containers

Iago uses a plugin-based workload system where each machine workload corresponds to a **Fedora Bootc container**. This approach provides immutable infrastructure where the entire machine's configuration and applications are packaged as container images.

### Bootc Container Architecture

Each workload creates a container based on `quay.io/fedora/fedora-bootc:42` that includes:

- **Base OS**: Fedora CoreOS with bootc capabilities
- **Application Stack**: Your service (PostgreSQL, web server, etc.)
- **Configuration**: Machine-specific config files and scripts
- **Systemd Services**: Proper service lifecycle management
- **Health Monitoring**: Built-in health checks and logging

### Using the Default Configuration

After deploying a machine with Iago, the default bootc container configuration is completely self-contained and ready to run without additional setup. The auto-generated container includes a simple "hello world" service that demonstrates the bootc architecture and confirms your machine is working correctly.

**What happens automatically after deployment:**
- The machine boots with CoreOS and applies the generated ignition configuration
- The `bootc-{machine-name}.service` systemd service automatically starts
- The container image is pulled from your configured registry (e.g., `ghcr.io/username/machine-name:latest`)
- A demonstration service begins running inside the container, logging messages every 30 seconds
- Health checks validate the service is running properly
- Automatic daily updates are scheduled for 2:00 AM

**No additional configuration required:**
- **Secrets**: All passwords and keys are auto-generated during ignition build and embedded in the machine configuration
- **Container selection**: The machine automatically knows which container image to run based on the configuration in `/etc/iago/containers/{machine-name}.env`
- **1Password integration**: Not used in the default setup - all secrets are self-contained and generated locally
- **Service management**: The bootc service starts automatically on boot and manages container lifecycle

To verify your default configuration is working, SSH to your deployed machine and check the service status:
```bash
ssh user@machine-name.your-domain.com
sudo systemctl status bootc-machine-name.service
sudo journalctl -u bootc-machine-name.service -f
```

You should see the hello world service logging messages, confirming the bootc container architecture is functioning correctly. From this working baseline, you can then customize the container by editing the `Containerfile` and rebuilding with `iago build machine-name`.

### Example Containerfile Structure

Here's what a typical bootc container looks like:

```dockerfile
# Bootc container for your-service
FROM quay.io/fedora/fedora-bootc:42

# Install application and utilities
RUN dnf install -y \
    postgresql-server \
    postgresql-contrib \
    htop \
    && dnf clean all

# Create application user
RUN useradd -r -s /bin/bash your-service-app

# Copy configuration and scripts
COPY config/* /etc/your-service/
COPY scripts/* /usr/local/bin/
COPY systemd/* /etc/systemd/system/

# Make scripts executable
RUN chmod +x /usr/local/bin/*.sh

# Create directories with proper ownership
RUN mkdir -p /var/lib/your-service /var/log/your-service && \
    chown -R your-service-app:your-service-app /var/lib/your-service /var/log/your-service

# Enable the main service
RUN systemctl enable your-service-app.service

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD systemctl is-active --quiet your-service-app.service || exit 1

# Default command
CMD ["/sbin/init"]
```

### Workload Plugin System

Workloads implement a simple interface:

```go
type Workload interface {
    Name() string
    ContainerImage() string
    ContainerTag() string
    GetButaneOverlay() (string, error)  // Contributes systemd services
    GetSecrets() []SecretDefinition     // Defines generated secrets
    Validate(config MachineConfig) error
}
```

Each workload can:
- **Contribute Butane configuration** for systemd services and file configurations
- **Define secrets** that are auto-generated (passwords, keys, etc.)
- **Validate machine configuration** for workload-specific requirements

### Default Workload

Every machine gets a default workload that creates:
- A systemd service `bootc-{machine-name}.service`
- Container configuration for running the bootc image
- Basic directory structure and secrets
- Health monitoring and restart policies

### Specialized Workloads

#### PostgreSQL Workload

The PostgreSQL workload provides:
- Bootc container with PostgreSQL
- Automatic daily backups
- Password generation
- Health monitoring
- Auto-updates with rollback

Generated secrets:
- PostgreSQL password: `/etc/iago/secrets/postgres-password`

## Container Update Workflow

Iago implements a robust container update system that ensures machines automatically stay current while providing rollback capabilities for failed updates.

### Update Schedule

The system operates on a dual-update schedule configured in `config/defaults.toml`:

- **Container Updates**: Daily at **02:00** via `bootc-update.timer`
- **CoreOS Updates**: Daily at **03:00** via Zincati
- **Health Check Wait**: 30 seconds after restart before validation

### How Container Updates Work

Each machine runs a systemd timer that executes the update workflow:

```bash
# The bootc-update.timer runs daily at 02:00
systemctl status bootc-update.timer

# Which executes this service
systemctl status bootc-update.service
```

### Update Process

The `/usr/local/bin/bootc-update.sh` script handles the complete update cycle:

1. **Save Current Image**: Tags current container as `:previous` for rollback
   ```bash
   podman tag ghcr.io/user/machine-name:latest ghcr.io/user/machine-name:previous
   ```

2. **Pull Latest Image**: Downloads the latest container from registry
   ```bash
   podman pull ghcr.io/user/machine-name:latest
   ```

3. **Restart Service**: Restarts the machine's bootc service with new image
   ```bash
   systemctl restart bootc-machine-name.service
   ```

4. **Health Check**: Waits 30 seconds, then verifies service is healthy
   ```bash
   # If service is not active after restart
   systemctl is-active --quiet bootc-machine-name.service || rollback
   ```

5. **Automatic Rollback**: If health check fails, reverts to previous image
   ```bash
   # Restore previous image and restart
   podman tag ghcr.io/user/machine-name:previous ghcr.io/user/machine-name:latest
   systemctl restart bootc-machine-name.service
   ```

### Container Execution

Each machine runs its bootc container via a systemd service `bootc-{machine-name}.service`:

```ini
[Unit]
Description=Bootc machine-name Container
After=network-online.target podman.service
Wants=network-online.target
Requires=podman.service

[Service]
Type=notify
NotifyAccess=all
Restart=always
RestartSec=30
TimeoutStartSec=300

# Always pull latest image before starting
ExecStartPre=/usr/bin/podman pull ghcr.io/user/machine-name:latest

# Run container with host integration
ExecStart=/usr/bin/podman run \
  --rm \
  --name bootc-machine-name \
  --net host \
  --pid host \
  --privileged \
  --security-opt label=disable \
  --volume /etc:/etc \
  --volume /var:/var \
  --volume /run:/run \
  --env MACHINE_NAME=machine-name \
  --sdnotify=conmon \
  --health-cmd "/usr/local/bin/health.sh" \
  --health-interval=30s \
  --health-retries=3 \
  --health-start-period=60s \
  ghcr.io/user/machine-name:latest

ExecStop=/usr/bin/podman stop -t 30 bootc-machine-name

[Install]
WantedBy=multi-user.target
```

### Update Frequency Configuration

You can customize update timing in `config/defaults.toml`:

```toml
[updates]
stream = "stable"           # Fedora CoreOS stream
strategy = "periodic"       # Update strategy
period = "daily"           # Update frequency
reboot_time = "03:00"      # CoreOS reboot time

[bootc]
update_time = "02:00:00"   # Container update time (HH:MM:SS)
health_check_wait = 30     # Seconds to wait before health check
```

### Manual Rollback

If needed, you can manually rollback containers:

```bash
# View available images
sudo podman images | grep machine-name

# Rollback to previous version
sudo podman tag ghcr.io/user/machine-name:previous ghcr.io/user/machine-name:latest
sudo systemctl restart bootc-machine-name.service

# Pin to specific version (edit systemd service)
sudo systemctl edit bootc-machine-name.service
# Change :latest to :specific-tag in ExecStartPre and ExecStart
sudo systemctl daemon-reload
sudo systemctl restart bootc-machine-name.service
```

## Flexible Container Management

Iago implements a **flexible container management system** that completely decouples container configuration from machine provisioning. This allows you to change container images, registries, tags, and update strategies **without rebuilding machines**.

### Architecture Overview

The system uses a **template-based approach** with runtime configuration files:

- **Generic systemd template**: `bootc@.service` handles any container
- **Runtime configuration**: `/etc/iago/containers/{machine}.env` files control container behavior
- **Per-machine templates**: Each machine has its own complete `butane.yaml.tmpl`
- **Auto-management**: `bootc-manager.service` automatically starts containers based on config files

### How It Works

#### 1. Template System

Each machine has a complete `machines/{machine}/butane.yaml.tmpl` file that includes:

```yaml
# Generic bootc template (handles any container)
- name: bootc@.service
  contents: |
    [Unit]
    Description=Bootc Container %i
    After=network-online.target podman.service
    Wants=network-online.target
    Requires=podman.service

    [Service]
    Type=notify
    NotifyAccess=all
    Restart=always
    RestartSec=30
    TimeoutStartSec=300
    EnvironmentFile=/etc/iago/containers/%i.env
    ExecStart=/usr/local/bin/bootc-run.sh %i
    ExecStop=/usr/bin/podman stop -t 30 bootc-%i

    [Install]
    WantedBy=multi-user.target
```

#### 2. Runtime Container Configuration

During machine build, a configuration file is created at `/etc/iago/containers/{machine}.env`:

```bash
# Example: /etc/iago/containers/postgres.env
CONTAINER_IMAGE=ghcr.io/andreweick/postgres:latest
CONTAINER_NAME=bootc-postgres
HEALTH_CHECK_WAIT=30
UPDATE_STRATEGY=latest
```

#### 3. Container Management Scripts

Three key scripts handle container operations:

- **`bootc-manager.sh`**: Scans for container configs and auto-starts services
- **`bootc-run.sh`**: Generic container runner that reads environment files
- **`bootc-update.sh`**: Updates containers based on their configuration

#### 4. Service Instantiation

The system automatically creates services like `bootc@postgres.service` that:
- Read configuration from `/etc/iago/containers/postgres.env`
- Use the generic `bootc@.service` template
- Run the specific container with appropriate settings

### Update Strategies

Configure how containers update by editing the `.env` file:

```bash
# Always update to latest (default)
UPDATE_STRATEGY=latest

# Pin to current version - no automatic updates
UPDATE_STRATEGY=pinned

# Use staging tag for testing before promoting to latest
UPDATE_STRATEGY=staging
```

### Container Flexibility Examples

#### Change Container Image/Registry

```bash
# SSH to machine
ssh user@postgres.organmorgan.com

# Edit container configuration
sudo vim /etc/iago/containers/postgres.env

# Change to different image/registry
CONTAINER_IMAGE=docker.io/library/postgres:15

# Restart service with new container
sudo systemctl restart bootc@postgres.service

# Verify new container is running
sudo podman ps
sudo systemctl status bootc@postgres.service
```

#### Pin to Specific Version

```bash
# Edit container config
sudo vim /etc/iago/containers/postgres.env

# Set specific tag and pin strategy
CONTAINER_IMAGE=ghcr.io/andreweick/postgres:v1.2.3
UPDATE_STRATEGY=pinned

# Restart with pinned version
sudo systemctl restart bootc@postgres.service
```

#### Switch to Staging/Testing

```bash
# Test with staging image
sudo vim /etc/iago/containers/postgres.env

# Use staging tag
CONTAINER_IMAGE=ghcr.io/andreweick/postgres:staging
UPDATE_STRATEGY=staging

sudo systemctl restart bootc@postgres.service
```

### "Break Glass" Emergency Updates

For immediate updates bypassing the normal schedule:

#### Force Immediate Container Update

```bash
# SSH to machine
ssh user@machine.domain.com

# Trigger immediate update (respects UPDATE_STRATEGY)
sudo systemctl start bootc-update.service

# Check update logs
sudo journalctl -u bootc-update.service -f
```

#### Force Specific Container Version

```bash
# Override container config temporarily
sudo vim /etc/iago/containers/postgres.env

# Set emergency version
CONTAINER_IMAGE=ghcr.io/andreweick/postgres:emergency-fix
UPDATE_STRATEGY=pinned

# Immediate restart
sudo systemctl restart bootc@postgres.service
```

#### Manual Container Pull and Restart

```bash
# Pull specific version manually
sudo podman pull ghcr.io/andreweick/postgres:hotfix

# Tag as latest to force use
sudo podman tag ghcr.io/andreweick/postgres:hotfix ghcr.io/andreweick/postgres:latest

# Restart service
sudo systemctl restart bootc@postgres.service
```

### Naming Convention

The system follows a consistent naming pattern:

| Component         | Pattern                          | Example                               |
|-------------------|----------------------------------|---------------------------------------|
| Machine name      | `{name}`                         | `postgres`                            |
| Container image   | `{registry}/{name}:{tag}`        | `ghcr.io/user/postgres:latest`        |
| Service name      | `bootc@{name}.service`           | `bootc@postgres.service`              |
| Container config  | `/etc/iago/containers/{name}.env` | `/etc/iago/containers/postgres.env` |
| Secrets path      | `/etc/iago/secrets/{name}-*`  | `/etc/iago/secrets/postgres-password` |

### Multiple Containers Per Machine (Hybrid Approach)

The system supports running multiple containers on a single machine:

#### Default Behavior: Single Container
```bash
# Machine 'postgres' looks for:
/etc/iago/containers/postgres.env
# Creates service: bootc@postgres.service
```

#### Multiple Containers: Additional Services
```bash
# Add additional container configs:
/etc/iago/containers/postgres.env      # Main database
/etc/iago/containers/redis.env         # Cache service
/etc/iago/containers/backup.env        # Backup service

# Results in services:
bootc@postgres.service
bootc@redis.service
bootc@backup.service
```

#### Creating Additional Container Configs

```bash
# SSH to machine
ssh user@postgres.organmorgan.com

# Create additional container config
sudo tee /etc/iago/containers/redis.env << EOF
CONTAINER_IMAGE=docker.io/library/redis:7-alpine
CONTAINER_NAME=bootc-redis
HEALTH_CHECK_WAIT=15
UPDATE_STRATEGY=latest
EOF

# Start the additional service
sudo systemctl enable --now bootc@redis.service

# Verify both containers running
sudo podman ps
sudo systemctl status bootc@postgres.service
sudo systemctl status bootc@redis.service
```

### Configuration Management

#### Machine-Level Configuration

Set default container image in `machines/{machine}/machine.toml`:

```toml
name = "postgres"
fqdn = "postgres.organmorgan.com"
container_image = "ghcr.io/andreweick/postgres"
container_tag = "latest"
mac_address = "02:05:56:39:1b:21"
```

#### Runtime Overrides

The `/etc/iago/containers/{machine}.env` file can override the built-in configuration:

```bash
# Override registry and tag at runtime
CONTAINER_IMAGE=docker.io/my-private-registry/postgres:v2.0
UPDATE_STRATEGY=pinned
HEALTH_CHECK_WAIT=60
```

#### Update Frequency

Container updates follow the configured schedule in `defaults.toml`:

```toml
[bootc]
update_time = "02:00:00"    # Daily at 2 AM
health_check_wait = 30      # Wait 30s before health check
```

The update process:
1. Checks all `.env` files in `/etc/iago/containers/`
2. Skips containers with `UPDATE_STRATEGY=pinned`
3. Pulls latest images for `UPDATE_STRATEGY=latest`
4. Restarts services and validates health
5. Automatically rolls back failed updates

### Troubleshooting

#### Check Container Status
```bash
# View all bootc services
sudo systemctl list-units 'bootc@*'

# Check specific container
sudo systemctl status bootc@postgres.service
sudo journalctl -u bootc@postgres.service -f

# List running containers
sudo podman ps

# Check container health
sudo podman healthcheck run bootc-postgres
```

#### View Container Configuration
```bash
# Check current container config
cat /etc/iago/containers/postgres.env

# View update logs
sudo journalctl -u bootc-update.service

# Check update timer status
sudo systemctl status bootc-update.timer
```

#### Manual Rollback
```bash
# Use previous image
sudo podman tag ghcr.io/user/postgres:previous ghcr.io/user/postgres:latest
sudo systemctl restart bootc@postgres.service

# Or edit config for specific version
sudo vim /etc/iago/containers/postgres.env
# Set: CONTAINER_IMAGE=ghcr.io/user/postgres:known-good-version
sudo systemctl restart bootc@postgres.service
```

This flexible system allows you to manage containers independently of machine provisioning while maintaining all the safety features of automatic updates and health monitoring.

## Container Lifecycle

Iago manages the complete container lifecycle from build to deployment using a **pure Go approach** that eliminates Docker/Podman dependencies for building.

### 1. Container Building (Pure Go)

Iago uses `github.com/google/go-containerregistry` to build containers without requiring Docker or Podman:

```bash
# Build using iago's pure Go builder (recommended)
./bin/iago build machine-name

# Build all containers
./bin/iago build --all

# Build for local testing (pushes to localhost:5000)
./bin/iago build machine-name --local

# Build without pushing
./bin/iago build machine-name --no-push

# Build with custom tag
./bin/iago build machine-name --tag v1.2.3

# Build and sign with cosign keyless signing
./bin/iago build machine-name --sign
```

**Advantages of Pure Go Building:**
- No Docker/Podman daemon required
- Faster builds in CI/CD environments
- Consistent cross-platform behavior
- Better integration with Go toolchain
- No external runtime dependencies

### 2. Registry Push

Built containers are automatically pushed to your configured registry:

```toml
# config/defaults.toml
[container_registry]
url = "ghcr.io/your-username"
```

The build process:
1. **Reads Containerfile** from `containers/{machine-name}/`
2. **Builds layers** using go-containerregistry (pure Go, no Docker daemon)
3. **Tags image** as `{registry}/{machine-name}:latest`
4. **Pushes to registry** (unless `--no-push` specified)
5. **Optionally signs** with cosign keyless signing

### 3. Machine Deployment

When you generate ignition files, the bootc container workflow is embedded:

```bash
# Generate ignition with embedded container configuration
./bin/iago ignite machine-name
```

The ignition file includes:
- **Systemd service** definition for `bootc-{machine-name}.service`
- **Container pull** configuration from your registry
- **Health check** setup and monitoring
- **Update timer** configuration for daily updates

### 4. Runtime Execution

On the CoreOS machine, the container runs via systemd:

```bash
# Service pulls and runs the container
sudo systemctl start bootc-machine-name.service

# Check service status
sudo systemctl status bootc-machine-name.service

# View container logs
sudo journalctl -u bootc-machine-name.service -f

# List running containers
sudo podman ps

# Check container health
sudo podman healthcheck run bootc-machine-name
```

### 5. Continuous Updates

The machine automatically:
1. **Checks for updates** daily at 02:00 via `bootc-update.timer`
2. **Pulls latest image** from registry if available
3. **Restarts service** with new container
4. **Validates health** after 30 seconds
5. **Rolls back** automatically if health checks fail

### Alternative Building Methods

While pure Go building is recommended, you can also use traditional methods:

```bash
# Traditional podman build (requires podman installed)
cd containers/machine-name/
podman build -t ghcr.io/your-username/machine-name:latest .
podman push ghcr.io/your-username/machine-name:latest

# Using make tasks (legacy)
make build-container-podman NAME=machine-name
make push-container NAME=machine-name
make deploy-container NAME=machine-name  # build + push
```

## Bootc Containers

### Auto-Generated Container Scaffold

When you run `iago init`, it automatically creates a complete container scaffold:

```
containers/[machine-name]/
├── Containerfile           # Bootc container definition
├── scripts/
│   ├── init.sh            # Initialization script
│   ├── health.sh          # Health check script
│   └── hello.sh           # Simple hello world service
├── config/                # Configuration files directory
├── systemd/
│   └── [machine-name]-app.service  # Systemd service
└── tests/                 # Container-specific tests
```

### Container Features

The generated container includes:
- **Base**: Fedora bootc:42 with essential utilities (htop, procps-ng, systemd)
- **Hello World Service**: Simple demonstration service that logs messages every 30 seconds
- **Health Checks**: Built-in health monitoring via systemd service status
- **Application User**: Dedicated non-root user for the application
- **Systemd Integration**: Service files for proper lifecycle management
- **Logging**: Structured logging to systemd journal
- **Template Support**: Uses shared script templates from `containers/_shared/scripts/` when available

### Testing Your Container

```bash
# Build and push using iago (recommended)
./bin/iago build your-machine-name

# Alternative: Build using podman directly
cd containers/your-machine-name
podman build -t ghcr.io/your-username/your-machine-name:latest .

# Test locally (bootc containers use systemd, not typical web servers)
podman run --rm --name test-container ghcr.io/your-username/your-machine-name:latest /usr/local/bin/hello.sh

# Check health script
podman run --rm ghcr.io/your-username/your-machine-name:latest /usr/local/bin/health.sh

# Push to registry (if built with podman)
podman push ghcr.io/your-username/your-machine-name:latest
```

## Machine Lifecycle

1. **Provisioning**: Generate ignition file and boot CoreOS
2. **First Boot**:
   - Apply base configuration from butane templates
   - Generate secrets and passwords automatically
   - Pull and start bootc containers via systemd services
   - Configure network (MAC addresses, DNS, timezone)
   - Set up update timers and monitoring
3. **Runtime Operations**:
   - Containers run via `bootc-{machine-name}.service` systemd services
   - Health checks every 30 seconds with automatic restart on failure
   - Structured logging to systemd journal
   - Secret access via `/etc/iago/secrets/` directory
4. **Automatic Updates**:
   - **Container updates**: Daily at 02:00 via `bootc-update.timer`
   - **CoreOS updates**: Daily at 03:00 via Zincati with periodic reboot strategy
   - **Health validation**: 30-second wait before validating successful updates
   - **Automatic rollback**: Failed updates automatically revert to previous container version
5. **Manual Operations**:
   - Container rollback instructions in `/etc/iago/rollback-instructions.txt`
   - Manual rollback: `podman tag previous latest && systemctl restart service`
   - Pin to specific versions by editing systemd service files

## Security

### Public (stored in Git)
- Machine names and FQDNs
- MAC addresses
- Workload assignments
- Network configuration

### Generated at Build Time
- Password hashes (using SHA-512)
- Cached locally (gitignored)

### Generated at Boot Time
- Application passwords
- TLS certificates
- API keys

### SSH Access
Secrets are accessible on each machine:
```bash
ssh user@machine-fqdn
sudo cat /etc/iago/secrets/postgres-password
```

## Development

### Adding New Workloads

1. **Initialize a new machine**: `iago init my-new-service`
   - This creates the container scaffold in `containers/my-new-service/`
   - Adds machine configuration to `machines/my-new-service/machine.toml`
   - Creates machine-specific butane template `machines/my-new-service/butane.yaml.tmpl`
2. **Customize the container**: Edit files in `containers/my-new-service/`
   - Modify `Containerfile` for your application
   - Update scripts in `scripts/` directory
   - Configure systemd services in `systemd/` directory
3. **Build and push**: Use `iago build my-new-service`
4. **Generate ignition**: Run `iago ignite my-new-service`

### Testing

```bash
make test           # Run unit tests
make dev-full      # Full development workflow (fmt, lint, test, build, validate)
make build         # Build iago binary
make lint          # Run golangci-lint
make fmt           # Format code
```

## Contributing

1. Follow Go best practices from `CLAUDE.md`
2. Use conventional commits
3. Test your changes
4. Update documentation
