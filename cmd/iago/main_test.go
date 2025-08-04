package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestValidateMachineTemplate_Success(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	machineDir := filepath.Join(tempDir, "machines", "test-machine")
	err := os.MkdirAll(machineDir, 0755)
	require.NoError(t, err)

	// Create machine template file with required Go template variables
	templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	templateContent := `variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      groups:
{{ range .User.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .User.PasswordHash }}"
storage:
  files:
    - path: /etc/hostname
      contents:
        inline: "{{ .Machine.Name }}"
{{ if .Machine.MACAddress }}    - path: /etc/NetworkManager/system-connections/eth0.nmconnection
      mode: 0600
      contents:
        inline: |
          [connection]
          id=eth0
          cloned-mac-address={{ .Machine.MACAddress }}
{{ end }}    - path: /etc/registry-config
      contents:
        inline: "{{ .ContainerRegistry.URL }}"
systemd:
  units:
    - name: podman.service
      enabled: true`

	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Change to temp directory to test
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test validation passes
	err = validateBaseButaneTemplate()
	assert.NoError(t, err)
}

func TestValidateBaseButaneTemplate_MissingFile(t *testing.T) {
	// Create temporary directory without machines directory
	tempDir := t.TempDir()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test validation fails for missing machines directory
	err = validateBaseButaneTemplate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "machines directory 'machines' does not exist")
}

func TestValidateBaseButaneTemplate_WithNetworkInterface(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	machineDir := filepath.Join(tempDir, "machines", "test-machine")
	err := os.MkdirAll(machineDir, 0755)
	require.NoError(t, err)

	// Create machine template file with NetworkInterface support
	templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	templateContent := `variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      password_hash: "{{ .User.PasswordHash }}"
storage:
  files:
    - path: /etc/hostname
      contents:
        inline: "{{ .Machine.Name }}"
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
{{ end }}    - path: /etc/registry-config
      contents:
        inline: "{{ .ContainerRegistry.URL }}"
systemd:
  units:
    - name: podman.service
      enabled: true`

	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	// Test validation
	err = validateBaseButaneTemplate()
	assert.NoError(t, err, "Template with NetworkInterface support should validate successfully")
}

func TestValidateBaseButaneTemplate_MissingConstant(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	machinesDir := filepath.Join(tempDir, "machines")
	err := os.MkdirAll(machinesDir, 0755)
	require.NoError(t, err)

	// Create machine template file WITHOUT required constant
	machineDir := filepath.Join(machinesDir, "test-machine")
	require.NoError(t, os.MkdirAll(machineDir, 0755))
	templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	templateContent := `variant: fcos
version: 1.5.0
storage:
  files:
    - path: /etc/hostname
      contents:
        inline: "{{ .Machine.Name }}"
    - path: /usr/local/bin/script.sh
      mode: 0755`

	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test validation fails for missing template variables
	err = validateBaseButaneTemplate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required template variable '{{ .User.Username }}' not found")
}

func TestListCommand_OutputFormat(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	machinesDir := filepath.Join(tempDir, "machines")
	err := os.MkdirAll(configDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(machinesDir, 0755)
	require.NoError(t, err)

	// Create machine configurations using new structure
	machines := []struct {
		name string
		fqdn string
		mac  string
		intf string
		tag  string
	}{
		{"proxmox-vm", "proxmox-vm.example.com", "02:05:56:12:34:56", "ens18", "latest"},
		{"bare-metal", "bare-metal.example.com", "02:05:56:78:90:12", "enp0s31f6", "latest"},
		{"default-interface", "default.example.com", "02:05:56:34:56:78", "", "latest"},
		{"no-mac", "no-mac.example.com", "", "", "latest"},
	}

	// Create machine directories and configs
	for _, machine := range machines {
		machineDir := filepath.Join(machinesDir, machine.name)
		require.NoError(t, os.MkdirAll(machineDir, 0755))

		// Create machine.toml
		config := fmt.Sprintf(`name = "%s"
fqdn = "%s"
container_image = "registry.example.com/%s"
container_tag = "%s"`, machine.name, machine.fqdn, machine.name, machine.tag)
		if machine.mac != "" {
			config += fmt.Sprintf(`
mac_address = "%s"`, machine.mac)
		}
		if machine.intf != "" {
			config += fmt.Sprintf(`
network_interface = "%s"`, machine.intf)
		}

		machineFile := filepath.Join(machineDir, "machine.toml")
		require.NoError(t, os.WriteFile(machineFile, []byte(config), 0644))
	}

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	// Create a mock CLI context
	app := &cli.App{}
	ctx := cli.NewContext(app, nil, nil)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run list command
	err = listCommand(ctx)
	assert.NoError(t, err)

	// Restore stdout and read output
	w.Close()
	os.Stdout = oldStdout
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	outputStr := string(output[:n])

	// Verify output format and content
	assert.Contains(t, outputStr, "NAME", "Should contain NAME header")
	assert.Contains(t, outputStr, "FQDN", "Should contain FQDN header")
	assert.Contains(t, outputStr, "MAC ADDRESS", "Should contain MAC ADDRESS header")
	assert.Contains(t, outputStr, "INTERFACE", "Should contain INTERFACE header")
	assert.Contains(t, outputStr, "proxmox-vm", "Should contain proxmox-vm entry")
	assert.Contains(t, outputStr, "ens18", "Should contain ens18 interface")
	assert.Contains(t, outputStr, "enp0s31f6", "Should contain enp0s31f6 interface")
	assert.Contains(t, outputStr, "02:05:56:12:34:56", "Should contain MAC address")

	// Check for dash placeholders where values are missing
	lines := strings.Split(outputStr, "\n")
	var dataLines []string
	for _, line := range lines {
		if strings.Contains(line, "no-mac") || strings.Contains(line, "default-interface") {
			dataLines = append(dataLines, line)
		}
	}

	// Verify dash placeholders for missing values
	for _, line := range dataLines {
		if strings.Contains(line, "no-mac") {
			// Should have dashes for both MAC and interface
			assert.Regexp(t, `no-mac\s+.*\s+-\s+-`, line, "no-mac should have dashes for MAC and interface")
		} else if strings.Contains(line, "default-interface") {
			// Should have MAC but dash for interface
			assert.Regexp(t, `default-interface\s+.*\s+02:05:56:34:56:78\s+-`, line, "default-interface should have dash for interface")
		}
	}
}
