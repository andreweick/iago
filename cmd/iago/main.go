package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/andreweick/iago/internal/auth"
	"github.com/andreweick/iago/internal/build"
	"github.com/andreweick/iago/internal/container"
	"github.com/andreweick/iago/internal/github"
	"github.com/andreweick/iago/internal/machine"
	"github.com/andreweick/iago/internal/scaffold"
	"github.com/andreweick/iago/internal/workload"
	"github.com/urfave/cli/v2"
)

// exitWithError prints an error message and exits with the given code
// This avoids the cli.Exit() wrapper which causes 'just' to add its own error message
func exitWithError(message string, code int) error {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(code)
	return nil // never reached
}

// getContainerBuildHelpText returns formatted help text with current registry info
func getContainerBuildHelpText() string {
	baseHelp := `Build and push container for workload.

Format: {registry}/{workload-name}:{tag}
Example: ghcr.io/andreweick/my-app:latest

Flags:
  --all              Build all workloads
  --local, -l        Push to local registry (localhost:5000)
  --no-push          Build in memory only for testing, don't push to registry
  --sign             Sign container with cosign after building
  --tag value        Override default tag (default: "latest")
  --token value      Registry token/password for authentication`

	// Try to load current registry from defaults.toml
	loader := machine.NewConfigLoader()
	if err := loader.LoadDefaults(); err == nil {
		defaults := loader.GetDefaults()
		if defaults.ContainerRegistry.URL != "" {
			baseHelp += fmt.Sprintf("\n\nWith the current config as set in defaults.toml, the registry is: %s", defaults.ContainerRegistry.URL)
		}
	}

	return baseHelp
}

func main() {
	app := &cli.App{
		Name:  "iago",
		Usage: "Fedora CoreOS machine management with bootc containers",
		Description: `Iago helps you create, manage, and update Fedora CoreOS machines
   with bootc containers for your homelab and VPS infrastructure.`,
		Version: "1.0.0",
		Commands: []*cli.Command{
			{
				Name:      "init",
				Aliases:   []string{"i"},
				Usage:     "Initialize a new machine: create config, container scaffold, and ignition file",
				ArgsUsage: "[machine-name]",
				Action:    initCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "domain",
						Aliases: []string{"d"},
						Value:   "spouterinn.org",
						Usage:   "Domain suffix for FQDN",
					},
					&cli.BoolFlag{
						Name:  "generate-mac",
						Value: true,
						Usage: "Generate MAC address for homelab machines",
					},
					&cli.BoolFlag{
						Name:    "machine-only",
						Aliases: []string{"m"},
						Usage:   "Create only machine configuration (config, template, ignition)",
					},
					&cli.BoolFlag{
						Name:    "container-only",
						Aliases: []string{"c"},
						Usage:   "Create only container scaffold (directory, Containerfile, prompt)",
					},
				},
			},
			{
				Name:    "list",
				Aliases: []string{"ls"},
				Usage:   "List all configured machines",
				Action:  listCommand,
			},
			{
				Name:      "rm",
				Aliases:   []string{"remove", "delete"},
				Usage:     "Remove a machine and all its associated files",
				ArgsUsage: "[machine-name]",
				Action:    removeCommand,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Skip confirmation prompt",
					},
				},
			},
			{
				Name:      "ignite",
				Aliases:   []string{"gen"},
				Usage:     "Generate ignition file for an existing machine",
				ArgsUsage: "[machine-name]",
				Action:    igniteCommand,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output file path (defaults to output/ignition/<machine-name>.ign)",
					},
					&cli.BoolFlag{
						Name:    "strict",
						Aliases: []string{"s"},
						Value:   true,
						Usage:   "Enable strict mode (treat warnings as errors)",
					},
				},
			},
			{
				Name:    "validate",
				Aliases: []string{"val"},
				Usage:   "Validate configuration",
				Action:  validateCommand,
			},
			{
				Name:      "build",
				Usage:     getContainerBuildHelpText(),
				ArgsUsage: "[workload-name]",
				Action:    containerBuildCommand,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "local",
						Aliases: []string{"l"},
						Usage:   "Push to local registry (localhost:5000)",
					},
					&cli.BoolFlag{
						Name:  "no-push",
						Usage: "Build in memory only for testing, don't push to registry (image is discarded after build)",
					},
					&cli.BoolFlag{
						Name:  "sign",
						Usage: "Sign container with cosign after building",
					},
					&cli.StringFlag{
						Name:  "tag",
						Usage: "Override default tag",
						Value: "latest",
					},
					&cli.BoolFlag{
						Name:  "all",
						Usage: "Build all workloads",
					},
					&cli.StringFlag{
						Name:  "token",
						Usage: "Registry token/password for authentication (takes precedence over env vars and 1Password)",
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		// Don't print anything here - errors are already handled in commands
		os.Exit(1)
	}
}

