# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Iago is a Go CLI application for managing Fedora CoreOS machines with bootc containers. It creates immutable infrastructure where machines are defined by container images and automatically managed through ignition files.

### Core Architecture

- **Fedora CoreOS**: Base operating system with container-centric approach
- **Bootc Containers**: Application workloads packaged as bootc containers
- **Ignition**: Machine provisioning and configuration system
- **Template System**: Go templates for butane configuration files
- **Pure Go Building**: Container building using go-containerregistry (no Docker/Podman dependency)

### Key Components

- `cmd/iago/`: CLI application entry point using urfave/cli
- `internal/machine/`: Machine configuration management and loading
- `internal/container/`: Pure Go container building with go-containerregistry
- `internal/butane/`: Template rendering for butane configuration
- `internal/build/`: Ignition file generation
- `internal/scaffold/`: Auto-scaffolding for new machines
- `machines/`: Per-machine configuration and butane templates
- `containers/`: Container definitions (Containerfile, scripts, configs)
- `config/`: Global defaults and embedded scripts

## Common Development Tasks

### Building and Testing

```bash
# Build binary
just build

# Run tests with verbose output
just test

# Full development workflow (format, lint, test, build, validate)
just dev-full

# Run linting
just lint

# Format code
just fmt

# Validate configuration
just validate
```

### Container Operations

```bash
# Build single container (pure Go, no Docker/Podman needed)
just iago container build machine-name

# Build all containers
just build-all-containers

# Build with cosign signing
just build-container-signed
```

### Machine Management

```bash
# Initialize new machine (creates config, container scaffold, ignition)
just iago init machine-name

# Generate ignition file for existing machine
just iago ignite machine-name

# List all configured machines
just iago list

# Remove machine and all files
just iago rm machine-name
```

### Release Management

```bash
# Generate development snapshot with goreleaser
just snapshot

# Create tagged release with goreleaser
just release
```

## Architecture Details

### Template System

Each machine has its own complete butane template at `machines/{name}/butane.yaml.tmpl`. Templates use Go template syntax with variables from:

- `defaults.toml`: Global configuration (users, network, registry)
- `machine.toml`: Machine-specific settings (name, FQDN, MAC, container image)
- Generated secrets and SSH keys

### Container Lifecycle

1. **Build**: Pure Go building with `github.com/google/go-containerregistry`
2. **Registry Push**: Automatic push to configured registry (ghcr.io, localhost:5000, etc.)
3. **Machine Deployment**: Ignition file includes container configuration
4. **Runtime**: Systemd service manages container via podman
5. **Updates**: Daily automatic updates with health checks and rollback

### Workload System

Plugin-based workloads implement the `Workload` interface to contribute:
- Butane configuration overlays
- Secret definitions (auto-generated passwords, keys)
- Validation rules

## Key Files and Directories

- `justfile`: Task automation (preferred over Makefile)
- `go.mod`: Go 1.24.5 with key dependencies: urfave/cli, go-containerregistry, butane
- `config/defaults.toml`: System-wide configuration (users, registry, update settings)
- `config/scripts/`: Embedded scripts (bootc-update.sh, motd.sh)
- `machines/{name}/machine.toml`: Machine configuration
- `machines/{name}/butane.yaml.tmpl`: Complete machine butane template
- `containers/{name}/`: Container definitions with Containerfile and scripts
- `output/ignition/`: Generated ignition files (gitignored)

## Testing

- Unit tests: `*_test.go` files colocated with source
- Integration tests: `internal/integration_test.go`
- Test machine configurations in `machines/` directory
- Container building tests use local registry validation

## Authentication and Registry

- Supports GitHub Container Registry (ghcr.io) and Docker Hub
- Authentication via environment variables, 1Password, or CLI flags
- Pure Go container building eliminates Docker daemon dependency
- Local registry support for testing (localhost:5000)

## Development Notes

- Use `just` for task automation instead of make
- Follow standard Go project layout with `internal/` and `pkg/`
- Container images based on `quay.io/fedora/fedora-bootc:42`
- MAC address generation for homelab DHCP reservations
- Auto-generated secrets and passwords embedded in ignition
- Flexible container management with runtime configuration updates