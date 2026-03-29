package pluginmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func writePluginMeta(t *testing.T, pluginDir, pluginFolder, yaml string) {
	t.Helper()
	err := os.MkdirAll(filepath.Join(pluginDir, pluginFolder), 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(pluginDir, pluginFolder, "plugin.yaml"), []byte(yaml), 0o644)
	assert.NoError(t, err)
}

func TestDiscoverPluginsInDir_SkipsInvalidMetadataAndReportsIssues(t *testing.T) {
	tmpDir := t.TempDir()

	writePluginMeta(t, tmpDir, "asset", `
name: asset
version: 1.0.0
route: /asset
`)
	writePluginMeta(t, tmpDir, "broken-yaml", `
name: broken
version: "1.0.0
route: /broken
`)
	writePluginMeta(t, tmpDir, "missing-name", `
version: 1.0.0
route: /missing
`)

	plugins, issues := discoverPluginsInDir(tmpDir)

	assert.Len(t, plugins, 1)
	assert.Equal(t, "asset", plugins[0].Name)
	assert.Len(t, issues, 2)
	assert.Contains(t, issues[0].Message+issues[1].Message, "invalid metadata")
}

func TestDiscoverPluginsInDir_DetectsNameAndRouteConflicts(t *testing.T) {
	tmpDir := t.TempDir()

	writePluginMeta(t, tmpDir, "asset-v1", `
name: asset
version: 1.0.0
route: /asset
`)
	writePluginMeta(t, tmpDir, "asset-v2", `
name: asset
version: 2.0.0
route: /asset-v2
`)
	writePluginMeta(t, tmpDir, "work-orders", `
name: work-orders
version: 1.0.0
route: /asset
`)

	plugins, issues := discoverPluginsInDir(tmpDir)

	assert.Len(t, plugins, 1)
	assert.Equal(t, "asset", plugins[0].Name)
	assert.Len(t, issues, 2)
	assert.Equal(t, IssueTypeConflict, issues[0].Type)
	assert.Equal(t, IssueTypeConflict, issues[1].Type)
}

func TestDiscoverPluginsInDir_ReturnsIssueWhenDirectoryMissing(t *testing.T) {
	plugins, issues := discoverPluginsInDir(filepath.Join(t.TempDir(), "missing"))

	assert.Empty(t, plugins)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueTypeFilesystem, issues[0].Type)
}

func TestDiscoverPluginsInDir_ReportsIssueForUnreadablePluginYaml(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root, file permission restrictions are not enforced")
	}

	tmpDir := t.TempDir()

	// Create the plugin sub-directory but write plugin.yaml with no read permission
	pluginSubdir := filepath.Join(tmpDir, "locked-plugin")
	assert.NoError(t, os.MkdirAll(pluginSubdir, 0o755))
	yamlPath := filepath.Join(pluginSubdir, "plugin.yaml")
	assert.NoError(t, os.WriteFile(yamlPath, []byte("name: locked\nversion: 1.0.0\nroute: /locked\n"), 0o000))
	t.Cleanup(func() { os.Chmod(yamlPath, 0o644) }) //nolint:errcheck

	plugins, issues := discoverPluginsInDir(tmpDir)

	assert.Empty(t, plugins)
	assert.Len(t, issues, 1)
	assert.Equal(t, IssueTypeMetadata, issues[0].Type)
	assert.Equal(t, "locked-plugin", issues[0].Plugin)
}
