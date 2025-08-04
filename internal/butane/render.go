package butane

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"

	"github.com/andreweick/iago/internal/github"
	"github.com/andreweick/iago/internal/machine"
	"github.com/andreweick/iago/internal/workload"
	"gopkg.in/yaml.v3"
)

// getTemplateFuncs returns custom template functions for butane templates
func (r *Renderer) getTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"indent":  indent,
		"toYAML":  toYAML,
		"default": defaultValue,
		"hasKey":  hasKey,
		"list":    list,
	}
}

// indent adds the specified number of spaces to each line
func indent(spaces int, v string) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(v, "\n")
	for i := range lines {
		// Don't indent empty lines
		if lines[i] != "" {
			lines[i] = pad + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

// toYAML converts a value to YAML format
func toYAML(v interface{}) (string, error) {
	data, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

// defaultValue returns the value if it's not zero, otherwise returns the default
func defaultValue(defaultVal, value interface{}) interface{} {
	// Check if value is nil or zero value
	if value == nil {
		return defaultVal
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return defaultVal
		}
	case int:
		if v == 0 {
			return defaultVal
		}
	case []string:
		if len(v) == 0 {
			return defaultVal
		}
	case []interface{}:
		if len(v) == 0 {
			return defaultVal
		}
	}

	return value
}

// hasKey checks if a map has a specific key
func hasKey(m map[string]interface{}, key string) bool {
	_, ok := m[key]
	return ok
}

// list creates a list from arguments (useful for default values)
func list(items ...string) []string {
	return items
}

type TemplateData struct {
	User              machine.UserConfig
	Admin             machine.AdminConfig
	Network           machine.NetworkConfig
	Updates           machine.UpdateConfig
	Bootc             machine.BootcConfig
	ContainerRegistry machine.ContainerRegistryConfig
	Machine           machine.Config
	GeneratedSecrets  machine.GeneratedSecrets
	UserSSHKeys       []string // SSH keys fetched from GitHub
}

type Renderer struct {
	defaults machine.Defaults
	registry *workload.Registry
}

func NewRenderer(defaults machine.Defaults, registry *workload.Registry) *Renderer {
	return &Renderer{
		defaults: defaults,
		registry: registry,
	}
}

func (r *Renderer) RenderMachine(machineConfig machine.Config) (string, error) {
	// Generate secrets for the machine
	secrets, err := r.generateMachineSecrets(machineConfig.Name)
	if err != nil {
		return "", fmt.Errorf("failed to generate secrets: %w", err)
	}

	// Fetch SSH keys from GitHub if username is configured
	var userSSHKeys []string
	if r.defaults.User.GitHubUsername != "" {
		keys, err := github.FetchSSHKeys(r.defaults.User.GitHubUsername)
		if err != nil {
			return "", fmt.Errorf("failed to fetch SSH keys from GitHub: %w", err)
		}
		userSSHKeys = keys
	}

	// Prepare template data
	templateData := TemplateData{
		User:              r.defaults.User,
		Admin:             r.defaults.Admin,
		Network:           r.defaults.Network,
		Updates:           r.defaults.Updates,
		Bootc:             r.defaults.Bootc,
		ContainerRegistry: r.defaults.ContainerRegistry,
		Machine:           machineConfig,
		GeneratedSecrets:  secrets,
		UserSSHKeys:       userSSHKeys,
	}

	// Render complete per-machine template
	machineButanePath := fmt.Sprintf("machines/%s/butane.yaml.tmpl", machineConfig.Name)
	rendered, err := r.renderPureYAMLTemplate(machineButanePath, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to render machine butane template %s: %w", machineButanePath, err)
	}

	return rendered, nil
}

func (r *Renderer) renderTemplate(templatePath string, data TemplateData) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	return r.renderTemplateString(string(content), data)
}

func (r *Renderer) renderTemplateString(templateContent string, data TemplateData) (string, error) {
	// Create template with custom functions
	tmpl := template.New("butane").Funcs(r.getTemplateFuncs())

	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Post-process to convert quoted octal strings back to proper octal notation
	result := r.convertQuotedOctalToOctal(buf.String())

	return result, nil
}

func (r *Renderer) generateMachineSecrets(machineName string) (machine.GeneratedSecrets, error) {
	secrets := machine.GeneratedSecrets{}

	// Generate a general password for the machine
	password, err := generateRandomPassword()
	if err != nil {
		return secrets, fmt.Errorf("failed to generate password: %w", err)
	}
	secrets.Password = password

	// Machine-specific secrets can be added here if needed
	switch machineName {
	case "caddy-work":
		// Add any caddy-specific secrets here if needed
	case "photos":
		// Add any photos-specific secrets here if needed
	}

	return secrets, nil
}

// generateRandomPassword creates a secure random password
func generateRandomPassword() (string, error) {
	return machine.GenerateRandomPassword()
}

func (r *Renderer) renderMachineButane(machineName string, data TemplateData) (string, error) {
	// Try to load machine-specific butane file
	machineButanePath := fmt.Sprintf("machines/%s/butane.yaml", machineName)

	// Check if file exists
	if _, err := os.Stat(machineButanePath); os.IsNotExist(err) {
		// File doesn't exist, return empty string (no machine-specific config)
		return "", nil
	}

	// File exists, render it
	return r.renderTemplate(machineButanePath, data)
}

func (r *Renderer) mergeButaneConfigs(base, machine, workload string) (string, error) {
	// Parse base configuration while preserving key order
	var baseConfig yaml.Node
	if err := yaml.Unmarshal([]byte(base), &baseConfig); err != nil {
		return "", fmt.Errorf("failed to parse base config: %w", err)
	}

	// Merge machine-specific configuration if it exists
	if machine != "" {
		var machineConfig yaml.Node
		if err := yaml.Unmarshal([]byte(machine), &machineConfig); err != nil {
			return "", fmt.Errorf("failed to parse machine config: %w", err)
		}
		if err := r.mergeYAMLNodes(&baseConfig, &machineConfig); err != nil {
			return "", fmt.Errorf("failed to merge machine config: %w", err)
		}
	}

	// Merge workload configuration if it exists
	if workload != "" {
		var workloadConfig yaml.Node
		if err := yaml.Unmarshal([]byte(workload), &workloadConfig); err != nil {
			return "", fmt.Errorf("failed to parse workload config: %w", err)
		}
		if err := r.mergeYAMLNodes(&baseConfig, &workloadConfig); err != nil {
			return "", fmt.Errorf("failed to merge workload config: %w", err)
		}
	}

	// Convert back to YAML preserving the original order
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&baseConfig); err != nil {
		return "", fmt.Errorf("failed to marshal merged config: %w", err)
	}
	encoder.Close()

	// Post-process to convert quoted octal strings back to proper octal notation
	processedResult := r.convertQuotedOctalToOctal(buf.String())

	return processedResult, nil
}

