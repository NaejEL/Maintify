package pluginmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"maintify/core/pkg/config"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Package-level wrapper handlers
// ---------------------------------------------------------------------------

func setupGlobalManager(t *testing.T) func() {
	t.Helper()
	origManager := lifecycleManager
	origHandler := lifecycleHandler
	origPlugins := plugins

	lifecycleManager = NewLifecycleManager(&fakeComposeRunner{})
	lifecycleManager.SetPlugins([]PluginMeta{{Name: "asset"}})
	lifecycleHandler = NewLifecycleHandler(lifecycleManager)
	plugins = []PluginMeta{{Name: "asset"}}

	return func() {
		lifecycleManager = origManager
		lifecycleHandler = origHandler
		plugins = origPlugins
	}
}

func TestPluginListHandler(t *testing.T) {
	cleanup := setupGlobalManager(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/plugins", nil)
	w := httptest.NewRecorder()
	PluginListHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var out []PluginMeta
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &out))
	assert.Len(t, out, 1)
}

func TestPluginStatusHandler_Wrapper(t *testing.T) {
	cleanup := setupGlobalManager(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/status", nil)
	w := httptest.NewRecorder()
	PluginStatusHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPluginStartHandler_Wrapper(t *testing.T) {
	cleanup := setupGlobalManager(t)
	defer cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/start", PluginStartHandler).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPluginStopHandler_Wrapper(t *testing.T) {
	cleanup := setupGlobalManager(t)
	defer cleanup()

	_ = lifecycleManager.Start("asset") // put into running first
	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/stop", PluginStopHandler).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPluginRestartHandler_Wrapper(t *testing.T) {
	cleanup := setupGlobalManager(t)
	defer cleanup()

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/restart", PluginRestartHandler).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/restart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPluginDiagnosticsHandler_Wrapper(t *testing.T) {
	cleanup := setupGlobalManager(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/diagnostics", nil)
	w := httptest.NewRecorder()
	PluginDiagnosticsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// ---------------------------------------------------------------------------
// DefaultLaunchOptions
// ---------------------------------------------------------------------------

func TestDefaultLaunchOptions(t *testing.T) {
	opts := DefaultLaunchOptions()
	assert.Equal(t, "/plugins", opts.PluginDir)
	assert.NotEmpty(t, opts.NetworkName)
}

// ---------------------------------------------------------------------------
// HTTPBuilderClient.BuildImage via fake HTTP server
// ---------------------------------------------------------------------------

func TestHTTPBuilderClient_BuildImage_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(BuildResponse{Status: "success", Image: "test-plugin:v1"})
	}))
	defer srv.Close()

	client := &HTTPBuilderClient{URL: srv.URL, APIKey: "key", client: &http.Client{}}
	img, err := client.BuildImage("test-plugin", "/ctx")
	assert.NoError(t, err)
	assert.Equal(t, "test-plugin:v1", img)
}

func TestHTTPBuilderClient_BuildImage_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer srv.Close()

	client := &HTTPBuilderClient{URL: srv.URL, APIKey: "", client: &http.Client{}}
	_, err := client.BuildImage("x", "/ctx")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestHTTPBuilderClient_BuildImage_BuilderFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(BuildResponse{Status: "error", Image: ""})
	}))
	defer srv.Close()

	client := &HTTPBuilderClient{URL: srv.URL, APIKey: "", client: &http.Client{}}
	_, err := client.BuildImage("x", "/ctx")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failure")
}

func TestHTTPBuilderClient_BuildImage_Unreachable(t *testing.T) {
	client := &HTTPBuilderClient{URL: "http://127.0.0.1:1/build", APIKey: "", client: &http.Client{}}
	_, err := client.BuildImage("x", "/ctx")
	assert.Error(t, err)
}

func TestHTTPBuilderClient_BuildImage_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json{{"))
	}))
	defer srv.Close()

	client := &HTTPBuilderClient{URL: srv.URL, APIKey: "", client: &http.Client{}}
	_, err := client.BuildImage("x", "/ctx")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// InitWithOptions — fully testable Init path
// ---------------------------------------------------------------------------

func TestInitWithOptions_NoPlugins_DoesNotPanic(t *testing.T) {
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

	lifecycleManager = NewLifecycleManager(&fakeComposeRunner{})
	lifecycleHandler = NewLifecycleHandler(lifecycleManager)

	assert.NotPanics(t, func() {
		InitWithOptions(
			t.TempDir(),
			&fakeBuildClient{},
			&fakeComposeRunner{},
			LaunchOptions{PluginDir: t.TempDir(), NetworkName: "test"},
		)
	})
}

func TestInitWithOptions_WithPlugin_DiscoveredAndRegistered(t *testing.T) {
	tmpDir := t.TempDir()
	writePluginMeta(t, tmpDir, "asset", "name: asset\nversion: 1.0.0\nroute: /asset\n")
	scaffoldPlugin(t, tmpDir, "asset")

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

	runner := &fakeComposeRunner{}
	lifecycleManager = NewLifecycleManager(runner)
	lifecycleHandler = NewLifecycleHandler(lifecycleManager)

	builder := &fakeBuildClient{imageByPlugin: map[string]string{"asset": "asset:v1"}}
	InitWithOptions(tmpDir, builder, runner, LaunchOptions{PluginDir: tmpDir, NetworkName: "test-net"})

	assert.Len(t, plugins, 1)
	assert.Equal(t, "asset", plugins[0].Name)
}

func TestInitWithOptions_WithDiscoveryIssue_LogsAndContinues(t *testing.T) {
	tmpDir := t.TempDir()
	// Write a plugin dir whose plugin.yaml has a required field missing
	writePluginMeta(t, tmpDir, "incomplete", "name: incomplete\nversion: 1.0.0\n")

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

	lifecycleManager = NewLifecycleManager(&fakeComposeRunner{})
	lifecycleHandler = NewLifecycleHandler(lifecycleManager)

	assert.NotPanics(t, func() {
		InitWithOptions(tmpDir, &fakeBuildClient{}, &fakeComposeRunner{},
			LaunchOptions{PluginDir: tmpDir, NetworkName: "test"})
	})
	assert.NotEmpty(t, pluginDiscoveryIssues)
}

// ---------------------------------------------------------------------------
// Init — delegates to InitWithOptions (production entry point)
// Use config.Current to verify the config branch in NewHTTPBuilderClient
// ---------------------------------------------------------------------------

func TestNewHTTPBuilderClient_WithConfig_AppliesAPIKey(t *testing.T) {
	orig := config.Current
	config.Current = &config.Config{BuilderAPIKey: "from-config"}
	defer func() { config.Current = orig }()

	c := NewHTTPBuilderClient()
	assert.Equal(t, "from-config", c.APIKey)
}

// ---------------------------------------------------------------------------
// Init — production entry point (delegates to InitWithOptions)
// ---------------------------------------------------------------------------

func TestInit_DoesNotPanic_WhenPluginsDirMissing(t *testing.T) {
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

	// Swap in a fake runner so Init doesn't wait for Docker
	lifecycleManager = NewLifecycleManager(&fakeComposeRunner{})
	lifecycleHandler = NewLifecycleHandler(lifecycleManager)

	// /plugins does not exist; Init should log a filesystem issue and return cleanly
	assert.NotPanics(t, Init)
}