func initCommand(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return exitWithError("Error: requires exactly one argument (machine name). Usage: iago init [flags] [machine-name]", 1)
	}

	machineName := ctx.Args().Get(0)
	domain := ctx.String("domain")
	generateMAC := ctx.Bool("generate-mac")
	machineOnly := ctx.Bool("machine-only")
	containerOnly := ctx.Bool("container-only")

	// Validate flag combinations
	if machineOnly && containerOnly {
		return exitWithError("Error: --machine-only and --container-only flags are mutually exclusive", 1)
	}

	// Load defaults to get MAC prefix
	loader := machine.NewConfigLoader()
	if err := loader.LoadDefaults(); err != nil {
		return exitWithError(fmt.Sprintf("Error loading defaults: %v", err), 1)
	}

	defaults := loader.GetDefaults()

	// Generate MAC if requested (only needed for machine config)
	var macAddress string
	if generateMAC && !containerOnly {
		var err error
		macAddress, err = machine.GenerateMAC(machine.DefaultMACPrefix)
		if err != nil {
			return exitWithError(fmt.Sprintf("Error generating MAC address: %v", err), 1)
		}
	}

	// Generate FQDN using machine name (only needed for machine config)
	var fqdn string
	if !containerOnly {
		fqdn = fmt.Sprintf("%s.%s", machineName, domain)
	}

	// Create scaffolder
	scaffolder := scaffold.NewScaffolder(defaults)

	// Prepare scaffold options
	opts := scaffold.ScaffoldOptions{
		MachineName: machineName,
		FQDN:        fqdn,
		MACAddress:  macAddress,
		OutputDir:   "output/ignition",
	}

	// Display what will be created
	fmt.Printf("Initializing machine: %s\n", machineName)
	if !containerOnly {
		fmt.Printf("  FQDN: %s\n", fqdn)
		if macAddress != "" {
			fmt.Printf("  MAC Address: %s\n", macAddress)
		}
	}

	fmt.Printf("\nCreating:\n")

	var err error
	switch {
	case containerOnly:
		fmt.Printf("  ✓ Container scaffold: containers/%s/\n", machineName)
		err = scaffolder.CreateContainerScaffoldOnly(opts)
	case machineOnly:
		fmt.Printf("  ✓ Machine config: machines/%s/machine.toml\n", machineName)
		fmt.Printf("  ✓ Butane template: machines/%s/butane.yaml.tmpl\n", machineName)
		fmt.Printf("  ✓ Ignition file: output/ignition/%s.ign\n", machineName)
		err = scaffolder.CreateMachineConfigOnly(opts)
	default:
		// Default behavior: create both
		fmt.Printf("  ✓ Container scaffold: containers/%s/\n", machineName)
		fmt.Printf("  ✓ Machine config: machines/%s/machine.toml\n", machineName)
		fmt.Printf("  ✓ Butane template: machines/%s/butane.yaml.tmpl\n", machineName)
		fmt.Printf("  ✓ Ignition file: output/ignition/%s.ign\n", machineName)
		err = scaffolder.CreateMachineScaffold(opts)
	}

	if err != nil {
		return exitWithError(fmt.Sprintf("Error creating scaffold: %v", err), 1)
	}

	// Generate ignition file (only for machine-only and default modes)
	if !containerOnly {
		builder, err := build.NewBuilder()
		if err != nil {
			fmt.Printf("Warning: Could not generate ignition file: %v\n", err)
		} else {
			outputFile := fmt.Sprintf("output/ignition/%s.ign", machineName)
			if err := builder.GenerateMachine(machineName, outputFile); err != nil {
				fmt.Printf("Warning: Could not generate ignition file: %v\n", err)
			}
		}
	}

	// Success message and next steps
	fmt.Printf("\n🎉 Machine '%s' initialized successfully!\n", machineName)
	fmt.Printf("\nNext steps:\n")

	switch {
	case containerOnly:
		fmt.Printf("1. Customize the container: containers/%s/\n", machineName)
		fmt.Printf("2. Build the container: iago container build %s\n", machineName)
		fmt.Printf("3. Create machine config: iago init --machine-only %s\n", machineName)
	case machineOnly:
		fmt.Printf("1. Create container scaffold: iago init --container-only %s\n", machineName)
		fmt.Printf("2. Build the container: iago container build %s\n", machineName)
		fmt.Printf("3. Use ignition file: output/ignition/%s.ign\n", machineName)
	default:
		fmt.Printf("1. Customize the container: containers/%s/\n", machineName)
		fmt.Printf("2. Build the container: iago container build %s\n", machineName)
		fmt.Printf("3. Use ignition file: output/ignition/%s.ign\n", machineName)
	}

	return nil
}

