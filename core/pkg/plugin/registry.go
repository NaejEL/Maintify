package plugin

import (
	"fmt"
	"sync"
)

// SimpleRegistry provides a basic in-memory plugin registry
type SimpleRegistry struct {
	plugins map[string]Plugin
	mutex   sync.RWMutex
}

// NewSimpleRegistry creates a new SimpleRegistry
func NewSimpleRegistry() *SimpleRegistry {
	return &SimpleRegistry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry
func (r *SimpleRegistry) Register(plugin Plugin) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	metadata := plugin.GetMetadata()
	if metadata.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	
	if _, exists := r.plugins[metadata.Name]; exists {
		return fmt.Errorf("plugin '%s' is already registered", metadata.Name)
	}
	
	r.plugins[metadata.Name] = plugin
	return nil
}

// Unregister removes a plugin from the registry
func (r *SimpleRegistry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.plugins[name]; !exists {
		return fmt.Errorf("plugin '%s' is not registered", name)
	}
	
	delete(r.plugins, name)
	return nil
}

// Get retrieves a plugin by name
func (r *SimpleRegistry) Get(name string) (Plugin, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	plugin, exists := r.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' not found", name)
	}
	
	return plugin, nil
}

// List returns all registered plugins
func (r *SimpleRegistry) List() []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	plugins := make([]Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}
	
	return plugins
}

// GetByCapability returns plugins that have a specific capability
func (r *SimpleRegistry) GetByCapability(capability string) []Plugin {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var matchingPlugins []Plugin
	for _, plugin := range r.plugins {
		metadata := plugin.GetMetadata()
		for _, cap := range metadata.Capabilities {
			if cap == capability {
				matchingPlugins = append(matchingPlugins, plugin)
				break
			}
		}
	}
	
	return matchingPlugins
}

// GetNames returns all registered plugin names
func (r *SimpleRegistry) GetNames() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	
	return names
}

// Size returns the number of registered plugins
func (r *SimpleRegistry) Size() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	return len(r.plugins)
}