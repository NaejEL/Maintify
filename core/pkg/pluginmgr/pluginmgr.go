package pluginmgr

import (
	"encoding/json"
	"log"
	"net/http"
)

// PluginMeta is the metadata loaded from a plugin's plugin.yaml file.
type PluginMeta struct {
	Name        string   `yaml:"name" json:"name"`
	Description string   `yaml:"description" json:"description"`
	Version     string   `yaml:"version" json:"version"`
	BackendURL  string   `yaml:"backend_url" json:"backend_url"`
	FrontendURL string   `yaml:"frontend_url" json:"frontend_url"`
	Schema      string   `yaml:"schema" json:"schema"`
	Route       string   `yaml:"route" json:"route"`
	DependsOn   []string `yaml:"depends_on" json:"depends_on"`
	Resources   struct {
		MemoryMB      int64 `yaml:"memory_mb" json:"memory_mb"`
		CPUMilliCores int64 `yaml:"cpu_milli_cores" json:"cpu_milli_cores"`
	} `yaml:"resources" json:"resources"`
}

// BuildRequest and BuildResponse are the wire types for the builder service API.
type BuildRequest struct {
	PluginName string `json:"plugin_name"`
	ContextDir string `json:"context_dir"`
}

type BuildResponse struct {
	Status string `json:"status"`
	Image  string `json:"image"`
}

var plugins []PluginMeta
var pluginDiscoveryIssues []DiscoveryIssue
var lifecycleManager = NewLifecycleManager(nil)
var lifecycleHandler = NewLifecycleHandler(lifecycleManager)

// InitWithOptions is the fully-testable entry point for the plugin manager.
// Tests swap the pluginDir and inject fake builders/runners.
func InitWithOptions(
	pluginDir string,
	builderClient BuilderClient,
	composeRunner ComposeRunner,
	launchOpts LaunchOptions,
) {
	log.Printf("[pluginmgr] InitWithOptions() called with pluginDir=%s", pluginDir)
	var discovered []PluginMeta
	discovered, pluginDiscoveryIssues = discoverPluginsInDir(pluginDir)
	plugins = discovered
	lifecycleManager.SetPlugins(plugins)
	for _, issue := range pluginDiscoveryIssues {
		log.Printf("[pluginmgr] discovery issue (%s): %s", issue.Type, issue.Message)
	}
	LaunchPluginContainers(plugins, builderClient, composeRunner, launchOpts)
}

// Init is the production entry point; delegates to InitWithOptions with defaults.
func Init() {
	InitWithOptions(
		"/plugins",
		NewHTTPBuilderClient(),
		lifecycleManager.runner,
		DefaultLaunchOptions(),
	)
} // --- HTTP handler wrappers ---

func PluginListHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(plugins)
}

func PluginStatusHandler(w http.ResponseWriter, r *http.Request) {
	lifecycleHandler.PluginStatusHandler(w, r)
}

func PluginStartHandler(w http.ResponseWriter, r *http.Request) {
	lifecycleHandler.PluginActionHandler("start")(w, r)
}

func PluginStopHandler(w http.ResponseWriter, r *http.Request) {
	lifecycleHandler.PluginActionHandler("stop")(w, r)
}

func PluginRestartHandler(w http.ResponseWriter, r *http.Request) {
	lifecycleHandler.PluginActionHandler("restart")(w, r)
}

func PluginDiagnosticsHandler(w http.ResponseWriter, r *http.Request) {
	lifecycleHandler.DiagnosticsHandler(w, r)
}
