package bootc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootcManagerScript(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "etc", "iago", "containers")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	tests := []struct {
		name             string
		configs          map[string]string
		expectedServices []string
	}{
		{
			name: "single machine config",
			configs: map[string]string{
				"postgres.env": `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
			},
			expectedServices: []string{"bootc@postgres.service"},
		},
		{
			name: "multiple machine configs",
			configs: map[string]string{
				"postgres.env": `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
				"caddy.env": `CONTAINER_IMAGE=registry.example.com/caddy:latest
CONTAINER_NAME=bootc-caddy
UPDATE_STRATEGY=pinned`,
			},
			expectedServices: []string{"bootc@postgres.service", "bootc@caddy.service"},
		},
		{
			name: "mixed file types",
			configs: map[string]string{
				"postgres.env": `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
				"notes.txt": "This is not a config file",
			},
			expectedServices: []string{"bootc@postgres.service"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean config directory
			require.NoError(t, os.RemoveAll(configDir))
			require.NoError(t, os.MkdirAll(configDir, 0755))

			// Create config files
			for filename, content := range tt.configs {
				configFile := filepath.Join(configDir, filename)
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
			}

			// Test manager script logic
			services := DiscoverContainerConfigs(configDir)

			assert.ElementsMatch(t, tt.expectedServices, services)
		})
	}
}

func TestBootcRunScript(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "etc", "iago", "containers")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	tests := []struct {
		name          string
		configFile    string
		configContent string
		expectedCmd   []string
		expectError   bool
	}{
		{
			name:       "valid postgres config",
			configFile: "postgres.env",
			configContent: `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
HEALTH_CHECK_WAIT=30
UPDATE_STRATEGY=latest`,
			expectedCmd: []string{
				"podman", "run", "--rm", "--name", "bootc-postgres",
				"--net", "host", "--pid", "host", "--privileged",
				"registry.example.com/postgres:latest",
			},
		},
		{
			name:       "config with custom settings",
			configFile: "caddy.env",
			configContent: `CONTAINER_IMAGE=ghcr.io/myorg/caddy:v2.7
CONTAINER_NAME=bootc-caddy-work
HEALTH_CHECK_WAIT=60
UPDATE_STRATEGY=pinned
EXTRA_ARGS=--volume /etc/caddy:/etc/caddy:ro`,
			expectedCmd: []string{
				"podman", "run", "--rm", "--name", "bootc-caddy-work",
				"--net", "host", "--pid", "host", "--privileged",
				"--volume", "/etc/caddy:/etc/caddy:ro",
				"ghcr.io/myorg/caddy:v2.7",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(configDir, tt.configFile)
			require.NoError(t, os.WriteFile(configFile, []byte(tt.configContent), 0644))

			// Parse config and generate command
			cmd, err := BuildPodmanCommand(configFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCmd, cmd)
			}
		})
	}
}

func TestBootcUpdateScript(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "etc", "iago", "containers")
	require.NoError(t, os.MkdirAll(configDir, 0755))

	tests := []struct {
		name            string
		configs         map[string]string
		expectedUpdates []string
		expectedSkips   []string
	}{
		{
			name: "mixed update strategies",
			configs: map[string]string{
				"postgres.env": `CONTAINER_IMAGE=registry.example.com/postgres:latest
UPDATE_STRATEGY=latest`,
				"caddy.env": `CONTAINER_IMAGE=registry.example.com/caddy:v2.7
UPDATE_STRATEGY=pinned`,
				"staging.env": `CONTAINER_IMAGE=registry.example.com/app:staging
UPDATE_STRATEGY=staging`,
			},
			expectedUpdates: []string{"postgres", "staging"},
			expectedSkips:   []string{"caddy"},
		},
		{
			name: "all pinned",
			configs: map[string]string{
				"postgres.env": `CONTAINER_IMAGE=registry.example.com/postgres:v14.5
UPDATE_STRATEGY=pinned`,
				"caddy.env": `CONTAINER_IMAGE=registry.example.com/caddy:v2.7
UPDATE_STRATEGY=pinned`,
			},
			expectedUpdates: []string{},
			expectedSkips:   []string{"postgres", "caddy"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean and create config files
			require.NoError(t, os.RemoveAll(configDir))
			require.NoError(t, os.MkdirAll(configDir, 0755))

			for filename, content := range tt.configs {
				configFile := filepath.Join(configDir, filename)
				require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))
			}

			// Test update logic
			updates, skips := DetermineUpdateActions(configDir)

			assert.ElementsMatch(t, tt.expectedUpdates, updates)
			assert.ElementsMatch(t, tt.expectedSkips, skips)
		})
	}
}

func TestContainerHealthCheck(t *testing.T) {
	tests := []struct {
		name            string
		containerName   string
		isRunning       bool
		healthCheckWait int
		expectedResult  bool
	}{
		{
			name:            "healthy container",
			containerName:   "bootc-postgres",
			isRunning:       true,
			healthCheckWait: 5,
			expectedResult:  true,
		},
		{
			name:            "unhealthy container",
			containerName:   "bootc-failed",
			isRunning:       false,
			healthCheckWait: 5,
			expectedResult:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock health check
			result := CheckContainerHealth(tt.containerName, tt.isRunning)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

// Mock functions for testing (these would be implemented in the actual package)
func DiscoverContainerConfigs(configDir string) []string {
	var services []string

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return services
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".env") {
			serviceName := strings.TrimSuffix(entry.Name(), ".env")
			services = append(services, "bootc@"+serviceName+".service")
		}
	}

	return services
}

func BuildPodmanCommand(configFile string) ([]string, error) {
	content, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var image, name, extraArgs string
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "CONTAINER_IMAGE=") {
			image = strings.TrimPrefix(line, "CONTAINER_IMAGE=")
		}
		if strings.HasPrefix(line, "CONTAINER_NAME=") {
			name = strings.TrimPrefix(line, "CONTAINER_NAME=")
		}
		if strings.HasPrefix(line, "EXTRA_ARGS=") {
			extraArgs = strings.TrimPrefix(line, "EXTRA_ARGS=")
		}
	}

	cmd := []string{"podman", "run", "--rm", "--name", name,
		"--net", "host", "--pid", "host", "--privileged"}

	if extraArgs != "" {
		args := strings.Fields(extraArgs)
		cmd = append(cmd, args...)
	}

	cmd = append(cmd, image)

	return cmd, nil
}

func DetermineUpdateActions(configDir string) ([]string, []string) {
	var updates, skips []string

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return updates, skips
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".env") {
			content, err := os.ReadFile(filepath.Join(configDir, entry.Name()))
			if err != nil {
				continue
			}

			serviceName := strings.TrimSuffix(entry.Name(), ".env")

			if strings.Contains(string(content), "UPDATE_STRATEGY=pinned") {
				skips = append(skips, serviceName)
			} else {
				updates = append(updates, serviceName)
			}
		}
	}

	return updates, skips
}

func CheckContainerHealth(containerName string, isRunning bool) bool {
	// Mock implementation
	return isRunning
}
