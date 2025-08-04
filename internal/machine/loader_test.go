package machine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigLoader_GetMachine(t *testing.T) {
	tests := []struct {
		name          string
		machines      []Config
		machineName   string
		expectedError error
		shouldFind    bool
	}{
		{
			name: "existing machine",
			machines: []Config{
				{Name: "test-machine", FQDN: "test-machine.example.com", NetworkInterface: "eth0"},
				{Name: "another-machine", FQDN: "another-machine.example.com", NetworkInterface: "ens18"},
			},
			machineName: "test-machine",
			shouldFind:  true,
		},
		{
			name: "non-existent machine",
			machines: []Config{
				{Name: "test-machine", FQDN: "test-machine.example.com"},
			},
			machineName:   "missing-machine",
			expectedError: ErrMachineNotFound,
			shouldFind:    false,
		},
		{
			name:          "empty machine list",
			machines:      []Config{},
			machineName:   "any-machine",
			expectedError: ErrMachineNotFound,
			shouldFind:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loader := &ConfigLoader{
				machines: MachineList{Machines: tt.machines},
			}

			machine, err := loader.GetMachine(tt.machineName)

			if tt.shouldFind {
				assert.NoError(t, err)
				assert.Equal(t, tt.machineName, machine.Name)
				// Verify NetworkInterface is preserved
				if tt.machineName == "test-machine" {
					assert.Equal(t, "eth0", machine.NetworkInterface)
				} else if tt.machineName == "another-machine" {
					assert.Equal(t, "ens18", machine.NetworkInterface)
				}
			} else {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
				assert.Equal(t, Config{}, machine)
			}
		})
	}
}

func TestConfigLoader_LoadMachinesWithNetworkInterface(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	machinesDir := filepath.Join(tempDir, "machines")
	require.NoError(t, os.MkdirAll(machinesDir, 0755))

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	// Create machine directories and config files
	machines := []struct {
		name      string
		config    Config
		tomlExtra string
	}{
		{
			name: "proxmox-vm",
			config: Config{
				Name:             "proxmox-vm",
				FQDN:             "proxmox-vm.example.com",
				MACAddress:       "02:05:56:12:34:56",
				NetworkInterface: "ens18",
				ContainerTag:     "latest",
			},
		},
		{
			name: "bare-metal",
			config: Config{
				Name:             "bare-metal",
				FQDN:             "bare-metal.example.com",
				MACAddress:       "02:05:56:78:90:12",
				NetworkInterface: "enp0s31f6",
			},
		},
		{
			name: "default-interface",
			config: Config{
				Name:       "default-interface",
				FQDN:       "default.example.com",
				MACAddress: "02:05:56:34:56:78",
				// NetworkInterface intentionally left empty to test default behavior
			},
		},
	}

	// Create machine directories and files
	for _, machine := range machines {
		machineDir := filepath.Join(machinesDir, machine.name)
		require.NoError(t, os.MkdirAll(machineDir, 0755))

		machineContent := fmt.Sprintf(`name = "%s"
fqdn = "%s"
container_image = "registry.example.com/%s"
container_tag = "%s"`,
			machine.config.Name,
			machine.config.FQDN,
			machine.config.Name,
			"latest")

		if machine.config.MACAddress != "" {
			machineContent += fmt.Sprintf(`
mac_address = "%s"`, machine.config.MACAddress)
		}

		if machine.config.NetworkInterface != "" {
			machineContent += fmt.Sprintf(`
network_interface = "%s"`, machine.config.NetworkInterface)
		}

		machineFile := filepath.Join(machineDir, "machine.toml")
		require.NoError(t, os.WriteFile(machineFile, []byte(machineContent), 0644))
	}

	// Create loader and load machines from new structure
	loader := NewConfigLoader()
	require.NoError(t, loader.LoadMachines())

	// Verify all machines and their NetworkInterface fields are loaded correctly
	for _, expectedMachine := range machines {
		loadedMachine, err := loader.GetMachine(expectedMachine.config.Name)
		assert.NoError(t, err)
		assert.Equal(t, expectedMachine.config.Name, loadedMachine.Name)
		assert.Equal(t, expectedMachine.config.FQDN, loadedMachine.FQDN)
		assert.Equal(t, expectedMachine.config.MACAddress, loadedMachine.MACAddress)
		assert.Equal(t, expectedMachine.config.NetworkInterface, loadedMachine.NetworkInterface)
		assert.Equal(t, "latest", loadedMachine.ContainerTag)
	}

}

func TestConfigLoader_RemoveMachine_NewStructure(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	machinesDir := filepath.Join(tempDir, "machines")
	require.NoError(t, os.MkdirAll(machinesDir, 0755))

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tempDir))
	t.Cleanup(func() {
		os.Chdir(originalDir)
	})

	// Create machine directories
	testMachineDir := filepath.Join(machinesDir, "test-machine")
	require.NoError(t, os.MkdirAll(testMachineDir, 0755))

	machineContent := `name = "test-machine"
fqdn = "test-machine.example.com"
container_image = "registry.example.com/test-machine"
container_tag = "latest"
`
	machineFile := filepath.Join(testMachineDir, "machine.toml")
	require.NoError(t, os.WriteFile(machineFile, []byte(machineContent), 0644))

	tests := []struct {
		name          string
		machineName   string
		expectedError error
	}{
		{
			name:        "remove existing machine",
			machineName: "test-machine",
		},
		{
			name:          "remove non-existent machine",
			machineName:   "missing-machine",
			expectedError: ErrMachineNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recreate test machine directory if needed
			if tt.machineName == "test-machine" {
				require.NoError(t, os.MkdirAll(testMachineDir, 0755))
				require.NoError(t, os.WriteFile(machineFile, []byte(machineContent), 0644))
			}

			loader := NewConfigLoader()
			require.NoError(t, loader.LoadMachines())

			err := loader.RemoveMachine(tt.machineName)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.expectedError))
			} else {
				assert.NoError(t, err)
				// Verify directory was removed
				_, err := os.Stat(filepath.Join(machinesDir, tt.machineName))
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}
