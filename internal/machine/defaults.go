package machine

type Defaults struct {
	User              UserConfig              `toml:"user"`
	Admin             AdminConfig             `toml:"admin"`
	Network           NetworkConfig           `toml:"network"`
	Updates           UpdateConfig            `toml:"updates"`
	Bootc             BootcConfig             `toml:"bootc"`
	ContainerRegistry ContainerRegistryConfig `toml:"container_registry"`
}

type UserConfig struct {
	Username       string   `toml:"username"`
	GitHubUsername string   `toml:"github_username"`
	Groups         []string `toml:"groups"`
	PasswordHash   string   `toml:"password_hash"`
}

type AdminConfig struct {
	Username     string   `toml:"username"`
	Groups       []string `toml:"groups"`
	PasswordHash string   `toml:"password_hash"`
}

type NetworkConfig struct {
	DNSServers              []string `toml:"dns_servers"`
	Timezone                string   `toml:"timezone"`
	DefaultNetworkInterface string   `toml:"default_network_interface"`
}

type UpdateConfig struct {
	Strategy   string `toml:"strategy"`
	Period     string `toml:"period"`
	RebootTime string `toml:"reboot_time"`
	Stream     string `toml:"stream"`
}

type BootcConfig struct {
	UpdateTime      string `toml:"update_time"`
	HealthCheckWait int    `toml:"health_check_wait"`
}

type ContainerRegistryConfig struct {
	URL string `toml:"url"`
}