func listCommand(ctx *cli.Context) error {
	loader := machine.NewConfigLoader()
	if err := loader.LoadMachines(); err != nil {
		return exitWithError(fmt.Sprintf("Error loading machines: %v", err), 1)
	}

	machines := loader.GetMachines()
	if len(machines) == 0 {
		fmt.Println("No machines configured")
		return nil
	}

	fmt.Printf("%-18s %-27s %-18s %-12s\n", "NAME", "FQDN", "MAC ADDRESS", "INTERFACE")
	fmt.Println("--------------------------------------------------------------------------------------")
	for _, machine := range machines {
		macAddress := machine.MACAddress
		if macAddress == "" {
			macAddress = "-"
		}
		networkInterface := machine.NetworkInterface
		if networkInterface == "" {
			networkInterface = "-"
		}
		fmt.Printf("%-18s %-27s %-18s %-12s\n",
			machine.Name,
			machine.FQDN,
			macAddress,
			networkInterface)
	}

	return nil
}

func igniteCommand(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return exitWithError("Error: requires exactly one argument (machine name). Usage: iago ignite [flags] [machine-name]", 1)
	}

	machineName := ctx.Args().Get(0)
	outputFile := ctx.String("output")

	// If no output file specified, use machine name
	if outputFile == "" {
		outputFile = fmt.Sprintf("output/ignition/%s.ign", machineName)
	}

	builder, err := build.NewBuilder()
	if err != nil {
		return exitWithError(fmt.Sprintf("Error creating builder: %v", err), 1)
	}

	strictMode := ctx.Bool("strict")
	if err := builder.GenerateMachineWithOptions(machineName, outputFile, strictMode); err != nil {
		return exitWithError(fmt.Sprintf("Error generating machine: %v", err), 1)
	}

	fmt.Printf("Generated ignition for %s -> %s\n", machineName, outputFile)
	return nil
}

func validateCommand(ctx *cli.Context) error {
	loader := machine.NewConfigLoader()
	if err := loader.LoadAll(); err != nil {
		return exitWithError(fmt.Sprintf("Configuration validation failed: %v", err), 1)
	}

	// Validate workload definitions
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

	machines := loader.GetMachines()
	hasErrors := false

	for _, m := range machines {
		// Get default workload implementation
		workloadImpl := registry.GetDefault(m.Name)

		// Validate naming consistency
		if !strings.HasPrefix(m.FQDN, m.Name+".") {
			fmt.Fprintf(os.Stderr, "Machine %s: FQDN '%s' should start with machine name '%s.'\n",
				m.Name, m.FQDN, m.Name)
			hasErrors = true
		}

		// Check MAC address format if present
		if m.MACAddress != "" && !machine.ValidateMAC(m.MACAddress) {
			fmt.Fprintf(os.Stderr, "Machine %s: Invalid MAC address format: %s\n",
				m.Name, m.MACAddress)
			hasErrors = true
		}

		// Validate workload-specific requirements
		workloadConfig := workload.MachineConfig{
			Name:         m.Name,
			MACAddress:   m.MACAddress,
			FQDN:         m.FQDN,
			ContainerTag: m.ContainerTag,
		}

		if err := workloadImpl.Validate(workloadConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Machine %s validation failed: %v\n", m.Name, err)
			hasErrors = true
		}
	}

	// Validate base Butane template contains required constants
	if err := validateBaseButaneTemplate(); err != nil {
		fmt.Fprintf(os.Stderr, "Base Butane template validation failed: %v\n", err)
		hasErrors = true
	}

	// Validate script files
	if err := validateScriptFiles(); err != nil {
		fmt.Fprintf(os.Stderr, "Script files validation failed: %v\n", err)
		hasErrors = true
	}

	// Validate template local references
	if err := validateTemplateLocalReferences(); err != nil {
		fmt.Fprintf(os.Stderr, "Template local references validation failed: %v\n", err)
		hasErrors = true
	}

	// Validate GitHub SSH keys if configured
	defaults := loader.GetDefaults()
	if defaults.User.GitHubUsername != "" {
		fmt.Printf("Validating GitHub SSH keys for user '%s'...\n", defaults.User.GitHubUsername)
		keys, err := github.FetchSSHKeys(defaults.User.GitHubUsername)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch SSH keys from GitHub: %v\n", err)
			hasErrors = true
		} else {
			fmt.Printf("Found %d SSH key(s) for GitHub user '%s'\n", len(keys), defaults.User.GitHubUsername)
		}
	}

	if hasErrors {
		return exitWithError("Configuration validation failed", 1)
	}

	fmt.Println("Configuration is valid")
	return nil
}

