package butane

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/andreweick/iago/internal/machine"
	"github.com/andreweick/iago/internal/workload"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderer_LocalFileResolution(t *testing.T) {
	tests := []struct {
		name           string
		scriptContent  string
		expectedInline string
	}{
		{
			name:           "valid local script reference",
			scriptContent:  "#!/bin/bash\necho \"Hello from script\"",
			expectedInline: "#!/bin/bash\necho \"Hello from script\"",
		},
		{
			name:           "script with unicode characters",
			scriptContent:  "#!/bin/bash\necho \"🚀 Unicode test\"",
			expectedInline: "#!/bin/bash\necho \"🚀 Unicode test\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := t.TempDir()
			scriptsDir := filepath.Join(tempDir, "config", "scripts")
			require.NoError(t, os.MkdirAll(scriptsDir, 0755))

			// Create test script
			scriptPath := filepath.Join(scriptsDir, "test-script.sh")
			require.NoError(t, os.WriteFile(scriptPath, []byte(tt.scriptContent), 0644))

			// Create machine template with local: directive
			machineDir := filepath.Join(tempDir, "machines", "test-machine")
			require.NoError(t, os.MkdirAll(machineDir, 0755))

			template := `variant: fcos
version: 1.5.0
storage:
  files:
    - path: /usr/local/bin/test-script.sh
      mode: 0755
      contents:
        local: test-script.sh
systemd:
  units:
    - name: podman.service
      enabled: true`

			templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
			require.NoError(t, os.WriteFile(templatePath, []byte(template), 0644))

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

			// Create renderer
			defaults := machine.Defaults{
				User: machine.UserConfig{
					Username:     "testuser",
					Groups:       []string{"sudo", "wheel"},
					PasswordHash: "$6$test$hash",
				},
				Admin: machine.AdminConfig{
					Username:     "admin",
					Groups:       []string{"sudo"},
					PasswordHash: "$6$admin$hash",
				},
				Network: machine.NetworkConfig{
					Timezone: "UTC",
				},
			}
			registry := &workload.Registry{}
			renderer := NewRenderer(defaults, registry)

			// Create machine config
			machineConf := machine.Config{
				Name:           "test-machine",
				FQDN:           "test-machine.example.com",
				ContainerImage: "registry.example.com/test-machine",
				ContainerTag:   "latest",
			}

			// Render the machine
			result, err := renderer.RenderMachine(machineConf)
			assert.NoError(t, err)
			assert.Contains(t, result, "local: test-script.sh", "Should contain local: directive")
		})
	}
}

