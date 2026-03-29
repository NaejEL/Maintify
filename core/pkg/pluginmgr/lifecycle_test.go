package pluginmgr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

type fakeComposeRunner struct {
	errs  map[string]error
	calls []string
}

func (f *fakeComposeRunner) Run(pluginName string, composePath string, args ...string) error {
	call := pluginName + ":" + joinArgs(args)
	f.calls = append(f.calls, call)
	if f.errs == nil {
		return nil
	}
	if err, ok := f.errs[call]; ok {
		return err
	}
	return nil
}

func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
}

func TestLifecycleManager_StartStopRestart_StateTracking(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})

	err := manager.Start("asset")
	assert.NoError(t, err)
	assert.Equal(t, PluginStateRunning, manager.Status("asset").State)

	err = manager.Stop("asset")
	assert.NoError(t, err)
	assert.Equal(t, PluginStateStopped, manager.Status("asset").State)

	err = manager.Restart("asset")
	assert.NoError(t, err)
	assert.Equal(t, PluginStateRunning, manager.Status("asset").State)
}

func TestLifecycleManager_StartFailure_SetsFailedState(t *testing.T) {
	runner := &fakeComposeRunner{
		errs: map[string]error{
			"asset:up -d": errors.New("compose failed"),
		},
	}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})

	err := manager.Start("asset")
	assert.Error(t, err)
	status := manager.Status("asset")
	assert.Equal(t, PluginStateFailed, status.State)
	assert.Contains(t, status.LastError, "compose failed")
}

func TestLifecycleManager_PluginStatusHandler_ReturnsAllStatuses(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}, {Name: "work"}})
	_ = manager.Start("asset")

	handler := NewLifecycleHandler(manager)
	req := httptest.NewRequest(http.MethodGet, "/api/plugins/status", nil)
	resp := httptest.NewRecorder()

	handler.PluginStatusHandler(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	var statuses []PluginRuntimeStatus
	err := json.Unmarshal(resp.Body.Bytes(), &statuses)
	assert.NoError(t, err)
	assert.Len(t, statuses, 2)
}

func TestLifecycleManager_PluginActionHandler_Start(t *testing.T) {
	runner := &fakeComposeRunner{}
	manager := NewLifecycleManager(runner)
	manager.SetPlugins([]PluginMeta{{Name: "asset"}})
	handler := NewLifecycleHandler(manager)

	router := mux.NewRouter()
	router.HandleFunc("/api/plugins/{name}/start", handler.PluginActionHandler("start")).Methods(http.MethodPost)

	req := httptest.NewRequest(http.MethodPost, "/api/plugins/asset/start", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, PluginStateRunning, manager.Status("asset").State)
}

func TestLifecycleManager_Stop_UnknownPlugin_ReturnsError(t *testing.T) {
	manager := NewLifecycleManager(&fakeComposeRunner{})
	err := manager.Stop("nonexistent")
	assert.EqualError(t, err, "plugin not found")
}

func TestLifecycleManager_Start_UnknownPlugin_ReturnsError(t *testing.T) {
	manager := NewLifecycleManager(&fakeComposeRunner{})
	err := manager.Start("nonexistent")
	assert.EqualError(t, err, "plugin not found")
}