func validateBaseButaneTemplate() error {
	// With the new per-machine template system, we validate that machines directory exists
	// and contains at least one valid machine template
	const machinesDir = "machines"

	// Check if machines directory exists
	if _, err := os.Stat(machinesDir); os.IsNotExist(err) {
		return fmt.Errorf("machines directory '%s' does not exist", machinesDir)
	}

	// Read machines directory to find machine subdirectories
	entries, err := os.ReadDir(machinesDir)
	if err != nil {
		return fmt.Errorf("failed to read machines directory: %w", err)
	}

	foundValidTemplate := false
	for _, entry := range entries {
		if entry.IsDir() {
			templatePath := filepath.Join(machinesDir, entry.Name(), "butane.yaml.tmpl")
			if _, err := os.Stat(templatePath); err == nil {
				// Validate this machine template
				if err := validateMachineTemplate(templatePath); err != nil {
					return fmt.Errorf("invalid template %s: %w", templatePath, err)
				}
				foundValidTemplate = true
			}
		}
	}

	if !foundValidTemplate {
		return fmt.Errorf("no valid machine templates found in %s - each machine should have a butane.yaml.tmpl file", machinesDir)
	}

	return nil
}

// validateMachineTemplate validates a specific machine template file
func validateMachineTemplate(templatePath string) error {
	content, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	templateContent := string(content)

	// Check for expected Go template variables
	requiredVariables := []string{
		"{{ .User.Username }}",
		"{{ .Machine.Name }}",
		"{{ .ContainerRegistry.URL }}",
		"variant: fcos", // Ensure it's a valid butane file
	}

	for _, variable := range requiredVariables {
		if !strings.Contains(templateContent, variable) {
			return fmt.Errorf("required template variable '%s' not found", variable)
		}
	}

	return nil
}

