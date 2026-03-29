package pluginmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"maintify/core/pkg/config"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// discoverPluginsInDir — non-directory entry branch
// ---------------------------------------------------------------------------

func TestDiscoverPluginsInDir_SkipsNonDirectoryEntries(t *testing.T) {
	tmpDir := t.TempDir()
	assert.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("hi"), 0o644))
	writePluginMeta(t, tmpDir, "asset", "name: asset\nversion: 1.0.0\nroute: /asset\n")

	plugins, issues := discoverPluginsInDir(tmpDir)
	assert.Len(t, plugins, 1)
	assert.Empty(t, issues)
}

// ---------------------------------------------------------------------------
// LaunchPluginContainers — EnableContainerMode=false branch
// ---------------------------------------------------------------------------

func TestLaunchPluginContainers_SkipsWhenContainerModeDisabled(t *testing.T) {
	orig := config.Current
	config.Current = &config.Config{EnableContainerMode: false}
	defer func() { config.Current = orig }()

	tmpDir := t.TempDir()
	scaffoldPlugin(t, tmpDir, "asset")

	runner := &fakeComposeRunner{}
	builder := &fakeBuildClient{imageByPlugin: map[string]string{"asset": "asset:v1"}}
	opts := LaunchOptions{PluginDir: tmpDir, NetworkName: "test-net"}

	LaunchPluginContainers([]PluginMeta{{Name: "asset"}}, builder, runner, opts)

	assert.Empty(t, runner.calls)
}

// ---------------------------------------------------------------------------
// DiagnosticsHandler — healthy plugin increments healthyCount
// ---------------------------------------------------------------------------

func TestDiagnosticsHandler_HealthyCount_WhenPluginIsHealthy(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "healthy", BackendURL: backend.URL + "/health"}})
	_ = manager.Start("healthy")

	handler := NewLifecycleHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/diagnostics", nil)
	resp := httptest.NewRecorder()
	handler.DiagnosticsHandler(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var report DiagnosticsReport
	assert.NoError(t, json.Unmarshal(resp.Body.Bytes(), &report))
	assert.Equal(t, 1, report.HealthyCount)
}

// ---------------------------------------------------------------------------
// ClonePlugin — error path via obviously invalid URL
// ---------------------------------------------------------------------------

func TestClonePlugin_Error_WhenGitFails(t *testing.T) {
	origDir := PluginDir
	PluginDir = t.TempDir()
	defer func() { PluginDir = origDir }()

	err := ClonePlugin("not-a-real-repo-url-that-will-fail")
	assert.Error(t, err)
}

func TestClonePlugin_Success_WithLocalBareRepo(t *testing.T) {
	// Create a local bare git repository so git clone succeeds without network
	bareRepo := t.TempDir()
	initOut, err := runGitCmd(t, bareRepo, "git", "init", "--bare")
	if err != nil {
		t.Skipf("git not available: %v — %s", err, initOut)
	}

	destDir := t.TempDir()
	origDir := PluginDir
	PluginDir = destDir
	defer func() { PluginDir = origDir }()

	cloneErr := ClonePlugin("file://" + bareRepo)
	assert.NoError(t, cloneErr)
}

func runGitCmd(t *testing.T, dir string, name string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ---------------------------------------------------------------------------
// Init — observable side-effects without panicking (no real /plugins dir)
// ---------------------------------------------------------------------------

func TestInit_SideEffects_NoPanic(t *testing.T) {
	origPlugins := plugins
	origIssues := pluginDiscoveryIssues
	origManager := lifecycleManager
	origHandler := lifecycleHandler
	defer func() {
		plugins = origPlugins
		pluginDiscoveryIssues = origIssues
		lifecycleManager = origManager
		lifecycleHandler = origHandler
	}()

	assert.NotPanics(t, func() {
		plugins, pluginDiscoveryIssues = discoverPluginsInDir(t.TempDir())
		lifecycleManager = NewLifecycleManager(&fakeComposeRunner{})
		lifecycleManager.SetPlugins(plugins)
		lifecycleHandler = NewLifecycleHandler(lifecycleManager)
		LaunchPluginContainers(plugins, &fakeBuildClient{}, &fakeComposeRunner{}, LaunchOptions{
			PluginDir:   t.TempDir(),
			NetworkName: "test",
		})
	})
}
