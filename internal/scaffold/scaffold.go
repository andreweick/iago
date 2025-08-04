package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andreweick/iago/internal/machine"
)

type ScaffoldOptions struct {
	MachineName string
	FQDN        string
	MACAddress  string
	OutputDir   string
}

type Scaffolder struct {
	defaults machine.Defaults
}

func NewScaffolder(defaults machine.Defaults) *Scaffolder {
	return &Scaffolder{
		defaults: defaults,
	}
}

// copyAndCustomizeTemplate copies a template file and replaces placeholders
func (s *Scaffolder) copyAndCustomizeTemplate(templatePath, targetPath, serviceName string) error {
	// Read template file
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Replace {SERVICE} placeholder with actual service name
	customized := strings.ReplaceAll(string(content), "{SERVICE}", serviceName)

	// Write customized content to target
	return os.WriteFile(targetPath, []byte(customized), 0755)
}

// fileExists checks if a file exists
func (s *Scaffolder) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (s *Scaffolder) CreateMachineScaffold(opts ScaffoldOptions) error {
	// Create container directory structure
	if err := s.createContainerStructure(opts); err != nil {
		return fmt.Errorf("failed to create container structure: %w", err)
	}

	// Create machine config entry
	if err := s.addMachineToConfig(opts); err != nil {
		return fmt.Errorf("failed to add machine to config: %w", err)
	}

	// Generate ignition file
	if err := s.generateIgnition(opts); err != nil {
		return fmt.Errorf("failed to generate ignition: %w", err)
	}

	return nil
}

func (s *Scaffolder) createContainerStructure(opts ScaffoldOptions) error {
	// Create container in containers directory
	containerDir := filepath.Join("containers", opts.MachineName)
	return s.createContainerFiles(containerDir, opts)
}

