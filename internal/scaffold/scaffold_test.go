package scaffold

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andreweick/iago/internal/machine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScaffolder_createMachineButaneScaffold(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	machineDir := filepath.Join(tempDir, "machines", "test-machine")
	require.NoError(t, os.MkdirAll(machineDir, 0755))

	// Create scaffolder
	defaults := machine.Defaults{
		ContainerRegistry: machine.ContainerRegistryConfig{
			URL: "registry.example.com",
		},
	}
	scaffolder := NewScaffolder(defaults)

	// Create scaffold options
	opts := ScaffoldOptions{
		MachineName: "test-machine",
		FQDN:        "test-machine.example.com",
		OutputDir:   tempDir,
	}

	// Test createMachineButaneScaffold
	err := scaffolder.createMachineButaneScaffold(machineDir, opts)
	assert.NoError(t, err)

	// Verify file was created
	machineButanePath := filepath.Join(machineDir, "butane.yaml.tmpl")
	_, err = os.Stat(machineButanePath)
	assert.NoError(t, err, "butane.yaml.tmpl should be created")

	// Verify file content
	content, err := os.ReadFile(machineButanePath)
	require.NoError(t, err)

	contentStr := string(content)
	assert.Contains(t, contentStr, "variant: fcos")
	assert.Contains(t, contentStr, "{{ .Machine.Name }}")
	assert.Contains(t, contentStr, "bootc@.service")
	assert.Contains(t, contentStr, "password_hash")
}

func TestScaffolder_CreateMachineScaffold_WithMachineButane(t *testing.T) {
	// Create temporary directory and change to it
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create workloads directory to trigger new structure
	require.NoError(t, os.MkdirAll("workloads", 0755))

	// Create the prompt file that will be copied
	require.NoError(t, os.MkdirAll("containers", 0755))
	promptContent := "# Bootc Container Creation Prompt\nTest prompt content"
	err = os.WriteFile("containers/bootc-container-creation-prompt.md", []byte(promptContent), 0644)
	require.NoError(t, err)

	// Create scaffolder
	defaults := machine.Defaults{
		ContainerRegistry: machine.ContainerRegistryConfig{
			URL: "registry.example.com",
		},
	}
	scaffolder := NewScaffolder(defaults)

	// Create scaffold options
	opts := ScaffoldOptions{
		MachineName: "test-machine",
		FQDN:        "test-machine.example.com",
		OutputDir:   tempDir,
	}

	// Create machine scaffold
	err = scaffolder.CreateMachineScaffold(opts)
	assert.NoError(t, err)

	// Verify butane.yaml.tmpl was created
	machineButanePath := filepath.Join("machines", "test-machine", "butane.yaml.tmpl")
	_, err = os.Stat(machineButanePath)
	assert.NoError(t, err, "butane.yaml.tmpl should be created during scaffold")

	// Verify machine.toml was created (unified config)
	machinePath := filepath.Join("machines", "test-machine", "machine.toml")
	_, err = os.Stat(machinePath)
	assert.NoError(t, err, "machine.toml should be created")

	// Verify Containerfile was created
	containerfilePath := filepath.Join("containers", "test-machine", "Containerfile")
	_, err = os.Stat(containerfilePath)
	assert.NoError(t, err, "Containerfile should be created")

	// Verify prompt file was created
	promptPath := filepath.Join("containers", "test-machine", "test-machine-prompt.md")
	_, err = os.Stat(promptPath)
	assert.NoError(t, err, "prompt file should be created")

	// Verify no scripts directory was created
	scriptsPath := filepath.Join("containers", "test-machine", "scripts")
	_, err = os.Stat(scriptsPath)
	assert.True(t, os.IsNotExist(err), "scripts directory should not be created")

	// Verify no systemd directory was created
	systemdPath := filepath.Join("containers", "test-machine", "systemd")
	_, err = os.Stat(systemdPath)
	assert.True(t, os.IsNotExist(err), "systemd directory should not be created")

	// Verify no config directory was created
	configPath := filepath.Join("containers", "test-machine", "config")
	_, err = os.Stat(configPath)
	assert.True(t, os.IsNotExist(err), "config directory should not be created")
}

func TestScaffoldCreatesYamlFiles(t *testing.T) {
	// Create temporary directory and change to it
	tempDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(oldWd))
	})

	// Create workloads directory to trigger new structure
	require.NoError(t, os.MkdirAll("workloads", 0755))

	// Create the prompt file that will be copied
	require.NoError(t, os.MkdirAll("containers", 0755))
	promptContent := "# Bootc Container Creation Prompt\nTest prompt content"
	err = os.WriteFile("containers/bootc-container-creation-prompt.md", []byte(promptContent), 0644)
	require.NoError(t, err)

	// Create scaffolder
	defaults := machine.Defaults{
		ContainerRegistry: machine.ContainerRegistryConfig{
			URL: "registry.example.com",
		},
	}
	scaffolder := NewScaffolder(defaults)

	// Create scaffold options
	opts := ScaffoldOptions{
		MachineName: "yaml-test-machine",
		FQDN:        "yaml-test-machine.example.com",
		OutputDir:   tempDir,
	}

	// Create machine scaffold
	err = scaffolder.CreateMachineScaffold(opts)
	assert.NoError(t, err)

	// Verify .yaml.tmpl file is created, not .yml
	yamlPath := filepath.Join("machines", "yaml-test-machine", "butane.yaml.tmpl")
	_, err = os.Stat(yamlPath)
	assert.NoError(t, err, "Should create .yaml.tmpl file")

	// Verify no .yml file exists
	ymlPath := filepath.Join("machines", "yaml-test-machine", "butane.yml")
	_, err = os.Stat(ymlPath)
	assert.True(t, os.IsNotExist(err), "Should not create .yml file")
}

func TestScaffoldFilePathConstruction(t *testing.T) {
	tests := []struct {
		name        string
		machineName string
		expected    string
	}{
		{
			name:        "simple machine name",
			machineName: "web-server",
			expected:    "butane.yaml.tmpl",
		},
		{
			name:        "machine with numbers",
			machineName: "db-01",
			expected:    "butane.yaml.tmpl",
		},
		{
			name:        "complex machine name",
			machineName: "postgres-primary-cluster",
			expected:    "butane.yaml.tmpl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir := t.TempDir()
			machineDir := filepath.Join(tempDir, "machines", tt.machineName)
			require.NoError(t, os.MkdirAll(machineDir, 0755))

			// Create scaffolder
			defaults := machine.Defaults{
				ContainerRegistry: machine.ContainerRegistryConfig{
					URL: "registry.example.com",
				},
			}
			scaffolder := NewScaffolder(defaults)

			// Create scaffold options
			opts := ScaffoldOptions{
				MachineName: tt.machineName,
				FQDN:        tt.machineName + ".example.com",
				OutputDir:   tempDir,
			}

			// Test createMachineButaneScaffold
			err := scaffolder.createMachineButaneScaffold(machineDir, opts)
			assert.NoError(t, err)

			// Verify file was created with correct extension
			expectedPath := filepath.Join(machineDir, tt.expected)
			_, err = os.Stat(expectedPath)
			assert.NoError(t, err, "Should create file with .yaml.tmpl extension: %s", tt.expected)

			// Verify file content
			content, err := os.ReadFile(expectedPath)
			require.NoError(t, err)
			contentStr := string(content)
			assert.Contains(t, contentStr, "variant: fcos")
			assert.Contains(t, contentStr, "{{ .Machine.Name }}")
		})
	}
}
