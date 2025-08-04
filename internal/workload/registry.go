package workload

import (
	"fmt"
)

type Registry struct {
	workloads map[string]Workload
}

func NewRegistry() *Registry {
	return &Registry{
		workloads: make(map[string]Workload),
	}
}

func (r *Registry) Register(workload Workload) {
	r.workloads[workload.Name()] = workload
}

func (r *Registry) Get(name string) (Workload, error) {
	workload, exists := r.workloads[name]
	if !exists {
		return nil, fmt.Errorf("workload '%s' not found", name)
	}
	return workload, nil
}

// GetDefault returns a default workload implementation for any machine
func (r *Registry) GetDefault(machineName string) Workload {
	return &DefaultWorkload{
		machineName: machineName,
	}
}

func (r *Registry) List() []string {
	var names []string
	for name := range r.workloads {
		names = append(names, name)
	}
	return names
}

type WorkloadDefinition struct {
	Name           string
	ContainerImage string
	ContainerTag   string
	BootcSource    string
}

func CreateDefaultRegistry(workloadDefs []WorkloadDefinition) *Registry {
	// Since we're not using workload types anymore, just return an empty registry
	return NewRegistry()
}

// DefaultWorkload is a generic workload implementation for all machines
type DefaultWorkload struct {
	machineName string
}

func (d *DefaultWorkload) Name() string {
	return d.machineName
}

func (d *DefaultWorkload) ContainerImage() string {
	return "ghcr.io/" + d.machineName
}

func (d *DefaultWorkload) ContainerTag() string {
	return "latest"
}

func (d *DefaultWorkload) GetButaneOverlay() (string, error) {
	// Return only the sections that should be merged, not a complete butane config
	// This avoids duplicate keys like variant/version that are already in the base template
	return `systemd:
  units:
    - name: bootc-{{ .Machine.Name }}.service
      enabled: true
      contents: |
        [Unit]
        Description=Bootc {{ .Machine.Name }} Container
        After=network-online.target podman.service
        Wants=network-online.target
        Requires=podman.service
        
        [Service]
        Type=notify
        NotifyAccess=all
        Restart=always
        RestartSec=30
        TimeoutStartSec=300
        
        ExecStartPre=/usr/bin/podman pull {{ .ContainerRegistry.URL }}/{{ .Machine.Name }}:latest
        
        ExecStart=/usr/bin/podman run \
          --rm \
          --name bootc-{{ .Machine.Name }} \
          --net host \
          --pid host \
          --privileged \
          --security-opt label=disable \
          --volume /etc:/etc \
          --volume /var:/var \
          --volume /run:/run \
          --env MACHINE_NAME={{ .Machine.Name }} \
          --sdnotify=conmon \
          --health-cmd "/usr/local/bin/health.sh" \
          --health-interval=30s \
          --health-retries=3 \
          --health-start-period=60s \
          {{ .ContainerRegistry.URL }}/{{ .Machine.Name }}:latest
        
        ExecStop=/usr/bin/podman stop -t 30 bootc-{{ .Machine.Name }}
        
        [Install]
        WantedBy=multi-user.target

storage:
  directories:
    - path: /var/lib/{{ .Machine.Name }}
      mode: "0755"
    - path: /var/log/{{ .Machine.Name }}
      mode: "0755"
  files:
    - path: /etc/iago/secrets/{{ .Machine.Name }}-password
      mode: "0600"
      contents:
        inline: {{ .GeneratedSecrets.Password }}`, nil
}

func (d *DefaultWorkload) GetSecrets() []SecretDefinition {
	return []SecretDefinition{
		{
			Name:        "password",
			Path:        "/etc/iago/secrets/" + d.machineName + "-password",
			Description: "Application password for " + d.machineName,
			Generator:   generateRandomPassword,
		},
	}
}

func (d *DefaultWorkload) Validate(config MachineConfig) error {
	// No validation needed for default workload
	return nil
}

func generateRandomPassword() (string, error) {
	// Simple password generation - in real implementation use crypto/rand
	return "auto-generated-password-123", nil
}
