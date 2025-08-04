package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andreweick/iago/internal/build"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndToEndWithYamlFiles(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create workloads directory for new structure
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "machines"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "containers"), 0755))

	// Use a working minimal template that matches what works
	machineTemplate := `variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      groups:
{{ range .User.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .User.PasswordHash }}"
    - name: "{{ .Admin.Username }}"
      groups:
{{ range .Admin.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .Admin.PasswordHash }}"
storage:
  directories:
    - path: /etc/iago
      mode: "0755"
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: "{{ .Machine.Name }}"
systemd:
  units:
    - name: podman.service
      enabled: true
    - name: set-timezone.service
      enabled: true
      contents: |
        [Unit]
        Description=Set system timezone
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/timedatectl set-timezone {{ .Network.Timezone }}
        [Install]
        WantedBy=multi-user.target`

	// Create the machine directory, template, and config
	machineDir := filepath.Join(tempDir, "machines", "integration-test")
	require.NoError(t, os.MkdirAll(machineDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(machineDir, "butane.yaml.tmpl"), []byte(machineTemplate), 0644))

	// Create machine.toml
	machineConfig := `name = "integration-test"
fqdn = "integration-test.example.com"
container_image = "registry.example.com/integration-test"
container_tag = "latest"`
	require.NoError(t, os.WriteFile(filepath.Join(machineDir, "machine.toml"), []byte(machineConfig), 0644))

	// Create defaults.toml
	defaultsContent := `[user]
username = "testuser"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[admin]
username = "admin"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[network]
timezone = "America/New_York"
default_network_interface = "eth0"

[updates]
stream = "stable"
strategy = "periodic"
period = "daily"
reboot_time = "03:00"

[bootc]
update_time = "02:00:00"
health_check_wait = 30

[container_registry]
url = "registry.example.com"`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "defaults.toml"), []byte(defaultsContent), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Step 1: Build ignition file directly (we already created the machine setup manually)
	builder, err := build.NewBuilder()
	require.NoError(t, err)

	outputFile := filepath.Join(outputDir, "integration-test.ign")
	err = builder.GenerateMachine("integration-test", outputFile)
	require.NoError(t, err)

	// Step 2: Verify build creates .yaml debug files
	debugFile := filepath.Join(outputDir, "integration-test-final-butane.yaml")
	_, err = os.Stat(debugFile)
	assert.NoError(t, err, "Build should create .yaml debug file")

	// Verify no .yml debug files exist
	oldDebugFile := filepath.Join(outputDir, "integration-test-final-butane.yml")
	_, err = os.Stat(oldDebugFile)
	assert.True(t, os.IsNotExist(err), "Build should not create .yml debug file")

	// Step 3: Verify ignition file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Ignition file should be created")

	// Step 4: Verify file content integrity
	debugContent, err := os.ReadFile(debugFile)
	require.NoError(t, err)
	debugStr := string(debugContent)
	assert.Contains(t, debugStr, "variant: fcos")
	assert.Contains(t, debugStr, "integration-test")
}

