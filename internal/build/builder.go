package build

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andreweick/iago/internal/butane"
	"github.com/andreweick/iago/internal/machine"
	"github.com/andreweick/iago/internal/workload"
	"github.com/coreos/butane/config"
	"github.com/coreos/butane/config/common"
	ignitionConfig "github.com/coreos/ignition/v2/config"
)

type Builder struct {
	loader   *machine.ConfigLoader
	renderer *butane.Renderer
	registry *workload.Registry
}

type BuildOptions struct {
	OutputDir  string
	StrictMode bool // Enable strict validation (treat warnings as errors)
}

func NewBuilder() (*Builder, error) {
	loader := machine.NewConfigLoader()
	if err := loader.LoadAll(); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Convert machine workload definitions to workload definitions
	workloadDefs := make([]workload.WorkloadDefinition, len(loader.GetWorkloads()))
	for i, def := range loader.GetWorkloads() {
		workloadDefs[i] = workload.WorkloadDefinition{
			Name:           def.Name,
			ContainerImage: def.ContainerImage,
			ContainerTag:   def.ContainerTag,
			BootcSource:    def.BootcSource,
		}
	}
	registry := workload.CreateDefaultRegistry(workloadDefs)
	renderer := butane.NewRenderer(loader.GetDefaults(), registry)

	return &Builder{
		loader:   loader,
		renderer: renderer,
		registry: registry,
	}, nil
}

