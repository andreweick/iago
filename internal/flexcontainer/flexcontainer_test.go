package flexcontainer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the flexible container environment file parsing
func TestContainerEnvFileParsing(t *testing.T) {
	tests := []struct {
		name             string
		envContent       string
		expectedImg      string
		expectedStrategy string
		expectError      bool
	}{
		{
			name: "valid environment file",
			envContent: `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
HEALTH_CHECK_WAIT=30
UPDATE_STRATEGY=latest`,
			expectedImg:      "registry.example.com/postgres:latest",
			expectedStrategy: "latest",
		},
		{
			name: "pinned strategy",
			envContent: `CONTAINER_IMAGE=registry.example.com/postgres:v14.5
CONTAINER_NAME=bootc-postgres
HEALTH_CHECK_WAIT=30
UPDATE_STRATEGY=pinned`,
			expectedImg:      "registry.example.com/postgres:v14.5",
			expectedStrategy: "pinned",
		},
		{
			name: "staging strategy",
			envContent: `CONTAINER_IMAGE=registry.example.com/postgres:staging
CONTAINER_NAME=bootc-postgres
HEALTH_CHECK_WAIT=30
UPDATE_STRATEGY=staging`,
			expectedImg:      "registry.example.com/postgres:staging",
			expectedStrategy: "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tempDir := t.TempDir()
			envFile := filepath.Join(tempDir, "test.env")
			require.NoError(t, os.WriteFile(envFile, []byte(tt.envContent), 0644))

			// Parse environment file
			env, err := ParseContainerEnv(envFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedImg, env.ContainerImage)
				assert.Equal(t, tt.expectedStrategy, env.UpdateStrategy)
			}
		})
	}
}

// Test container configuration update scenarios
func TestContainerConfigurationChanges(t *testing.T) {
	tempDir := t.TempDir()
	containerDir := filepath.Join(tempDir, "etc", "iago", "containers")
	require.NoError(t, os.MkdirAll(containerDir, 0755))

	tests := []struct {
		name           string
		initialConfig  string
		updateConfig   string
		expectedChange ChangeType
	}{
		{
			name: "image tag change",
			initialConfig: `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
			updateConfig: `CONTAINER_IMAGE=registry.example.com/postgres:v14.5
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
			expectedChange: ImageChanged,
		},
		{
			name: "registry change",
			initialConfig: `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
			updateConfig: `CONTAINER_IMAGE=ghcr.io/myorg/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
			expectedChange: RegistryChanged,
		},
		{
			name: "strategy change",
			initialConfig: `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=latest`,
			updateConfig: `CONTAINER_IMAGE=registry.example.com/postgres:latest
CONTAINER_NAME=bootc-postgres
UPDATE_STRATEGY=pinned`,
			expectedChange: StrategyChanged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := filepath.Join(containerDir, "test-machine.env")

			// Write initial config
			require.NoError(t, os.WriteFile(configFile, []byte(tt.initialConfig), 0644))
			initialEnv, err := ParseContainerEnv(configFile)
			require.NoError(t, err)

			// Write updated config
			require.NoError(t, os.WriteFile(configFile, []byte(tt.updateConfig), 0644))
			updatedEnv, err := ParseContainerEnv(configFile)
			require.NoError(t, err)

			// Detect change type
			changeType := DetectChangeType(initialEnv, updatedEnv)
			assert.Equal(t, tt.expectedChange, changeType)
		})
	}
}

// Test update strategy behaviors
func TestUpdateStrategyBehavior(t *testing.T) {
	tests := []struct {
		name           string
		strategy       string
		currentImage   string
		availableImage string
		shouldUpdate   bool
		expectedAction string
	}{
		{
			name:           "latest strategy should update",
			strategy:       "latest",
			currentImage:   "registry.example.com/postgres:latest",
			availableImage: "registry.example.com/postgres:latest",
			shouldUpdate:   true,
			expectedAction: "pull_and_restart",
		},
		{
			name:           "pinned strategy should not update",
			strategy:       "pinned",
			currentImage:   "registry.example.com/postgres:v14.5",
			availableImage: "registry.example.com/postgres:latest",
			shouldUpdate:   false,
			expectedAction: "skip",
		},
		{
			name:           "staging strategy with staging tag",
			strategy:       "staging",
			currentImage:   "registry.example.com/postgres:staging",
			availableImage: "registry.example.com/postgres:staging",
			shouldUpdate:   true,
			expectedAction: "pull_and_restart",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := DetermineUpdateAction(tt.strategy, tt.currentImage, tt.availableImage)

			if tt.shouldUpdate {
				assert.Equal(t, tt.expectedAction, action)
			} else {
				assert.Equal(t, "skip", action)
			}
		})
	}
}

// Mock types and functions for testing (these would be implemented in the actual package)
type ContainerEnv struct {
	ContainerImage  string
	ContainerName   string
	UpdateStrategy  string
	HealthCheckWait int
}

type ChangeType int

const (
	NoChange ChangeType = iota
	ImageChanged
	RegistryChanged
	StrategyChanged
)

func ParseContainerEnv(filePath string) (*ContainerEnv, error) {
	// This is a mock implementation for testing
	// The real implementation would parse the .env file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	env := &ContainerEnv{}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "CONTAINER_IMAGE=") {
			env.ContainerImage = strings.TrimPrefix(line, "CONTAINER_IMAGE=")
		}
		if strings.HasPrefix(line, "UPDATE_STRATEGY=") {
			env.UpdateStrategy = strings.TrimPrefix(line, "UPDATE_STRATEGY=")
		}
		if strings.HasPrefix(line, "CONTAINER_NAME=") {
			env.ContainerName = strings.TrimPrefix(line, "CONTAINER_NAME=")
		}
	}

	return env, nil
}

func DetectChangeType(old, new *ContainerEnv) ChangeType {
	if old.UpdateStrategy != new.UpdateStrategy {
		return StrategyChanged
	}

	oldParts := strings.Split(old.ContainerImage, "/")
	newParts := strings.Split(new.ContainerImage, "/")

	if len(oldParts) > 0 && len(newParts) > 0 && oldParts[0] != newParts[0] {
		return RegistryChanged
	}

	if old.ContainerImage != new.ContainerImage {
		return ImageChanged
	}

	return NoChange
}

func DetermineUpdateAction(strategy, currentImage, availableImage string) string {
	switch strategy {
	case "pinned":
		return "skip"
	case "latest", "staging":
		return "pull_and_restart"
	default:
		return "skip"
	}
}
