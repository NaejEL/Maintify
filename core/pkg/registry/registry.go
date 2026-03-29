// Package registry is a Phase 2 placeholder for the plugin registry.
// It currently holds a simple in-memory set of registered plugin names.
// Full plugin discovery and registration logic will be implemented in Phase 2.
package registry

var RegisteredPlugins = map[string]bool{}

func Init() {
	RegisteredPlugins = make(map[string]bool)
}

func Register(name string) {
	RegisteredPlugins[name] = true
}

func Unregister(name string) {
	delete(RegisteredPlugins, name)
}
