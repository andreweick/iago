package machine

import (
	"errors"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// ErrMachineNotFound is returned when a machine cannot be found
var ErrMachineNotFound = errors.New("machine not found")

type ConfigLoader struct {
	defaults  Defaults
	machines  MachineList
	workloads WorkloadList
}

type WorkloadList struct {
	Workloads []WorkloadDefinition `toml:"workloads"`
}

type WorkloadDefinition struct {
	Name           string `toml:"name"`
	ContainerImage string `toml:"container_image"`
	ContainerTag   string `toml:"container_tag"`
	BootcSource    string `toml:"bootc_source"`
}

func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

func (cl *ConfigLoader) LoadAll() error {
	if err := cl.LoadDefaults(); err != nil {
		return fmt.Errorf("failed to load defaults: %w", err)
	}

	if err := cl.LoadMachines(); err != nil {
		return fmt.Errorf("failed to load machines: %w", err)
	}

	if err := cl.LoadWorkloads(); err != nil {
		return fmt.Errorf("failed to load workloads: %w", err)
	}

	return nil
}

func (cl *ConfigLoader) LoadDefaults() error {
	content, err := os.ReadFile("config/defaults.toml")
	if err != nil {
		return fmt.Errorf("failed to read defaults.toml: %w", err)
	}

	if err := toml.Unmarshal(content, &cl.defaults); err != nil {
		return fmt.Errorf("failed to parse defaults.toml: %w", err)
	}

	return nil
}

func (cl *ConfigLoader) LoadMachines() error {
	// Load from machines directory structure
	return cl.loadMachinesFromMachineDirs()
}

func (cl *ConfigLoader) loadMachinesFromMachineDirs() error {
	machineDirs, err := os.ReadDir("machines")
	if err != nil {
		if os.IsNotExist(err) {
			cl.machines.Machines = []Config{}
			return nil
		}
		return fmt.Errorf("failed to read machines directory: %w", err)
	}

	var machines []Config
	for _, dir := range machineDirs {
		if !dir.IsDir() {
			continue
		}

		// Read machine.toml from machines directory
		machinePath := fmt.Sprintf("machines/%s/machine.toml", dir.Name())
		content, err := os.ReadFile(machinePath)
		if err != nil {
			continue // Skip directories without machine.toml
		}

		var machine Config
		if err := toml.Unmarshal(content, &machine); err != nil {
			return fmt.Errorf("failed to parse %s: %w", machinePath, err)
		}
		machines = append(machines, machine)
	}

	cl.machines.Machines = machines
	return nil
}

func (cl *ConfigLoader) LoadWorkloads() error {
	// Load from containers directory structure
	return cl.loadWorkloadsFromContainerDirs()
}

func (cl *ConfigLoader) loadWorkloadsFromContainerDirs() error {
	containerDirs, err := os.ReadDir("containers")
	if err != nil {
		if os.IsNotExist(err) {
			cl.workloads.Workloads = []WorkloadDefinition{}
			return nil
		}
		return fmt.Errorf("failed to read containers directory: %w", err)
	}

	var workloads []WorkloadDefinition
	for _, dir := range containerDirs {
		if !dir.IsDir() || dir.Name() == "_shared" {
			continue
		}

		// For now, create workload definitions based on container directories
		// In the future, we might want to read metadata from the container
		workload := WorkloadDefinition{
			Name:           dir.Name(),
			ContainerImage: fmt.Sprintf("%s/%s", cl.defaults.ContainerRegistry.URL, dir.Name()),
			ContainerTag:   "latest",
			BootcSource:    fmt.Sprintf("containers/%s/", dir.Name()),
		}
		workloads = append(workloads, workload)
	}

	cl.workloads.Workloads = workloads
	return nil
}

func (cl *ConfigLoader) GetDefaults() Defaults {
	return cl.defaults
}

func (cl *ConfigLoader) GetMachines() []Config {
	return cl.machines.Machines
}

func (cl *ConfigLoader) GetWorkloads() []WorkloadDefinition {
	return cl.workloads.Workloads
}

func (cl *ConfigLoader) GetMachine(name string) (Config, error) {
	for _, machine := range cl.machines.Machines {
		if machine.Name == name {
			return machine, nil
		}
	}
	return Config{}, fmt.Errorf("machine '%s': %w", name, ErrMachineNotFound)
}

func (cl *ConfigLoader) RemoveMachine(name string) error {
	// Remove from new machine directory structure
	machineDir := fmt.Sprintf("machines/%s", name)
	if _, err := os.Stat(machineDir); os.IsNotExist(err) {
		return fmt.Errorf("machine '%s': %w", name, ErrMachineNotFound)
	}

	// Remove the entire machine directory
	if err := os.RemoveAll(machineDir); err != nil {
		return fmt.Errorf("failed to remove machine directory: %w", err)
	}

	// Reload machines list
	return cl.LoadMachines()
}

func (cl *ConfigLoader) removeMachineFromOldStructure(name string) error {
	// Find and remove the machine
	newMachines := make([]Config, 0, len(cl.machines.Machines))
	found := false

	for _, machine := range cl.machines.Machines {
		if machine.Name != name {
			newMachines = append(newMachines, machine)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("machine '%s': %w", name, ErrMachineNotFound)
	}

	cl.machines.Machines = newMachines

	// Write back to file
	file, err := os.Create("config/machines.toml")
	if err != nil {
		return fmt.Errorf("failed to open machines.toml for writing: %w", err)
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(cl.machines); err != nil {
		return fmt.Errorf("failed to write machines.toml: %w", err)
	}

	return nil
}