func (s *Scaffolder) createContainerFiles(containerDir string, opts ScaffoldOptions) error {
	// Create directories
	dirs := []string{
		containerDir,
		filepath.Join(containerDir, "scripts"),
		filepath.Join(containerDir, "config"),
		filepath.Join(containerDir, "systemd"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create Containerfile
	if err := s.createContainerfile(containerDir, opts); err != nil {
		return fmt.Errorf("failed to create Containerfile: %w", err)
	}

	return nil
}

func (s *Scaffolder) createContainerfile(containerDir string, opts ScaffoldOptions) error {
	containerfile := "FROM quay.io/fedora/fedora-bootc:42\n"

	return os.WriteFile(filepath.Join(containerDir, "Containerfile"), []byte(containerfile), 0644)
}

func (s *Scaffolder) createScripts(containerDir string, opts ScaffoldOptions) error {
	scriptsDir := filepath.Join(containerDir, "scripts")

	// Use shared templates if available, otherwise create basic scripts
	sharedScriptsDir := "containers/_shared/scripts"

	// Copy and customize init script from shared template
	initTemplatePath := filepath.Join(sharedScriptsDir, "basic-init-template.sh")
	initTargetPath := filepath.Join(scriptsDir, "init.sh")

	if s.fileExists(initTemplatePath) {
		if err := s.copyAndCustomizeTemplate(initTemplatePath, initTargetPath, opts.MachineName); err != nil {
			return fmt.Errorf("failed to copy init template: %w", err)
		}
	} else {
		// Fallback to hardcoded script if template doesn't exist
		initScript := fmt.Sprintf(`#!/bin/bash
# Init script for %s
set -euo pipefail

echo "[$(date)] Initializing %s container..."

# Create required directories
mkdir -p /var/lib/%s /var/log/%s

# Set proper permissions
chown -R %s-app:%s-app /var/lib/%s /var/log/%s

echo "[$(date)] %s container initialized successfully"
`, opts.MachineName, opts.MachineName, opts.MachineName,
			opts.MachineName, opts.MachineName, opts.MachineName,
			opts.MachineName, opts.MachineName, opts.MachineName)

		if err := os.WriteFile(initTargetPath, []byte(initScript), 0755); err != nil {
			return err
		}
	}

	// Copy and customize health check script from shared template
	healthTemplatePath := filepath.Join(sharedScriptsDir, "health-check-template.sh")
	healthTargetPath := filepath.Join(scriptsDir, "health.sh")

	if s.fileExists(healthTemplatePath) {
		if err := s.copyAndCustomizeTemplate(healthTemplatePath, healthTargetPath, opts.MachineName); err != nil {
			return fmt.Errorf("failed to copy health template: %w", err)
		}
	} else {
		// Fallback to hardcoded script if template doesn't exist
		healthScript := fmt.Sprintf(`#!/bin/bash
# Health check script for %s
set -euo pipefail

# Simple health check - verify service is running
if systemctl is-active --quiet %s-app.service; then
    echo "[$(date)] %s service is healthy"
    exit 0
else
    echo "[$(date)] %s service is not running"
    exit 1
fi
`, opts.MachineName, opts.MachineName, opts.MachineName, opts.MachineName)

		if err := os.WriteFile(healthTargetPath, []byte(healthScript), 0755); err != nil {
			return err
		}
	}

	// Create a simple hello world script (always custom, no shared template needed)
	helloScript := fmt.Sprintf(`#!/bin/bash
# Simple hello world service for %s
set -euo pipefail

echo "[$(date)] Starting hello world service for %s"

# Log hello messages every 30 seconds
while true; do
    echo "[$(date)] Hello from %s! Service is running..."
    sleep 30
done
`, opts.MachineName, opts.MachineName, opts.MachineName)

	return os.WriteFile(filepath.Join(scriptsDir, "hello.sh"), []byte(helloScript), 0755)
}

func (s *Scaffolder) createSystemdService(containerDir string, opts ScaffoldOptions) error {
	systemdDir := filepath.Join(containerDir, "systemd")

	serviceFile := fmt.Sprintf(`[Unit]
Description=%s Hello World Service
After=multi-user.target

[Service]
Type=simple
User=%s-app
Group=%s-app
WorkingDirectory=/var/lib/%s
ExecStartPre=/usr/local/bin/init.sh
ExecStart=/usr/local/bin/hello.sh
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, opts.MachineName, opts.MachineName, opts.MachineName, opts.MachineName)

	return os.WriteFile(filepath.Join(systemdDir, opts.MachineName+"-app.service"), []byte(serviceFile), 0644)
}

func (s *Scaffolder) addMachineToConfig(opts ScaffoldOptions) error {
	// Create machine config in machines directory
	machineDir := filepath.Join("machines", opts.MachineName)
	return s.createMachineConfig(machineDir, opts)
}

func (s *Scaffolder) createMachineConfig(machineDir string, opts ScaffoldOptions) error {
	// Ensure machine directory exists
	if err := os.MkdirAll(machineDir, 0755); err != nil {
		return fmt.Errorf("failed to create machine directory: %w", err)
	}

	// Create unified machine.toml
	machineContent := fmt.Sprintf(`name = "%s"
fqdn = "%s"
container_image = "%s/%s"
container_tag = "latest"`, opts.MachineName, opts.FQDN, s.defaults.ContainerRegistry.URL, opts.MachineName)

	if opts.MACAddress != "" {
		machineContent += fmt.Sprintf(`
mac_address = "%s"`, opts.MACAddress)
	}

	machinePath := filepath.Join(machineDir, "machine.toml")
	if err := os.WriteFile(machinePath, []byte(machineContent), 0644); err != nil {
		return fmt.Errorf("failed to write machine.toml: %w", err)
	}

	// Create scaffold butane.yaml
	if err := s.createMachineButaneScaffold(machineDir, opts); err != nil {
		return fmt.Errorf("failed to create butane.yaml scaffold: %w", err)
	}

	return nil
}

func (s *Scaffolder) addToOldMachinesConfig(opts ScaffoldOptions) error {
	configEntry := fmt.Sprintf(`
# Added by iago init
[[machines]]
name = "%s"
fqdn = "%s"`, opts.MachineName, opts.FQDN)

	if opts.MACAddress != "" {
		configEntry += fmt.Sprintf(`
mac_address = "%s"`, opts.MACAddress)
	}

	configEntry += "\n"

	// Append to machines.toml
	f, err := os.OpenFile("config/machines.toml", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(configEntry)
	return err
}

func (s *Scaffolder) createMachineButaneScaffold(machineDir string, opts ScaffoldOptions) error {
	// Create a complete butane template file for the machine
	scaffoldContent := fmt.Sprintf(`variant: fcos
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
    - path: /var/log/iago
      mode: "0755"
    - path: /var/lib/{{ .Machine.Name }}
      mode: "0755"
    - path: /var/log/{{ .Machine.Name }}
      mode: "0755"
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: "{{ .Machine.Name }}"
    - path: /etc/iago/machine-info
      mode: 0644
      contents:
        inline: |
          MACHINE_NAME={{ .Machine.Name }}
          FQDN={{ .Machine.FQDN }}
    # Container configuration (mutable)
    - path: /etc/iago/containers/{{ .Machine.Name }}.env
      mode: 0644
      contents:
        inline: |
          CONTAINER_IMAGE={{ .Machine.ContainerImage }}:{{ .Machine.ContainerTag }}
          CONTAINER_NAME=bootc-{{ .Machine.Name }}
          HEALTH_CHECK_WAIT=30
          UPDATE_STRATEGY=latest
    # Generated secrets
    - path: /etc/iago/secrets/%s-password
      mode: 0600
      contents:
        inline: {{ .GeneratedSecrets.Password }}
    - path: /etc/iago/rollback-instructions.txt
      mode: 0644
      contents:
        inline: |
          BOOTC ROLLBACK INSTRUCTIONS
          ==========================

          To rollback to previous bootc image:
          1. sudo systemctl stop bootc@{{ .Machine.Name }}
          2. sudo podman tag {{ .Machine.ContainerImage }}:previous {{ .Machine.ContainerImage }}:{{ .Machine.ContainerTag }}
          3. sudo systemctl start bootc@{{ .Machine.Name }}

          To view available images:
          sudo podman images | grep {{ .Machine.ContainerImage }}

          To pin to specific version:
          1. Edit /etc/iago/containers/{{ .Machine.Name }}.env
          2. Change CONTAINER_IMAGE to specific tag
          3. Change UPDATE_STRATEGY to pinned
          4. sudo systemctl restart bootc@{{ .Machine.Name }}

          To switch to different container entirely:
          1. Edit /etc/iago/containers/{{ .Machine.Name }}.env
          2. Update CONTAINER_IMAGE to new image:tag
          3. sudo systemctl restart bootc@{{ .Machine.Name }}
    # Management scripts
    - path: /usr/local/bin/bootc-manager.sh
      mode: 0755
      contents:
        local: bootc-manager.sh
    - path: /usr/local/bin/bootc-run.sh
      mode: 0755
      contents:
        local: bootc-run.sh
    - path: /usr/local/bin/bootc-update.sh
      mode: 0755
      contents:
        local: bootc-update.sh
    - path: /usr/local/bin/motd.sh
      mode: 0755
      contents:
        local: motd.sh
    - path: /etc/profile.d/motd.sh
      mode: 0644
      contents:
        inline: |
          # Run custom MOTD on login
          /usr/local/bin/motd.sh
{{ if .Machine.MACAddress }}    - path: /etc/NetworkManager/system-connections/{{ default .Network.DefaultNetworkInterface .Machine.NetworkInterface }}.nmconnection
      mode: 0600
      contents:
        inline: |
          [connection]
          id={{ default .Network.DefaultNetworkInterface .Machine.NetworkInterface }}
          type=ethernet
          interface-name={{ default .Network.DefaultNetworkInterface .Machine.NetworkInterface }}
          
          [ethernet]
          cloned-mac-address={{ .Machine.MACAddress }}
          
          [ipv4]
          method=auto
{{ end }}

systemd:
  units:
    # Set timezone
    - name: set-timezone.service
      enabled: true
      contents: |
        [Unit]
        Description=Set system timezone
        Before=multi-user.target
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/timedatectl set-timezone {{ .Network.Timezone }}
        RemainAfterExit=true
        [Install]
        WantedBy=multi-user.target

    # CoreOS auto-updates
    - name: zincati.service
      dropins:
        - name: 55-update-strategy.conf
          contents: |
            [Service]
            Environment="ZINCATI_STRATEGY={{ .Updates.Strategy }}"
            Environment="ZINCATI_PERIODIC_TIME={{ .Updates.RebootTime }}"
            Environment="ZINCATI_STREAM={{ .Updates.Stream }}"

    # Enable Podman
    - name: podman.service
      enabled: true

    # Generic bootc template (handles any container)
    - name: bootc@.service
      contents: |
        [Unit]
        Description=Bootc Container %%i
        After=network-online.target podman.service
        Wants=network-online.target
        Requires=podman.service
        
        [Service]
        Type=notify
        NotifyAccess=all
        Restart=always
        RestartSec=30
        TimeoutStartSec=300
        EnvironmentFile=/etc/iago/containers/%%i.env
        ExecStart=/usr/local/bin/bootc-run.sh %%i
        ExecStop=/usr/bin/podman stop -t 30 bootc-%%i
        
        [Install]
        WantedBy=multi-user.target

    # Container manager (auto-starts containers based on config files)
    - name: bootc-manager.service
      enabled: true
      contents: |
        [Unit]
        Description=Bootc Container Manager
        After=multi-user.target
        
        [Service]
        Type=oneshot
        ExecStart=/usr/local/bin/bootc-manager.sh
        RemainAfterExit=true
        
        [Install]
        WantedBy=multi-user.target

    # Container update timer
    - name: bootc-update.timer
      enabled: true
      contents: |
        [Unit]
        Description=Daily bootc container update check
        [Timer]
        OnCalendar=*-*-* {{ .Bootc.UpdateTime }}
        Persistent=true
        [Install]
        WantedBy=timers.target

    # Container update service
    - name: bootc-update.service
      contents: |
        [Unit]
        Description=Update bootc containers
        After=network-online.target
        Wants=network-online.target
        [Service]
        Type=oneshot
        ExecStart=/usr/local/bin/bootc-update.sh
        StandardOutput=journal
        StandardError=journal`, opts.MachineName)

	machineButanePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	return os.WriteFile(machineButanePath, []byte(scaffoldContent), 0644)
}

func (s *Scaffolder) generateIgnition(opts ScaffoldOptions) error {
	// The actual ignition generation is handled by the main init command
	// This method is kept for compatibility but doesn't need to do anything
	return nil
}
