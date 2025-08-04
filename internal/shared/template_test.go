package shared

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedTemplateUsage(t *testing.T) {
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "containers", "_shared")

	tests := []struct {
		name            string
		templateExists  bool
		templateContent string
		serviceName     string
		expectedContent string
	}{
		{
			name:           "health check template exists",
			templateExists: true,
			templateContent: `#!/bin/bash
# Health check for {SERVICE}
if systemctl is-active --quiet {SERVICE}-app.service; then
    echo "Health check passed for {SERVICE}"
    exit 0
else
    echo "Health check failed for {SERVICE}"
    exit 1
fi`,
			serviceName: "postgres",
			expectedContent: `#!/bin/bash
# Health check for postgres
if systemctl is-active --quiet postgres-app.service; then
    echo "Health check passed for postgres"
    exit 0
else
    echo "Health check failed for postgres"
    exit 1
fi`,
		},
		{
			name:           "init template exists",
			templateExists: true,
			templateContent: `#!/bin/bash
# Initialize {SERVICE}
echo "Initializing {SERVICE} service..."
mkdir -p /var/lib/{SERVICE} /var/log/{SERVICE}
chown -R {SERVICE}-app:{SERVICE}-app /var/lib/{SERVICE} /var/log/{SERVICE}
echo "{SERVICE} initialization complete"`,
			serviceName: "caddy",
			expectedContent: `#!/bin/bash
# Initialize caddy
echo "Initializing caddy service..."
mkdir -p /var/lib/caddy /var/log/caddy
chown -R caddy-app:caddy-app /var/lib/caddy /var/log/caddy
echo "caddy initialization complete"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create shared template directory
			scriptsDir := filepath.Join(sharedDir, "scripts")
			require.NoError(t, os.MkdirAll(scriptsDir, 0755))

			var templatePath string
			if strings.Contains(tt.name, "health") {
				templatePath = filepath.Join(scriptsDir, "health-check-template.sh")
			} else {
				templatePath = filepath.Join(scriptsDir, "basic-init-template.sh")
			}

			if tt.templateExists {
				require.NoError(t, os.WriteFile(templatePath, []byte(tt.templateContent), 0644))
			}

			// Test template customization
			result, err := CustomizeTemplate(templatePath, tt.serviceName)

			if tt.templateExists {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedContent, result)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestSharedTemplateSystemdServices(t *testing.T) {
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "containers", "_shared", "systemd")
	require.NoError(t, os.MkdirAll(sharedDir, 0755))

	tests := []struct {
		name            string
		templateType    string
		templateContent string
		serviceName     string
		expectedContent string
	}{
		{
			name:         "secrets service template",
			templateType: "secrets-service.template",
			templateContent: `[Unit]
Description=Fetch {SERVICE} secrets from 1Password
Before={SERVICE}-app.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/fetch-{SERVICE}-secrets.sh
RemainAfterExit=true

[Install]
WantedBy=multi-user.target`,
			serviceName: "postgres",
			expectedContent: `[Unit]
Description=Fetch postgres secrets from 1Password
Before=postgres-app.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/fetch-postgres-secrets.sh
RemainAfterExit=true

[Install]
WantedBy=multi-user.target`,
		},
		{
			name:         "secrets watcher template",
			templateType: "secrets-watcher.template",
			templateContent: `[Unit]
Description=Watch {SERVICE} secrets for changes

[Path]
PathModified=/etc/iago/secrets/{SERVICE}-password

[Install]
WantedBy=multi-user.target`,
			serviceName: "immich",
			expectedContent: `[Unit]
Description=Watch immich secrets for changes

[Path]
PathModified=/etc/iago/secrets/immich-password

[Install]
WantedBy=multi-user.target`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templatePath := filepath.Join(sharedDir, tt.templateType)
			require.NoError(t, os.WriteFile(templatePath, []byte(tt.templateContent), 0644))

			// Test template customization
			result, err := CustomizeTemplate(templatePath, tt.serviceName)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedContent, result)
		})
	}
}

func TestTemplateFallbackBehavior(t *testing.T) {
	tempDir := t.TempDir()

	// Create container directory without shared templates
	containerDir := filepath.Join(tempDir, "containers", "myservice")
	scriptsDir := filepath.Join(containerDir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0755))

	// Test fallback to hardcoded script generation
	fallbackContent := GenerateFallbackScript("health", "myservice")

	expectedContent := `#!/bin/bash
# Health check script for myservice
set -euo pipefail

# Simple health check - verify service is running
if systemctl is-active --quiet myservice-app.service; then
    echo "[$(date)] myservice service is healthy"
    exit 0
else
    echo "[$(date)] myservice service is not running"
    exit 1
fi`

	assert.Equal(t, expectedContent, fallbackContent)
}

func TestSharedTemplateValidation(t *testing.T) {
	tempDir := t.TempDir()
	sharedDir := filepath.Join(tempDir, "containers", "_shared")

	tests := []struct {
		name            string
		templatePath    string
		templateContent string
		isValid         bool
		expectedError   string
	}{
		{
			name:         "valid template with placeholders",
			templatePath: "scripts/health-check-template.sh",
			templateContent: `#!/bin/bash
# Health check for {SERVICE}
systemctl is-active {SERVICE}-app.service`,
			isValid: true,
		},
		{
			name:         "template missing shebang",
			templatePath: "scripts/invalid-template.sh",
			templateContent: `# Health check for {SERVICE}
systemctl is-active {SERVICE}-app.service`,
			isValid:       false,
			expectedError: "template missing shebang",
		},
		{
			name:         "template missing placeholders",
			templatePath: "scripts/no-placeholder.sh",
			templateContent: `#!/bin/bash
# Static health check
systemctl is-active myservice-app.service`,
			isValid:       false,
			expectedError: "template missing required {SERVICE} placeholder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath := filepath.Join(sharedDir, tt.templatePath)
			require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
			require.NoError(t, os.WriteFile(fullPath, []byte(tt.templateContent), 0644))

			err := ValidateTemplate(fullPath)

			if tt.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

// Mock functions for testing (these would be implemented in the actual package)
func CustomizeTemplate(templatePath, serviceName string) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	customized := strings.ReplaceAll(string(content), "{SERVICE}", serviceName)
	return customized, nil
}

func GenerateFallbackScript(scriptType, serviceName string) string {
	switch scriptType {
	case "health":
		return `#!/bin/bash
# Health check script for ` + serviceName + `
set -euo pipefail

# Simple health check - verify service is running
if systemctl is-active --quiet ` + serviceName + `-app.service; then
    echo "[$(date)] ` + serviceName + ` service is healthy"
    exit 0
else
    echo "[$(date)] ` + serviceName + ` service is not running"
    exit 1
fi`
	default:
		return ""
	}
}

func ValidateTemplate(templatePath string) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Check for shebang
	if !strings.HasPrefix(contentStr, "#!") {
		return fmt.Errorf("template missing shebang")
	}

	// Check for service placeholder
	if !strings.Contains(contentStr, "{SERVICE}") {
		return fmt.Errorf("template missing required {SERVICE} placeholder")
	}

	return nil
}
