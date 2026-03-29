package pluginmgr

import (
	"os"
	"os/exec"
	"path/filepath"
)

var (
	// PluginDir is the directory where plugins are stored
	PluginDir = "/plugins"
)

// ClonePlugin clones a plugin from a Git repository into the plugins directory.
// callers must validate repoURL before passing it here.
func ClonePlugin(repoURL string) error {
	cmd := exec.Command("git", "clone", repoURL) // #nosec G204 -- repoURL is admin-sourced, not from untrusted HTTP input
	cmd.Dir = PluginDir
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

// RegisterPlugin registers a plugin at runtime
func RegisterPlugin(name string) error {
	metaPath := filepath.Join(PluginDir, name, "plugin.yaml")
	if _, err := os.Stat(metaPath); err != nil {
		return err
	}
	// Optionally, validate plugin.yaml here
	return nil
}

// UnregisterPlugin removes a plugin from the registry
func UnregisterPlugin(name string) error {
	return os.RemoveAll(filepath.Join(PluginDir, name))
}