// validateScriptFiles checks that all required script files exist and are readable
func validateScriptFiles() error {
	const scriptsDir = "config/scripts"

	// Check if scripts directory exists
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		return fmt.Errorf("scripts directory '%s' does not exist", scriptsDir)
	}

	// List of required script files
	requiredScripts := []string{
		"bootc-update.sh",
		"motd.sh",
	}

	var errors []string

	for _, script := range requiredScripts {
		scriptPath := filepath.Join(scriptsDir, script)

		// Check if file exists
		fileInfo, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			// Check for common misspellings
			commonMisspellings := []string{
				strings.Replace(script, "-", "_", -1),  // bootc_update.sh
				strings.Replace(script, ".sh", "", -1), // bootc-update
			}

			for _, misspelling := range commonMisspellings {
				if misspellingPath := filepath.Join(scriptsDir, misspelling); misspellingPath != scriptPath {
					if _, err := os.Stat(misspellingPath); err == nil {
						errors = append(errors, fmt.Sprintf("script '%s' not found, but found '%s' - check spelling", script, misspelling))
						continue
					}
				}
			}
			errors = append(errors, fmt.Sprintf("required script '%s' not found", script))
			continue
		} else if err != nil {
			errors = append(errors, fmt.Sprintf("error accessing script '%s': %v", script, err))
			continue
		}

		// Check if file is readable
		if fileInfo.Mode().Perm()&0044 == 0 {
			errors = append(errors, fmt.Sprintf("script '%s' is not readable", script))
		}

		// Basic validation - should start with shebang
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("cannot read script '%s': %v", script, err))
			continue
		}

		if !strings.HasPrefix(string(content), "#!") {
			errors = append(errors, fmt.Sprintf("script '%s' should start with shebang (#!)", script))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("script validation errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	fmt.Printf("Script files validation passed (%d files checked)\n", len(requiredScripts))
	return nil
}

// validateTemplateLocalReferences checks that all local: references in templates point to existing files
func validateTemplateLocalReferences() error {
	var templatePaths []string

	// Add machine-specific templates (now using .tmpl extension)
	if entries, err := os.ReadDir("machines"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				machineName := entry.Name()
				templatePath := filepath.Join("machines", machineName, "butane.yaml.tmpl")
				if _, err := os.Stat(templatePath); err == nil {
					templatePaths = append(templatePaths, templatePath)
				}
			}
		}
	}

	// If no machine templates found, that's an error
	if len(templatePaths) == 0 {
		return fmt.Errorf("no machine templates found - each machine should have a butane.yaml.tmpl file")
	}

	var errors []string
	localReferences := make(map[string][]string) // file -> templates that reference it

	for _, templatePath := range templatePaths {
		content, err := os.ReadFile(templatePath)
		if err != nil {
			continue // Skip if template doesn't exist or can't be read
		}

		// Find all local: references
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if strings.Contains(line, "local:") {
				// Extract the filename after "local:"
				parts := strings.Split(line, "local:")
				if len(parts) > 1 {
					filename := strings.TrimSpace(parts[1])
					// Remove quotes if present
					filename = strings.Trim(filename, "\"'")

					if filename != "" {
						localReferences[filename] = append(localReferences[filename], fmt.Sprintf("%s:%d", templatePath, i+1))

						// Check if the referenced file exists
						scriptPath := filepath.Join("config/scripts", filename)
						if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
							errors = append(errors, fmt.Sprintf("template %s:%d references missing file 'config/scripts/%s'", templatePath, i+1, filename))
						}
					}
				}
			}
		}
	}

	// Check for unused script files
	if entries, err := os.ReadDir("config/scripts"); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sh") {
				if _, referenced := localReferences[entry.Name()]; !referenced {
					fmt.Printf("Warning: script file '%s' is not referenced by any template\n", entry.Name())
				}
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("template local reference errors:\n  - %s", strings.Join(errors, "\n  - "))
	}

	if len(localReferences) > 0 {
		fmt.Printf("Template local references validation passed (%d references checked)\n", len(localReferences))
	}
	return nil
}

func removeCommand(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return exitWithError("Error: requires exactly one argument (machine name). Usage: iago rm [flags] [machine-name]", 1)
	}

	machineName := ctx.Args().Get(0)
	force := ctx.Bool("force")

	// Load machines to verify it exists
	loader := machine.NewConfigLoader()
	if err := loader.LoadMachines(); err != nil {
		return exitWithError(fmt.Sprintf("Error loading machines: %v", err), 1)
	}

	// Check if machine exists
	_, err := loader.GetMachine(machineName)
	if err != nil {
		if errors.Is(err, machine.ErrMachineNotFound) {
			fmt.Printf("Machine '%s' not found\n", machineName)
			return nil
		}
		return exitWithError(fmt.Sprintf("Error checking machine: %v", err), 1)
	}

	// Show what will be removed
	fmt.Printf("The following will be removed:\n")
	fmt.Printf("  ✓ Machine config: machines/%s/\n", machineName)
	fmt.Printf("  ✓ Container directory: containers/%s/\n", machineName)
	fmt.Printf("  ✓ Ignition file: output/ignition/%s.ign\n", machineName)

	// Confirm unless force flag is set
	if !force {
		fmt.Printf("\nAre you sure you want to remove machine '%s'? [y/N] ", machineName)
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Aborted")
			return nil
		}
	}

	// Remove from config
	if err := loader.RemoveMachine(machineName); err != nil {
		if errors.Is(err, machine.ErrMachineNotFound) {
			fmt.Printf("Machine '%s' not found in configuration\n", machineName)
			return nil
		}
		return exitWithError(fmt.Sprintf("Error removing machine from config: %v", err), 1)
	}

	// Remove container directory
	containerDir := fmt.Sprintf("containers/%s", machineName)
	if err := os.RemoveAll(containerDir); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: Could not remove container directory: %v\n", err)
	}

	// Remove machine directory
	machineDir := fmt.Sprintf("machines/%s", machineName)
	if err := os.RemoveAll(machineDir); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: Could not remove machine directory: %v\n", err)
	}

	// Remove ignition file
	ignitionFile := fmt.Sprintf("output/ignition/%s.ign", machineName)
	if err := os.Remove(ignitionFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Warning: Could not remove ignition file: %v\n", err)
	}

	fmt.Printf("\n🗑️  Machine '%s' removed successfully!\n", machineName)
	return nil
}