// mergeYAMLNodes merges YAML nodes while preserving key order from the base document
func (r *Renderer) mergeYAMLNodes(base, override *yaml.Node) error {
	// Only merge document nodes and mapping nodes
	if base.Kind == yaml.DocumentNode && override.Kind == yaml.DocumentNode {
		if len(base.Content) > 0 && len(override.Content) > 0 {
			return r.mergeYAMLNodes(base.Content[0], override.Content[0])
		}
		return nil
	}

	if base.Kind != yaml.MappingNode || override.Kind != yaml.MappingNode {
		return nil
	}

	// Create a map of keys from override for quick lookup
	overrideMap := make(map[string]*yaml.Node)
	for i := 0; i < len(override.Content); i += 2 {
		key := override.Content[i].Value
		value := override.Content[i+1]
		overrideMap[key] = value
	}

	// Merge existing keys and add new ones while preserving base order
	for i := 0; i < len(base.Content); i += 2 {
		key := base.Content[i].Value
		baseValue := base.Content[i+1]

		if overrideValue, exists := overrideMap[key]; exists {
			// If both are mapping nodes, merge recursively
			if baseValue.Kind == yaml.MappingNode && overrideValue.Kind == yaml.MappingNode {
				if err := r.mergeYAMLNodes(baseValue, overrideValue); err != nil {
					return err
				}
			} else if baseValue.Kind == yaml.SequenceNode && overrideValue.Kind == yaml.SequenceNode {
				// For sequences, append override items to base
				baseValue.Content = append(baseValue.Content, overrideValue.Content...)
			} else {
				// For other types, override takes precedence
				*baseValue = *overrideValue
			}
			// Remove from override map since we've processed it
			delete(overrideMap, key)
		}
	}

	// Add any remaining keys from override
	for key, value := range overrideMap {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		}
		base.Content = append(base.Content, keyNode, value)
	}

	return nil
}

// mergeYAMLMaps deeply merges two YAML maps, with values from 'override' taking precedence
func (r *Renderer) mergeYAMLMaps(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all keys from base
	for key, value := range base {
		result[key] = value
	}

	// Merge/override with values from override
	for key, overrideValue := range override {
		if baseValue, exists := result[key]; exists {
			// If both values are maps, merge them recursively
			if baseMap, ok := baseValue.(map[string]interface{}); ok {
				if overrideMap, ok := overrideValue.(map[string]interface{}); ok {
					result[key] = r.mergeYAMLMaps(baseMap, overrideMap)
					continue
				}
			}
			// If both values are slices, append them
			if baseSlice, ok := baseValue.([]interface{}); ok {
				if overrideSlice, ok := overrideValue.([]interface{}); ok {
					result[key] = append(baseSlice, overrideSlice...)
					continue
				}
			}
		}
		// For all other cases, override takes precedence
		result[key] = overrideValue
	}

	return result
}

func (r *Renderer) renderPureYAMLTemplate(templatePath string, data TemplateData) (string, error) {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", templatePath, err)
	}

	// Use Go template engine instead of string replacement
	return r.renderTemplateString(string(content), data)
}

func (r *Renderer) convertQuotedOctalToOctal(yamlContent string) string {
	// Convert quoted octal strings like mode: "0755" back to unquoted octal notation mode: 0755
	// This preserves the octal representation in the final YAML output
	re := regexp.MustCompile(`mode:\s+"(0[0-7]{3})"`)
	return re.ReplaceAllString(yamlContent, "mode: $1")
}

func (r *Renderer) mergeButane(base, overlay string) (string, error) {
	// For now, simple concatenation
	// In a real implementation, you might want to parse YAML and merge properly
	return base + "\n" + overlay, nil
}