func TestTemplateHelpers_indent(t *testing.T) {
	tests := []struct {
		name     string
		spaces   int
		input    string
		expected string
	}{
		{
			name:     "single line",
			spaces:   2,
			input:    "hello",
			expected: "  hello",
		},
		{
			name:     "multiple lines",
			spaces:   4,
			input:    "line1\nline2",
			expected: "    line1\n    line2",
		},
		{
			name:     "empty lines preserved",
			spaces:   2,
			input:    "line1\n\nline3",
			expected: "  line1\n\n  line3",
		},
		{
			name:     "zero spaces",
			spaces:   0,
			input:    "hello\nworld",
			expected: "hello\nworld",
		},
		{
			name:     "empty string",
			spaces:   2,
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := indent(tt.spaces, tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateHelpers_toYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "simple string",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "string array",
			input:    []string{"item1", "item2"},
			expected: "- item1\n- item2",
		},
		{
			name:     "simple map",
			input:    map[string]string{"key": "value"},
			expected: "key: value",
		},
		{
			name:     "number",
			input:    42,
			expected: "42",
		},
		{
			name:     "boolean",
			input:    true,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := toYAML(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateHelpers_defaultValue(t *testing.T) {
	tests := []struct {
		name        string
		defaultVal  interface{}
		value       interface{}
		expectedVal interface{}
	}{
		{
			name:        "empty string uses default",
			defaultVal:  "default",
			value:       "",
			expectedVal: "default",
		},
		{
			name:        "non-empty string uses value",
			defaultVal:  "default",
			value:       "actual",
			expectedVal: "actual",
		},
		{
			name:        "nil uses default",
			defaultVal:  "default",
			value:       nil,
			expectedVal: "default",
		},
		{
			name:        "zero int uses default",
			defaultVal:  42,
			value:       0,
			expectedVal: 42,
		},
		{
			name:        "non-zero int uses value",
			defaultVal:  42,
			value:       100,
			expectedVal: 100,
		},
		{
			name:        "empty slice uses default",
			defaultVal:  []string{"default"},
			value:       []string{},
			expectedVal: []string{"default"},
		},
		{
			name:        "non-empty slice uses value",
			defaultVal:  []string{"default"},
			value:       []string{"actual"},
			expectedVal: []string{"actual"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultValue(tt.defaultVal, tt.value)
			assert.Equal(t, tt.expectedVal, result)
		})
	}
}

func TestTemplateHelpers_hasKey(t *testing.T) {
	tests := []struct {
		name     string
		m        map[string]interface{}
		key      string
		expected bool
	}{
		{
			name:     "key exists",
			m:        map[string]interface{}{"test": "value"},
			key:      "test",
			expected: true,
		},
		{
			name:     "key does not exist",
			m:        map[string]interface{}{"other": "value"},
			key:      "test",
			expected: false,
		},
		{
			name:     "empty map",
			m:        map[string]interface{}{},
			key:      "test",
			expected: false,
		},
		{
			name:     "nil value but key exists",
			m:        map[string]interface{}{"test": nil},
			key:      "test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasKey(tt.m, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTemplateHelpers_list(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		expected []string
	}{
		{
			name:     "empty list",
			items:    []string{},
			expected: []string{},
		},
		{
			name:     "single item",
			items:    []string{"item1"},
			expected: []string{"item1"},
		},
		{
			name:     "multiple items",
			items:    []string{"item1", "item2", "item3"},
			expected: []string{"item1", "item2", "item3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := list(tt.items...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderer_TemplateErrors(t *testing.T) {
	tests := []struct {
		name            string
		templateContent string
		expectError     bool
		errorSubstring  string
	}{
		{
			name:            "invalid template syntax",
			templateContent: `variant: fcos\nversion: 1.5.0\nstorage:\n  files:\n    - path: /test\n      contents:\n        inline: "{{ .InvalidSyntax"`,
			expectError:     true,
			errorSubstring:  "failed to parse template",
		},
		{
			name:            "missing local file",
			templateContent: `variant: fcos\nversion: 1.5.0\nstorage:\n  files:\n    - path: /test\n      contents:\n        local: nonexistent-script.sh`,
			expectError:     false, // This would be caught during butane-to-ignition conversion, not template rendering
		},
		{
			name:            "template with undefined variable",
			templateContent: `variant: fcos\nversion: 1.5.0\nstorage:\n  files:\n    - path: /test\n      contents:\n        inline: "{{ .NonExistent.Field }}"`,
			expectError:     true,
			errorSubstring:  "failed to execute template",
		},
		{
			name:            "valid template",
			templateContent: `variant: fcos\nversion: 1.5.0\nstorage:\n  files:\n    - path: /etc/hostname\n      contents:\n        inline: "{{ .Machine.Name }}"`,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory structure
			tempDir := t.TempDir()
			machineDir := filepath.Join(tempDir, "machines", "test-machine")
			require.NoError(t, os.MkdirAll(machineDir, 0755))

			// Create machine template
			templatePath := filepath.Join(machineDir, "butane.yaml.tmpl")
			require.NoError(t, os.WriteFile(templatePath, []byte(tt.templateContent), 0644))

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

			// Create renderer
			defaults := machine.Defaults{
				User: machine.UserConfig{
					Username:     "testuser",
					Groups:       []string{"sudo", "wheel"},
					PasswordHash: "$6$test$hash",
				},
				Admin: machine.AdminConfig{
					Username:     "admin",
					Groups:       []string{"sudo"},
					PasswordHash: "$6$admin$hash",
				},
				Network: machine.NetworkConfig{
					Timezone: "UTC",
				},
			}
			registry := &workload.Registry{}
			renderer := NewRenderer(defaults, registry)

			// Create machine config
			machineConf := machine.Config{
				Name:           "test-machine",
				FQDN:           "test-machine.example.com",
				ContainerImage: "registry.example.com/test-machine",
				ContainerTag:   "latest",
			}

			// Test rendering
			_, err = renderer.RenderMachine(machineConf)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorSubstring != "" {
					assert.Contains(t, err.Error(), tt.errorSubstring)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRenderer_getTemplateFuncs(t *testing.T) {
	renderer := &Renderer{}
	funcs := renderer.getTemplateFuncs()

	// Test that all expected functions are present
	expectedFuncs := []string{"indent", "toYAML", "default", "hasKey", "list"}
	for _, funcName := range expectedFuncs {
		assert.Contains(t, funcs, funcName, "Template function %s should be available", funcName)
	}
}