func containerBuildCommand(ctx *cli.Context) error {
	buildAll := ctx.Bool("all")
	local := ctx.Bool("local")
	noPush := ctx.Bool("no-push")
	sign := ctx.Bool("sign")
	tag := ctx.String("tag")
	token := ctx.String("token")

	// Load defaults to get registry configuration
	loader := machine.NewConfigLoader()
	if err := loader.LoadDefaults(); err != nil {
		return exitWithError(fmt.Sprintf("Error loading defaults: %v", err), 1)
	}
	defaults := loader.GetDefaults()

	if buildAll {
		return buildAllWorkloads(ctx, defaults, local, noPush, sign, tag, "", token)
	}

	// Single workload build
	if ctx.NArg() != 1 {
		return exitWithError("Error: requires exactly one argument (workload name) or use --all flag. Usage: iago build [flags] [workload-name]", 1)
	}

	workloadName := ctx.Args().Get(0)
	return buildSingleWorkload(ctx, workloadName, defaults, local, noPush, sign, tag, "", token)
}

func buildSingleWorkload(ctx *cli.Context, workloadName string, defaults machine.Defaults, local, noPush, sign bool, tag, username, token string) error {
	contextPath := fmt.Sprintf("containers/%s", workloadName)

	// Check if container directory exists
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return exitWithError(fmt.Sprintf("Container directory not found: %s\nTip: Run 'iago init %s' first", contextPath, workloadName), 1)
	}

	// Validate local registry if needed
	if local && !noPush {
		if err := container.ValidateLocalRegistry(ctx.Context); err != nil {
			return exitWithError(err.Error(), 1)
		}
	}

	// Get authentication configuration (only if we're pushing)
	var authConfig *container.AuthConfig
	if !noPush {
		authCfg, err := auth.GetAuthConfig(ctx.Context, username, token)
		if err != nil {
			return exitWithError(fmt.Sprintf("Authentication error: %v", err), 1)
		}
		authConfig = authCfg.ToContainerAuthConfig()
		fmt.Printf("Using authentication: %s\n", authCfg.Source)
	}

	// Configure build options
	buildOptions := container.BuildOptions{
		WorkloadName: workloadName,
		ContextPath:  contextPath,
		Tag:          tag,
		RegistryURL:  defaults.ContainerRegistry.URL,
		Local:        local,
		NoPush:       noPush,
		Sign:         sign,
		AuthConfig:   authConfig,
	}

	// Create builder and build
	builder := container.NewBuilder(buildOptions)

	fmt.Printf("Building container for workload: %s\n", workloadName)
	if local {
		fmt.Printf("Target registry: localhost:5000\n")
	} else {
		fmt.Printf("Target registry: %s\n", defaults.ContainerRegistry.URL)
	}

	err := builder.BuildAndPush(ctx.Context)
	if err != nil {
		return exitWithError(fmt.Sprintf("Container build failed: %v", err), 1)
	}

	fmt.Printf("\n✅ Container build completed for %s\n", workloadName)
	return nil
}

func buildAllWorkloads(ctx *cli.Context, defaults machine.Defaults, local, noPush, sign bool, tag, username, token string) error {
	// Find all container directories
	containersDir := "containers"
	entries, err := os.ReadDir(containersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return exitWithError("No containers directory found. Create containers with 'iago init' first", 1)
		}
		return exitWithError(fmt.Sprintf("Error reading containers directory: %v", err), 1)
	}

	workloads := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			workloads = append(workloads, entry.Name())
		}
	}

	if len(workloads) == 0 {
		return exitWithError("No containers found in containers directory", 1)
	}

	// Validate local registry once if needed
	if local && !noPush {
		if err := container.ValidateLocalRegistry(ctx.Context); err != nil {
			return exitWithError(err.Error(), 1)
		}
	}

	fmt.Printf("Building %d workloads: %v\n", len(workloads), workloads)

	// Build each workload
	for _, workload := range workloads {
		fmt.Printf("\n--- Building %s ---\n", workload)
		err := buildSingleWorkload(ctx, workload, defaults, local, noPush, sign, tag, username, token)
		if err != nil {
			fmt.Printf("❌ Failed to build %s: %v\n", workload, err)
			// Continue with other workloads instead of failing completely
			continue
		}
		fmt.Printf("✅ Completed %s\n", workload)
	}

	fmt.Printf("\n🎉 All workload builds completed!\n")
	return nil
}
