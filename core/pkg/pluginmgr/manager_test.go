package pluginmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterUnregisterPlugin(t *testing.T) {
	// Use a temporary directory for plugins
	tmpDir := t.TempDir()
	origPluginDir := PluginDir
	PluginDir = tmpDir
	defer func() { PluginDir = origPluginDir }()

	pluginName := "test_plugin"
	pluginPath := filepath.Join(tmpDir, pluginName)
	err := os.MkdirAll(pluginPath, 0755)
	assert.NoError(t, err)

	f, err := os.Create(filepath.Join(pluginPath, "plugin.yaml"))
	assert.NoError(t, err)
	f.WriteString(`{"name":"test_plugin"}`)
	f.Close()

	t.Run("RegisterPlugin success", func(t *testing.T) {
		err := RegisterPlugin(pluginName)
		assert.NoError(t, err)
	})

	t.Run("RegisterPlugin failure - no plugin.yaml", func(t *testing.T) {
		err := RegisterPlugin("non_existent")
		assert.Error(t, err)
	})

	t.Run("UnregisterPlugin success", func(t *testing.T) {
		err := UnregisterPlugin(pluginName)
		assert.NoError(t, err)
		_, err = os.Stat(pluginPath)
		assert.True(t, os.IsNotExist(err))
	})
}
