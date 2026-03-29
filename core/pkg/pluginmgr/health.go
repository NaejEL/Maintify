package pluginmgr

import (
	"fmt"
	"net/http"
	"time"
)

// HealthCheckerOptions controls probe behaviour.
type HealthCheckerOptions struct {
	TimeoutMs int // HTTP request timeout in milliseconds; 0 = use default (2000ms)
}

// PluginHealthResult is the outcome of a single health probe.
type PluginHealthResult struct {
	Name       string    `json:"name"`
	Healthy    bool      `json:"healthy"`
	StatusCode int       `json:"status_code,omitempty"`
	LatencyMs  int64     `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
	CheckedAt  time.Time `json:"checked_at"`
}

// DiagnosticsReport aggregates all plugin health results.
type DiagnosticsReport struct {
	CheckedAt    time.Time           `json:"checked_at"`
	TotalCount   int                 `json:"total_count"`
	HealthyCount int                 `json:"healthy_count"`
	Plugins      []PluginDiagnostics `json:"plugins"`
}

// PluginDiagnostics combines the runtime status and latest health result.
type PluginDiagnostics struct {
	Status PluginRuntimeStatus `json:"status"`
	Health PluginHealthResult  `json:"health"`
}

// HealthChecker probes plugin backends over HTTP.
type HealthChecker struct {
	client *http.Client
}

// NewHealthChecker creates a HealthChecker. Pass nil opts for defaults.
func NewHealthChecker(opts *HealthCheckerOptions) *HealthChecker {
	timeoutMs := 2000
	if opts != nil && opts.TimeoutMs > 0 {
		timeoutMs = opts.TimeoutMs
	}
	return &HealthChecker{
		client: &http.Client{
			Timeout: time.Duration(timeoutMs) * time.Millisecond,
		},
	}
}

// CheckPlugin performs a single HTTP GET against plugin.BackendURL.
func (h *HealthChecker) CheckPlugin(plugin PluginMeta) PluginHealthResult {
	result := PluginHealthResult{
		Name:      plugin.Name,
		CheckedAt: time.Now().UTC(),
	}

	if plugin.BackendURL == "" {
		result.Error = "no backend URL configured"
		return result
	}

	start := time.Now()
	resp, err := h.client.Get(plugin.BackendURL)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = fmt.Sprintf("probe failed: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Healthy = true
	} else {
		result.Error = fmt.Sprintf("unhealthy status code: %d", resp.StatusCode)
	}
	return result
}

// CheckAll probes all plugins concurrently and collects results.
func (h *HealthChecker) CheckAll(plugins []PluginMeta) []PluginHealthResult {
	type indexed struct {
		i      int
		result PluginHealthResult
	}

	ch := make(chan indexed, len(plugins))
	for i, p := range plugins {
		go func(idx int, plugin PluginMeta) {
			ch <- indexed{i: idx, result: h.CheckPlugin(plugin)}
		}(i, p)
	}

	results := make([]PluginHealthResult, len(plugins))
	for range plugins {
		item := <-ch
		results[item.i] = item.result
	}
	return results
}
