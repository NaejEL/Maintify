package pluginmgr

import (
	"errors"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	PluginStateUnknown = "unknown"
	PluginStateRunning = "running"
	PluginStateStopped = "stopped"
	PluginStateFailed  = "failed"
)

type PluginRuntimeStatus struct {
	Name        string    `json:"name"`
	State       string    `json:"state"`
	LastError   string    `json:"last_error,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
	ComposePath string    `json:"compose_path"`
}

type ComposeRunner interface {
	Run(pluginName string, composePath string, args ...string) error
}

type ShellComposeRunner struct{}

// Run is defined in shell_runner.go

type LifecycleManager struct {
	mu         sync.RWMutex
	plugins    map[string]PluginMeta
	statuses   map[string]PluginRuntimeStatus
	runner     ComposeRunner
	pluginsDir string
}

func NewLifecycleManager(runner ComposeRunner) *LifecycleManager {
	if runner == nil {
		runner = &ShellComposeRunner{}
	}
	return &LifecycleManager{
		plugins:    make(map[string]PluginMeta),
		statuses:   make(map[string]PluginRuntimeStatus),
		runner:     runner,
		pluginsDir: "/plugins",
	}
}

func (m *LifecycleManager) SetPlugins(pluginList []PluginMeta) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, plugin := range pluginList {
		m.plugins[plugin.Name] = plugin
		if _, exists := m.statuses[plugin.Name]; !exists {
			m.statuses[plugin.Name] = PluginRuntimeStatus{
				Name:        plugin.Name,
				State:       PluginStateStopped,
				UpdatedAt:   time.Now().UTC(),
				ComposePath: m.composePath(plugin.Name),
			}
		}
	}
}

func (m *LifecycleManager) Start(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[pluginName]; !exists {
		return errors.New("plugin not found")
	}

	composePath := m.composePath(pluginName)
	if err := m.runner.Run(pluginName, composePath, "up", "-d"); err != nil {
		m.statuses[pluginName] = PluginRuntimeStatus{
			Name:        pluginName,
			State:       PluginStateFailed,
			LastError:   err.Error(),
			UpdatedAt:   time.Now().UTC(),
			ComposePath: composePath,
		}
		return err
	}

	m.statuses[pluginName] = PluginRuntimeStatus{
		Name:        pluginName,
		State:       PluginStateRunning,
		UpdatedAt:   time.Now().UTC(),
		ComposePath: composePath,
	}
	return nil
}

func (m *LifecycleManager) Stop(pluginName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.plugins[pluginName]; !exists {
		return errors.New("plugin not found")
	}

	composePath := m.composePath(pluginName)
	if err := m.runner.Run(pluginName, composePath, "down"); err != nil {
		m.statuses[pluginName] = PluginRuntimeStatus{
			Name:        pluginName,
			State:       PluginStateFailed,
			LastError:   err.Error(),
			UpdatedAt:   time.Now().UTC(),
			ComposePath: composePath,
		}
		return err
	}

	m.statuses[pluginName] = PluginRuntimeStatus{
		Name:        pluginName,
		State:       PluginStateStopped,
		UpdatedAt:   time.Now().UTC(),
		ComposePath: composePath,
	}
	return nil
}

func (m *LifecycleManager) Restart(pluginName string) error {
	if err := m.Stop(pluginName); err != nil {
		return err
	}
	return m.Start(pluginName)
}

func (m *LifecycleManager) Status(pluginName string) PluginRuntimeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, exists := m.statuses[pluginName]
	if !exists {
		return PluginRuntimeStatus{Name: pluginName, State: PluginStateUnknown}
	}
	return status
}

func (m *LifecycleManager) ListStatus() []PluginRuntimeStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]PluginRuntimeStatus, 0, len(m.statuses))
	for _, status := range m.statuses {
		statuses = append(statuses, status)
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].Name < statuses[j].Name
	})
	return statuses
}

func (m *LifecycleManager) composePath(pluginName string) string {
	return filepath.Join(m.pluginsDir, pluginName, "docker-compose.yml")
}
