package machine

type Config struct {
	Name             string `toml:"name"`
	MACAddress       string `toml:"mac_address,omitempty"`
	NetworkInterface string `toml:"network_interface,omitempty"`
	FQDN             string `toml:"fqdn"`
	ContainerImage   string `toml:"container_image,omitempty"`
	ContainerTag     string `toml:"container_tag,omitempty"`
}

type MachineList struct {
	Machines []Config `toml:"machines"`
}