func TestFileExtensionConsistency(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "machines"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "containers"), 0755))

	// Use a working minimal template that matches what works
	machineTemplate := `variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      groups:
{{ range .User.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .User.PasswordHash }}"
    - name: "{{ .Admin.Username }}"
      groups:
{{ range .Admin.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .Admin.PasswordHash }}"
storage:
  directories:
    - path: /etc/iago
      mode: "0755"
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: "{{ .Machine.Name }}"
systemd:
  units:
    - name: podman.service
      enabled: true
    - name: set-timezone.service
      enabled: true
      contents: |
        [Unit]
        Description=Set system timezone
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/timedatectl set-timezone {{ .Network.Timezone }}
        [Install]
        WantedBy=multi-user.target`

	// Create defaults.toml
	defaultsContent := `[user]
username = "testuser"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[admin]
username = "admin"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[network]
timezone = "America/New_York"
default_network_interface = "eth0"

[updates]
stream = "stable"
strategy = "periodic"
period = "daily"
reboot_time = "03:00"

[bootc]
update_time = "02:00:00"
health_check_wait = 30

[container_registry]
url = "registry.example.com"`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "defaults.toml"), []byte(defaultsContent), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	machineNames := []string{"consist-01", "consist-02", "consist-03"}

	// Create machine templates for all machines
	for _, machineName := range machineNames {
		machineDir := filepath.Join(tempDir, "machines", machineName)
		require.NoError(t, os.MkdirAll(machineDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(machineDir, "butane.yaml.tmpl"), []byte(machineTemplate), 0644))

		// Create machine.toml
		machineConfig := fmt.Sprintf(`name = "%s"
fqdn = "%s.example.com"
container_image = "registry.example.com/%s"
container_tag = "latest"`, machineName, machineName, machineName)
		require.NoError(t, os.WriteFile(filepath.Join(machineDir, "machine.toml"), []byte(machineConfig), 0644))
	}

	// Build all machines
	builder, err := build.NewBuilder()
	require.NoError(t, err)

	for _, machineName := range machineNames {
		outputFile := filepath.Join(outputDir, machineName+".ign")
		err = builder.GenerateMachine(machineName, outputFile)
		require.NoError(t, err)
	}

	// Check that all butane-related files use .yaml extension
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		fileName := info.Name()

		// Check for .yml files (should not exist)
		if strings.HasSuffix(fileName, ".yml") {
			// Allow only this test file itself
			if !strings.Contains(path, "integration_test.go") {
				t.Errorf("Found .yml file: %s (should be .yaml)", path)
			}
		}

		// Verify all butane files use .yaml extension (but skip templates)
		if strings.Contains(fileName, "butane") && !strings.HasSuffix(fileName, ".tmpl") {
			assert.True(t, strings.HasSuffix(fileName, ".yaml"), "Butane file should have .yaml extension: %s", path)
		}

		return nil
	})
	require.NoError(t, err)
}

func TestNoYmlFilesInOutput(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "machines"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "containers"), 0755))

	// Use a working minimal template that matches what works
	machineTemplate := `variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      groups:
{{ range .User.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .User.PasswordHash }}"
    - name: "{{ .Admin.Username }}"
      groups:
{{ range .Admin.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .Admin.PasswordHash }}"
storage:
  directories:
    - path: /etc/iago
      mode: "0755"
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: "{{ .Machine.Name }}"
systemd:
  units:
    - name: podman.service
      enabled: true
    - name: set-timezone.service
      enabled: true
      contents: |
        [Unit]
        Description=Set system timezone
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/timedatectl set-timezone {{ .Network.Timezone }}
        [Install]
        WantedBy=multi-user.target`

	// Create defaults.toml
	defaultsContent := `[user]
username = "testuser"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[admin]
username = "admin"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[network]
timezone = "America/New_York"
default_network_interface = "eth0"

[updates]
stream = "stable"
strategy = "periodic"
period = "daily"
reboot_time = "03:00"

[bootc]
update_time = "02:00:00"
health_check_wait = 30

[container_registry]
url = "registry.example.com"`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "defaults.toml"), []byte(defaultsContent), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create machine template directory and file
	machineDir := filepath.Join(tempDir, "machines", "no-yml-test")
	require.NoError(t, os.MkdirAll(machineDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(machineDir, "butane.yaml.tmpl"), []byte(machineTemplate), 0644))

	// Create machine.toml
	machineConfig := `name = "no-yml-test"
fqdn = "no-yml-test.example.com"
container_image = "registry.example.com/no-yml-test"
container_tag = "latest"`
	require.NoError(t, os.WriteFile(filepath.Join(machineDir, "machine.toml"), []byte(machineConfig), 0644))

	// Build machine directly (already created manually)

	builder, err := build.NewBuilder()
	require.NoError(t, err)

	outputFile := filepath.Join(outputDir, "no-yml-test.ign")
	err = builder.GenerateMachine("no-yml-test", outputFile)
	require.NoError(t, err)

	// Use filepath.Glob to verify no .yml files exist in output
	ymlFiles, err := filepath.Glob(filepath.Join(outputDir, "*.yml"))
	require.NoError(t, err)
	assert.Empty(t, ymlFiles, "No .yml files should exist in output directory")

	// Verify .yaml files do exist
	yamlFiles, err := filepath.Glob(filepath.Join(outputDir, "*.yaml"))
	require.NoError(t, err)
	assert.NotEmpty(t, yamlFiles, "Should have .yaml files in output directory")
}