func (b *Builder) BuildAll(opts BuildOptions) error {
	machines := b.loader.GetMachines()

	if len(machines) == 0 {
		fmt.Println("No machines to build")
		return nil
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("Building %d machine(s)...\n", len(machines))

	for _, machine := range machines {
		outputFile := filepath.Join(opts.OutputDir, machine.Name+".ign")
		if err := b.GenerateMachineWithOptions(machine.Name, outputFile, opts.StrictMode); err != nil {
			fmt.Printf("✗ %s - %v\n", machine.Name, err)
			continue
		}
		fmt.Printf("✓ %s\n", machine.Name)
	}

	fmt.Printf("\nGenerated %d ignition files in %s\n", len(machines), opts.OutputDir)
	b.printSecretInstructions(machines)

	return nil
}

func (b *Builder) GenerateMachine(machineName, outputFile string) error {
	return b.GenerateMachineWithOptions(machineName, outputFile, false)
}

func (b *Builder) GenerateMachineWithOptions(machineName, outputFile string, strictMode bool) error {
	machineConfig, err := b.loader.GetMachine(machineName)
	if err != nil {
		if errors.Is(err, machine.ErrMachineNotFound) {
			// Get list of available machines for a helpful error message
			availableMachines := b.loader.GetMachines()
			if len(availableMachines) > 0 {
				machineNames := make([]string, len(availableMachines))
				for i, m := range availableMachines {
					machineNames[i] = m.Name
				}
				return fmt.Errorf("machine '%s' not found. Available machines: %s", machineName, strings.Join(machineNames, ", "))
			}
			return fmt.Errorf("machine '%s' not found. No machines configured. Use 'iago init %s' to create it", machineName, machineName)
		}
		return fmt.Errorf("failed to get machine: %w", err)
	}

	// Create a default workload implementation for the machine
	workloadImpl := b.registry.GetDefault(machineConfig.Name)

	workloadConfig := workload.MachineConfig{
		Name:         machineConfig.Name,
		MACAddress:   machineConfig.MACAddress,
		FQDN:         machineConfig.FQDN,
		ContainerTag: machineConfig.ContainerTag,
	}

	if err := workloadImpl.Validate(workloadConfig); err != nil {
		return fmt.Errorf("workload validation failed: %w", err)
	}

	// Render butane configuration
	butaneConfig, err := b.renderer.RenderMachine(machineConfig)
	if err != nil {
		return fmt.Errorf("failed to render butane: %w", err)
	}

	// Save combined butane YAML for debugging
	butaneDebugFile := filepath.Join(filepath.Dir(outputFile), machineConfig.Name+"-final-butane.yaml")
	if err := os.WriteFile(butaneDebugFile, []byte(butaneConfig), 0644); err != nil {
		fmt.Printf("Warning: Could not write debug butane file %s: %v\n", butaneDebugFile, err)
	}

	// Convert butane YAML to ignition JSON
	ignitionConfig, err := b.convertButaneToIgnition([]byte(butaneConfig), strictMode)
	if err != nil {
		return fmt.Errorf("failed to convert butane to ignition: %w", err)
	}

	// Validate the generated ignition configuration
	if err := b.ValidateIgnitionConfig(ignitionConfig); err != nil {
		return fmt.Errorf("failed to validate generated ignition config: %w", err)
	}

	// Write ignition JSON to output file
	if err := os.WriteFile(outputFile, ignitionConfig, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// convertButaneToIgnition converts a Butane YAML configuration to Ignition JSON
func (b *Builder) convertButaneToIgnition(butaneYAML []byte, strictMode bool) ([]byte, error) {
	// Configure translation options with strict validation
	options := common.TranslateBytesOptions{
		TranslateOptions: common.TranslateOptions{
			FilesDir:                  "config/scripts", // Point to scripts directory for local: directive
			NoResourceAutoCompression: false,            // Allow automatic compression
			DebugPrintTranslations:    false,            // No debug output
		},
		Pretty: true,  // Pretty-print the JSON output (equivalent to --pretty)
		Raw:    false, // Include any wrapper, not just the Ignition config
	}

	// Use the butane config package to translate from Butane to Ignition
	ignitionJSON, report, err := config.TranslateBytes(butaneYAML, options)
	if err != nil {
		return nil, fmt.Errorf("failed to translate butane config: %w", err)
	}

	// Handle warnings and errors based on strict mode
	if len(report.Entries) > 0 {
		var warnings []string
		var errors []string

		for _, entry := range report.Entries {
			entryStr := entry.String()
			if entry.Kind.IsFatal() {
				errors = append(errors, entryStr)
			} else {
				// Treat non-fatal entries as warnings
				warnings = append(warnings, entryStr)
			}
		}

		// Always fail on errors
		if len(errors) > 0 {
			return nil, fmt.Errorf("butane translation failed with errors: %s", strings.Join(errors, ", "))
		}

		// In strict mode, fail on any warnings
		if strictMode && len(warnings) > 0 {
			fmt.Printf("Butane translation warnings (strict mode): %s\n", strings.Join(warnings, ", "))
			return nil, fmt.Errorf("butane translation failed in strict mode due to warnings: %s", strings.Join(warnings, ", "))
		}

		// In non-strict mode, just log warnings
		if len(warnings) > 0 {
			fmt.Printf("Butane translation warnings: %s\n", strings.Join(warnings, ", "))
		}
	}

	// Check for fatal errors
	if report.IsFatal() {
		return nil, fmt.Errorf("butane translation failed with fatal errors: %s", report.String())
	}

	return ignitionJSON, nil
}

// ValidateIgnitionConfig validates a generated ignition JSON configuration
func (b *Builder) ValidateIgnitionConfig(ignitionJSON []byte) error {
	// Parse and validate the ignition configuration
	_, report, err := ignitionConfig.Parse(ignitionJSON)
	if err != nil {
		return fmt.Errorf("ignition parse error: %w", err)
	}

	// Check for fatal validation errors
	if report.IsFatal() {
		return fmt.Errorf("ignition validation failed: %s", report.String())
	}

	// Log warnings but don't fail the build
	if len(report.Entries) > 0 {
		var warnings []string
		for _, entry := range report.Entries {
			if !entry.Kind.IsFatal() {
				warnings = append(warnings, entry.String())
			}
		}
		if len(warnings) > 0 {
			fmt.Printf("Ignition validation warnings: %s\n", strings.Join(warnings, ", "))
		}
	}

	return nil
}

func (b *Builder) printSecretInstructions(machines []machine.Config) {
	hasSecrets := false
	for _, machine := range machines {
		workloadImpl := b.registry.GetDefault(machine.Name)
		if len(workloadImpl.GetSecrets()) > 0 {
			hasSecrets = true
			break
		}
	}

	if !hasSecrets {
		return
	}

	fmt.Println("\nSecrets generated for:")
	for _, machine := range machines {
		workloadImpl := b.registry.GetDefault(machine.Name)
		secrets := workloadImpl.GetSecrets()
		if len(secrets) > 0 {
			fmt.Printf("- %s: SSH to %s, check /etc/iago/secrets/\n", machine.Name, machine.FQDN)
		}
	}
}
