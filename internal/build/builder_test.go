package build

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createDefaultsToml creates a defaults.toml file for testing
func createDefaultsToml(t *testing.T, configDir string) {
	defaultsContent := `[user]
username = "testuser"
password_hash = "$6$test$hash"
groups = ["sudo", "wheel"]

[admin]
username = "admin"
password_hash = "$6$admin$hash"
groups = ["sudo"]

[network]
timezone = "UTC"

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
}

// createMachineStructure creates the new machine directory structure for testing
func createMachineStructure(t *testing.T, tempDir, machineName, fqdn string) {
	// Create machines directory
	machineDir := filepath.Join(tempDir, "machines", machineName)
	require.NoError(t, os.MkdirAll(machineDir, 0755))

	// Create minimal butane template
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

	templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	require.NoError(t, os.WriteFile(templatePath, []byte(machineTemplate), 0644))

	// Create machine.toml
	machineConfig := fmt.Sprintf(`name = "%s"
fqdn = "%s"
container_image = "registry.example.com/%s"
container_tag = "latest"`, machineName, fqdn, machineName)
	machineConfigPath := filepath.Join(machineDir, "machine.toml")
	require.NoError(t, os.WriteFile(machineConfigPath, []byte(machineConfig), 0644))
}

func TestDebugFileHasYamlExtension(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create defaults.toml
	createDefaultsToml(t, configDir)

	// Create machine structure using new format
	createMachineStructure(t, tempDir, "test-machine", "test-machine.example.com")

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create builder
	builder, err := NewBuilder()
	require.NoError(t, err)

	// Generate machine ignition
	outputFile := filepath.Join(outputDir, "test-machine.ign")
	err = builder.GenerateMachine("test-machine", outputFile)
	require.NoError(t, err)

	// Verify debug file was created with .yaml extension
	debugFile := filepath.Join(outputDir, "test-machine-final-butane.yaml")
	_, err = os.Stat(debugFile)
	assert.NoError(t, err, "Debug file should be created with .yaml extension")

	// Verify no .yml debug file exists
	oldDebugFile := filepath.Join(outputDir, "test-machine-final-butane.yml")
	_, err = os.Stat(oldDebugFile)
	assert.True(t, os.IsNotExist(err), "Should not create .yml debug file")

	// Verify debug file content
	content, err := os.ReadFile(debugFile)
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "variant: fcos")
	assert.Contains(t, contentStr, "test-machine")
}

func TestBuilderPathConstruction(t *testing.T) {
	tests := []struct {
		name        string
		machineName string
		expected    string
	}{
		{
			name:        "simple machine name",
			machineName: "web-server",
			expected:    "web-server-final-butane.yaml",
		},
		{
			name:        "machine with numbers",
			machineName: "db-01",
			expected:    "db-01-final-butane.yaml",
		},
		{
			name:        "complex machine name",
			machineName: "postgres-primary-cluster",
			expected:    "postgres-primary-cluster-final-butane.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := t.TempDir()
			configDir := filepath.Join(tempDir, "config")
			outputDir := filepath.Join(tempDir, "output")
			require.NoError(t, os.MkdirAll(configDir, 0755))
			require.NoError(t, os.MkdirAll(outputDir, 0755))

			// Create defaults.toml
			createDefaultsToml(t, configDir)

			// Create machine structure using new format
			createMachineStructure(t, tempDir, tt.machineName, tt.machineName+".example.com")

			// Change to temp directory
			oldWd, err := os.Getwd()
			require.NoError(t, err)
			require.NoError(t, os.Chdir(tempDir))
			t.Cleanup(func() {
				require.NoError(t, os.Chdir(oldWd))
			})

			// Create builder and generate machine
			builder, err := NewBuilder()
			require.NoError(t, err)

			outputFile := filepath.Join(outputDir, tt.machineName+".ign")
			err = builder.GenerateMachine(tt.machineName, outputFile)
			require.NoError(t, err)

			// Verify debug file path construction
			expectedPath := filepath.Join(outputDir, tt.expected)
			_, err = os.Stat(expectedPath)
			assert.NoError(t, err, "Should create debug file with correct .yaml extension: %s", tt.expected)

			// Verify file content contains machine name
			content, err := os.ReadFile(expectedPath)
			require.NoError(t, err)
			contentStr := string(content)
			assert.Contains(t, contentStr, tt.machineName)
		})
	}
}

func TestBuilderGeneratesCorrectDebugFileName(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	outputDir := filepath.Join(tempDir, "output")
	ignitionDir := filepath.Join(outputDir, "ignition")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(ignitionDir, 0755))

	// Create defaults.toml
	createDefaultsToml(t, configDir)

	// Create machine structure using new format
	createMachineStructure(t, tempDir, "debug-test", "debug-test.example.com")

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create builder
	builder, err := NewBuilder()
	require.NoError(t, err)

	// Test with ignition file in subdirectory
	outputFile := filepath.Join(ignitionDir, "debug-test.ign")
	err = builder.GenerateMachine("debug-test", outputFile)
	require.NoError(t, err)

	// Verify debug file is created in same directory as ignition file
	expectedDebugFile := filepath.Join(ignitionDir, "debug-test-final-butane.yaml")
	_, err = os.Stat(expectedDebugFile)
	assert.NoError(t, err, "Debug file should be created in same directory as ignition file")

	// Verify file content is valid YAML
	content, err := os.ReadFile(expectedDebugFile)
	require.NoError(t, err)
	contentStr := string(content)

	// Basic YAML validation - should contain proper structure
	assert.Contains(t, contentStr, "variant: fcos")
	assert.Contains(t, contentStr, "version: 1.5.0")
	assert.True(t, strings.HasPrefix(contentStr, "variant:") || strings.Contains(contentStr, "\nvariant:"), "Should start with or contain variant field")
}

func TestBuilderFilesDir(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	scriptsDir := filepath.Join(configDir, "scripts")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create test script files
	testScript := "#!/bin/bash\necho \"Test script\""
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "test-script.sh"), []byte(testScript), 0644))

	// Create defaults.toml
	createDefaultsToml(t, configDir)

	// Create machine structure with custom template for local: directive test
	machineDir := filepath.Join(tempDir, "machines", "test-machine")
	require.NoError(t, os.MkdirAll(machineDir, 0755))

	// Create machine template with local: directive
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
    - path: /usr/local/bin/test-script.sh
      mode: 0755
      contents:
        local: test-script.sh
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

	templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	require.NoError(t, os.WriteFile(templatePath, []byte(machineTemplate), 0644))

	// Create machine.toml
	machineConfig := `name = "test-machine"
fqdn = "test-machine.example.com"
container_image = "registry.example.com/test-machine"
container_tag = "latest"`
	machineConfigPath := filepath.Join(machineDir, "machine.toml")
	require.NoError(t, os.WriteFile(machineConfigPath, []byte(machineConfig), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create builder
	builder, err := NewBuilder()
	require.NoError(t, err)

	// Generate machine ignition - this should work with FilesDir set to config/scripts
	outputFile := filepath.Join(outputDir, "test-machine.ign")
	err = builder.GenerateMachine("test-machine", outputFile)
	require.NoError(t, err, "Builder should handle local: directive with FilesDir configuration")

	// Verify ignition file was created
	_, err = os.Stat(outputFile)
	assert.NoError(t, err, "Ignition file should be created")

	// Verify debug file contains local: reference
	debugFile := filepath.Join(outputDir, "test-machine-final-butane.yaml")
	content, err := os.ReadFile(debugFile)
	require.NoError(t, err)
	contentStr := string(content)
	assert.Contains(t, contentStr, "local: test-script.sh", "Debug file should contain local: directive")
}

func TestBuilderIgnitionValidation(t *testing.T) {
	tests := []struct {
		name          string
		ignitionJSON  string
		expectError   bool
		errorContains string
	}{
		{
			name: "valid ignition config",
			ignitionJSON: `{
				"ignition": {"version": "3.4.0"},
				"passwd": {
					"users": [{"name": "testuser", "groups": ["wheel"]}]
				}
			}`,
			expectError: false,
		},
		{
			name: "invalid ignition config - bad version",
			ignitionJSON: `{
				"ignition": {"version": "invalid"},
				"passwd": {
					"users": [{"name": "testuser"}]
				}
			}`,
			expectError:   true,
			errorContains: "ignition parse error",
		},
		{
			name: "invalid ignition config - malformed JSON",
			ignitionJSON: `{
				"ignition": {"version": "3.4.0"
				"passwd": {
					"users": [{"name": "testuser"}]
				}
			}`,
			expectError:   true,
			errorContains: "ignition parse error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary builder
			builder := &Builder{}

			// Test validation
			err := builder.ValidateIgnitionConfig([]byte(tt.ignitionJSON))

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBuilderValidationIntegration(t *testing.T) {
	// Create temporary directory structure
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	scriptsDir := filepath.Join(configDir, "scripts")
	outputDir := filepath.Join(tempDir, "output")
	require.NoError(t, os.MkdirAll(configDir, 0755))
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))
	require.NoError(t, os.MkdirAll(outputDir, 0755))

	// Create test script files
	testScript := "#!/bin/bash\necho \"Test script\""
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "test-script.sh"), []byte(testScript), 0644))

	// Create defaults.toml
	createDefaultsToml(t, configDir)

	// Create machine structure using new format
	createMachineStructure(t, tempDir, "test-machine", "test-machine.example.com")

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create builder
	builder, err := NewBuilder()
	require.NoError(t, err)

	// Generate machine ignition - this should include validation
	outputFile := filepath.Join(outputDir, "test-machine.ign")
	err = builder.GenerateMachine("test-machine", outputFile)
	require.NoError(t, err, "Builder should generate and validate ignition successfully")

	// Verify ignition file was created and is valid JSON
	ignitionContent, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	// Verify it's valid JSON by unmarshaling
	var ignitionObj map[string]interface{}
	err = json.Unmarshal(ignitionContent, &ignitionObj)
	require.NoError(t, err, "Generated ignition should be valid JSON")

	// Verify basic ignition structure
	assert.Contains(t, ignitionObj, "ignition", "Should contain ignition section")
	assert.Contains(t, ignitionObj, "storage", "Should contain storage section")
}
