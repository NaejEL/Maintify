package pluginmgr

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type LifecycleHandler struct {
	manager *LifecycleManager
	checker *HealthChecker
}

func NewLifecycleHandler(manager *LifecycleManager) *LifecycleHandler {
	return &LifecycleHandler{manager: manager, checker: NewHealthChecker(nil)}
}

func (h *LifecycleHandler) PluginStatusHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(h.manager.ListStatus())
}

func (h *LifecycleHandler) DiagnosticsHandler(w http.ResponseWriter, _ *http.Request) {
	statuses := h.manager.ListStatus()

	h.manager.mu.RLock()
	pluginList := make([]PluginMeta, 0, len(h.manager.plugins))
	for _, p := range h.manager.plugins {
		pluginList = append(pluginList, p)
	}
	h.manager.mu.RUnlock()

	healthResults := h.checker.CheckAll(pluginList)
	healthByName := make(map[string]PluginHealthResult, len(healthResults))
	for _, hr := range healthResults {
		healthByName[hr.Name] = hr
	}

	diagnostics := make([]PluginDiagnostics, 0, len(statuses))
	healthyCount := 0
	for _, s := range statuses {
		hr := healthByName[s.Name]
		if hr.Healthy {
			healthyCount++
		}
		diagnostics = append(diagnostics, PluginDiagnostics{
			Status: s,
			Health: hr,
		})
	}

	report := DiagnosticsReport{
		CheckedAt:    time.Now().UTC(),
		TotalCount:   len(statuses),
		HealthyCount: healthyCount,
		Plugins:      diagnostics,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(report)
}

func (h *LifecycleHandler) PluginActionHandler(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := mux.Vars(r)["name"]
		var err error

		switch action {
		case "start":
			err = h.manager.Start(name)
		case "stop":
			err = h.manager.Stop(name)
		case "restart":
			err = h.manager.Restart(name)
		default:
			http.Error(w, "unsupported action", http.StatusBadRequest)
			return
		}

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(h.manager.Status(name))
	}
}
