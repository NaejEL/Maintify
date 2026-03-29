package pluginmgr

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"maintify/core/pkg/config"

	"github.com/stretchr/testify/assert"
)

// fakeBuildClient is a test double for BuilderClient.
type fakeBuildClient struct {
	imageByPlugin map[string]string
	errByPlugin   map[string]error
}

func (f *fakeBuildClient) BuildImage(pluginName, _ string) (string, error) {
	if f.errByPlugin != nil {
		if err, ok := f.errByPlugin[pluginName]; ok {
			return "", err
		}
	}
	if f.imageByPlugin != nil {
		if img, ok := f.imageByPlugin[pluginName]; ok {
			return img, nil
		}
	}
	return pluginName + ":latest", nil
}

func scaffoldPlugin(t *testing.T, dir, name string) {
	t.Helper()
	backendDir := filepath.Join(dir, name, "backend")
	assert.NoError(t, os.MkdirAll(backendDir, 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(backendDir, "Dockerfile"), []byte("FROM scratch"), 0o644))
}

func TestLaunchPluginContainers_StartsPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	scaffoldPlugin(t, tmpDir, "asset")

	runner := &fakeComposeRunner{}
	builder := &fakeBuildClient{imageByPlugin: map[string]string{"asset": "asset:v1"}}
	opts := LaunchOptions{PluginDir: tmpDir, NetworkName: "test-net"}

	LaunchPluginContainers(
		[]PluginMeta{{Name: "asset"}},
		builder, runner, opts,
	)

	assert.Len(t, runner.calls, 1)
	assert.Contains(t, runner.calls[0], "asset:up -d")

	// compose file must be written
	composePath := filepath.Join(tmpDir, "asset", "docker-compose.yml")
	_, err := os.Stat(composePath)
	assert.NoError(t, err)
}

func TestLaunchPluginContainers_SkipsIfNoDockerfile(t *testing.T) {
	runner := &fakeComposeRunner{}
	builder := &fakeBuildClient{}
	opts := LaunchOptions{PluginDir: t.TempDir(), NetworkName: "test-net"}

	// No backend/Dockerfile created
	LaunchPluginContainers([]PluginMeta{{Name: "ghost"}}, builder, runner, opts)

	assert.Empty(t, runner.calls)
}

func TestLaunchPluginContainers_SkipsIfBuildFails(t *testing.T) {
	tmpDir := t.TempDir()
	scaffoldPlugin(t, tmpDir, "broken")

	runner := &fakeComposeRunner{}
	builder := &fakeBuildClient{errByPlugin: map[string]error{"broken": errors.New("builder down")}}
	opts := LaunchOptions{PluginDir: tmpDir, NetworkName: "test-net"}

	LaunchPluginContainers([]PluginMeta{{Name: "broken"}}, builder, runner, opts)

	assert.Empty(t, runner.calls)
}

func TestLaunchPluginContainers_SkipsIfComposeRunFails(t *testing.T) {
	tmpDir := t.TempDir()
	scaffoldPlugin(t, tmpDir, "flaky")

	runner := &fakeComposeRunner{errs: map[string]error{"flaky:up -d": errors.New("compose failed")}}
	builder := &fakeBuildClient{}
	opts := LaunchOptions{PluginDir: tmpDir, NetworkName: "test-net"}

	// Should not panic; compose failure is logged and skipped
	LaunchPluginContainers([]PluginMeta{{Name: "flaky"}}, builder, runner, opts)
}

func TestLaunchPluginContainers_SkipsIfComposeWriteFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping: running as root, file permission restrictions are not enforced")
	}

	tmpDir := t.TempDir()
	pluginName := "ro-plugin"
	scaffoldPlugin(t, tmpDir, pluginName)

	// Make the plugin dir read-only so WriteFile fails
	pluginDir := filepath.Join(tmpDir, pluginName)
	assert.NoError(t, os.Chmod(pluginDir, 0o555))
	t.Cleanup(func() { os.Chmod(pluginDir, 0o755) })

	runner := &fakeComposeRunner{}
	builder := &fakeBuildClient{}
	opts := LaunchOptions{PluginDir: tmpDir, NetworkName: "test-net"}

	LaunchPluginContainers([]PluginMeta{{Name: pluginName}}, builder, runner, opts)

	assert.Empty(t, runner.calls)
}

func TestHTTPBuilderClient_Construction(t *testing.T) {
	c := NewHTTPBuilderClient()
	assert.NotNil(t, c)
	assert.Equal(t, "http://builder:8081/build", c.URL)
}

func TestHTTPBuilderClient_Construction_WithConfig(t *testing.T) {
	orig := config.Current
	config.Current = &config.Config{BuilderAPIKey: "secret-key"}
	defer func() { config.Current = orig }()

	c := NewHTTPBuilderClient()
	assert.Equal(t, "secret-key", c.APIKey)
}

func TestNewHTTPBuilderClientWithKey(t *testing.T) {
	c := NewHTTPBuilderClientWithKey("explicit-key")
	assert.Equal(t, "explicit-key", c.APIKey)
	assert.Equal(t, "http://builder:8081/build", c.URL)
}
