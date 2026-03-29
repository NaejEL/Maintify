package pluginmgr

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

// --- helpers ----------------------------------------------------------------

// --- HealthChecker unit tests -----------------------------------------------

func TestHealthChecker_CheckPlugin_Healthy(t *testing.T) {
	// Start a fake backend that returns 200
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	checker := NewHealthChecker(nil)
	result := checker.CheckPlugin(PluginMeta{Name: "asset", BackendURL: backend.URL + "/health"})

	assert.True(t, result.Healthy)
	assert.Equal(t, "asset", result.Name)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Empty(t, result.Error)
	assert.GreaterOrEqual(t, result.LatencyMs, int64(0)) // sub-ms on loopback is valid
}

func TestHealthChecker_CheckPlugin_Unhealthy_ServerError(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer backend.Close()

	checker := NewHealthChecker(nil)
	result := checker.CheckPlugin(PluginMeta{Name: "work", BackendURL: backend.URL + "/health"})

	assert.False(t, result.Healthy)
	assert.Equal(t, http.StatusServiceUnavailable, result.StatusCode)
	assert.NotEmpty(t, result.Error)
}

func TestHealthChecker_CheckPlugin_Unhealthy_Unreachable(t *testing.T) {
	checker := NewHealthChecker(nil)
	result := checker.CheckPlugin(PluginMeta{Name: "ghost", BackendURL: "http://127.0.0.1:1/health"})

	assert.False(t, result.Healthy)
	assert.NotEmpty(t, result.Error)
}

func TestHealthChecker_CheckPlugin_NoBackendURL(t *testing.T) {
	checker := NewHealthChecker(nil)
	// Plugin with no backend URL should not be probed — returns unknown state
	result := checker.CheckPlugin(PluginMeta{Name: "ui-only", BackendURL: ""})

	assert.False(t, result.Healthy)
	assert.Equal(t, "no backend URL configured", result.Error)
}

func TestHealthChecker_CheckAll_ReturnsResultPerPlugin(t *testing.T) {
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer healthy.Close()

	plugins := []PluginMeta{
		{Name: "a", BackendURL: healthy.URL + "/health"},
		{Name: "b", BackendURL: "http://127.0.0.1:1/health"},
	}

	checker := NewHealthChecker(nil)
	results := checker.CheckAll(plugins)

	assert.Len(t, results, 2)
	byName := map[string]PluginHealthResult{}
	for _, r := range results {
		byName[r.Name] = r
	}
	assert.True(t, byName["a"].Healthy)
	assert.False(t, byName["b"].Healthy)
}

// --- HealthChecker with custom timeout --------------------------------------

func TestHealthChecker_RespectsTimeout(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer slow.Close()

	checker := NewHealthChecker(&HealthCheckerOptions{TimeoutMs: 50})
	result := checker.CheckPlugin(PluginMeta{Name: "slow", BackendURL: slow.URL + "/health"})

	assert.False(t, result.Healthy)
	assert.NotEmpty(t, result.Error)
}

// --- Diagnostics HTTP handler -----------------------------------------------

func TestDiagnosticsHandler_ReturnsAggregatedReport(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{
		{Name: "asset", BackendURL: "http://localhost:9001/health"},
		{Name: "work", BackendURL: "http://localhost:9002/health"},
	})
	_ = manager.Start("asset")

	// Use a checker whose HTTP client will fail (no real backends) — that's fine,
	// we only check the response structure, not individual health values.
	handler := NewLifecycleHandler(manager)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/diagnostics", nil)
	resp := httptest.NewRecorder()
	handler.DiagnosticsHandler(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var report DiagnosticsReport
	err := json.Unmarshal(resp.Body.Bytes(), &report)
	assert.NoError(t, err)
	assert.Len(t, report.Plugins, 2)
	assert.NotZero(t, report.CheckedAt)
	assert.GreaterOrEqual(t, report.TotalCount, 0)
}

// --- Diagnostics handler via router -----------------------------------------

func TestDiagnosticsHandler_RouteWired(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})
	handler := NewLifecycleHandler(manager)

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/diagnostics", handler.DiagnosticsHandler).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/api/plugins/diagnostics", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
}

// --- LifecycleManager unknown plugin action ---------------------------------

func TestLifecycleHandler_ActionHandler_UnsupportedAction(t *testing.T) {
	manager := NewLifecycleManager(&fakeComposeRunner{})
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})
	handler := NewLifecycleHandler(manager)

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/nuke", handler.PluginActionHandler("nuke")).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/nuke", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// --- LifecycleManager unknown plugin stop -----------------------------------

func TestLifecycleManager_StopFailure_SetsFailedState(t *testing.T) {
	runner := &fakeComposeRunner{
		errs: map[string]error{
			"asset:down": assert.AnError,
		},
	}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})

	err := manager.Stop("asset")
	assert.Error(t, err)
	assert.Equal(t, PluginStateFailed, manager.Status("asset").State)
}

func TestLifecycleManager_Restart_FailsIfStopFails(t *testing.T) {
	runner := &fakeComposeRunner{
		errs: map[string]error{
			"asset:down": assert.AnError,
		},
	}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})

	err := manager.Restart("asset")
	assert.Error(t, err)
}

func TestLifecycleManager_Status_UnknownPlugin(t *testing.T) {
	manager := NewLifecycleManager(&fakeComposeRunner{})
	status := manager.Status("nonexistent")
	assert.Equal(t, PluginStateUnknown, status.State)
}

func TestLifecycleManager_ActionHandlerStop_UpdatesState(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})
	_ = manager.Start("asset")
	handler := NewLifecycleHandler(manager)

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/stop", handler.PluginActionHandler("stop")).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/stop", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, PluginStateStopped, manager.Status("asset").State)
}

func TestLifecycleManager_ActionHandlerRestart_UpdatesState(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})
	handler := NewLifecycleHandler(manager)

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/restart", handler.PluginActionHandler("restart")).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/restart", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, PluginStateRunning, manager.Status("asset").State)
}

func TestLifecycleManager_ActionHandler_PluginNotFound(t *testing.T) {
	manager := NewLifecycleManager(&fakeComposeRunner{})
	handler := NewLifecycleHandler(manager)

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/start", handler.PluginActionHandler("start")).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/ghost/start", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}
