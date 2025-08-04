package workload

type MachineConfig struct {
	Name             string `toml:"name"`
	MACAddress       string `toml:"mac_address,omitempty"`
	NetworkInterface string `toml:"network_interface,omitempty"`
	FQDN             string `toml:"fqdn"`
	ContainerImage   string `toml:"container_image,omitempty"`
	ContainerTag     string `toml:"container_tag,omitempty"`
}

type Workload interface {
	Name() string
	ContainerImage() string
	ContainerTag() string
	GetButaneOverlay() (string, error)
	GetSecrets() []SecretDefinition
	Validate(config MachineConfig) error
}

type SecretDefinition struct {
	Name        string
	Path        string
	Description string
	Generator   func() (string, error)
}

type Definition struct {
	Name           string `toml:"name"`
	ContainerImage string `toml:"container_image"`
	ContainerTag   string `toml:"container_tag"`
	BootcSource    string `toml:"bootc_source"`
}
