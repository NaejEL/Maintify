package pluginmgr

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// ComposeConfig represents a docker-compose.yml file structure
type ComposeConfig struct {
	Version  string             `yaml:"version"`
	Services map[string]Service `yaml:"services"`
	Networks map[string]Network `yaml:"networks,omitempty"`
}

// Service represents a service in docker-compose
type Service struct {
	Image         string            `yaml:"image"`
	ContainerName string            `yaml:"container_name,omitempty"`
	Environment   map[string]string `yaml:"environment,omitempty"`
	Ports         []string          `yaml:"ports,omitempty"`
	Networks      []string          `yaml:"networks,omitempty"`
	Deploy        *Deploy           `yaml:"deploy,omitempty"`
	Restart       string            `yaml:"restart,omitempty"`
}

// Deploy represents deployment configuration (resources)
type Deploy struct {
	Resources Resources `yaml:"resources,omitempty"`
}

// Resources represents resource limits
type Resources struct {
	Limits Limits `yaml:"limits,omitempty"`
}

// Limits represents specific resource limits
type Limits struct {
	Cpus   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// Network represents a network configuration
type Network struct {
	External bool `yaml:"external,omitempty"`
	Name     string `yaml:"name,omitempty"`
}

// GenerateComposeFile generates the docker-compose YAML content for a plugin
func GenerateComposeFile(plugin PluginMeta, imageName string, networkName string) ([]byte, error) {
	serviceName := "plugin-" + plugin.Name
	containerName := "maintify-plugin-" + plugin.Name

	// Convert resources to string format expected by compose
	cpus := ""
	if plugin.Resources.CPUMilliCores > 0 {
		cpus = fmt.Sprintf("%.1f", float64(plugin.Resources.CPUMilliCores)/1000.0)
	}

	memory := ""
	if plugin.Resources.MemoryMB > 0 {
		memory = fmt.Sprintf("%dM", plugin.Resources.MemoryMB)
	}

	service := Service{
		Image:         imageName,
		ContainerName: containerName,
		Restart:       "unless-stopped",
		Networks:      []string{networkName},
		Environment: map[string]string{
			"PLUGIN_NAME": plugin.Name,
		},
	}

	if cpus != "" || memory != "" {
		service.Deploy = &Deploy{
			Resources: Resources{
				Limits: Limits{
					Cpus:   cpus,
					Memory: memory,
				},
			},
		}
	}

	config := ComposeConfig{
		Version: "3.8",
		Services: map[string]Service{
			serviceName: service,
		},
		Networks: map[string]Network{
			networkName: {
				External: true,
				Name:     networkName,
			},
		},
	}

	return yaml.Marshal(&config)
}