func TestYamlFilesAreCreated(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "machines"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "containers"), 0755))

	// Use a working minimal template that matches what works
	machineTemplate := `variant: fcos
version: 1.5.0
passwd:
  users:
    - name: "{{ .User.Username }}"
      groups:
{{ range .User.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .User.PasswordHash }}"
    - name: "{{ .Admin.Username }}"
      groups:
{{ range .Admin.Groups }}        - {{ . }}
{{ end }}
      password_hash: "{{ .Admin.PasswordHash }}"
storage:
  directories:
    - path: /etc/iago
      mode: "0755"
  files:
    - path: /etc/hostname
      mode: 0644
      contents:
        inline: "{{ .Machine.Name }}"
systemd:
  units:
    - name: podman.service
      enabled: true
    - name: set-timezone.service
      enabled: true
      contents: |
        [Unit]
        Description=Set system timezone
        [Service]
        Type=oneshot
        ExecStart=/usr/bin/timedatectl set-timezone {{ .Network.Timezone }}
        [Install]
        WantedBy=multi-user.target`

	// Create defaults.toml
	defaultsContent := `[user]
username = "testuser"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[admin]
username = "admin"
groups = ["sudo", "wheel"]
password_hash = "$6$test$hash"

[network]
timezone = "America/New_York"
default_network_interface = "eth0"

[updates]
stream = "stable"
strategy = "periodic"
period = "daily"
reboot_time = "03:00"

[bootc]
update_time = "02:00:00"
health_check_wait = 30

[container_registry]
url = "registry.example.com"`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "defaults.toml"), []byte(defaultsContent), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create machine template directory and file
	machineDir := filepath.Join(tempDir, "machines", "yaml-created-test")
	require.NoError(t, os.MkdirAll(machineDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(machineDir, "butane.yaml.tmpl"), []byte(machineTemplate), 0644))

	// Create machine.toml
	machineConfig := `name = "yaml-created-test"
fqdn = "yaml-created-test.example.com"
container_image = "registry.example.com/yaml-created-test"
container_tag = "latest"`
	require.NoError(t, os.WriteFile(filepath.Join(machineDir, "machine.toml"), []byte(machineConfig), 0644))

	// Build machine directly (already created manually)

	builder, err := build.NewBuilder()
	require.NoError(t, err)

	outputFile := filepath.Join(outputDir, "yaml-created-test.ign")
	err = builder.GenerateMachine("yaml-created-test", outputFile)
	require.NoError(t, err)

	// Verify expected .yaml files are created
	expectedFiles := []string{
		filepath.Join(outputDir, "yaml-created-test-final-butane.yaml"),
	}

	for _, expectedFile := range expectedFiles {
		_, err = os.Stat(expectedFile)
		assert.NoError(t, err, "Expected .yaml file should be created: %s", expectedFile)

		// Verify file content is valid
		content, err := os.ReadFile(expectedFile)
		require.NoError(t, err)
		contentStr := string(content)
		assert.NotEmpty(t, contentStr, "YAML file should have content: %s", expectedFile)
	}

	// Use filepath.Glob to verify expected .yaml files pattern
	yamlFiles, err := filepath.Glob(filepath.Join(outputDir, "*-final-butane.yaml"))
	require.NoError(t, err)
	assert.Len(t, yamlFiles, 1, "Should have exactly one final butane .yaml file")
	assert.Contains(t, yamlFiles[0], "yaml-created-test-final-butane.yaml")
}
