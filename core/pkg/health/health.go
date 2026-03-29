package health

import (
	"context"
	"encoding/json"
	"fmt"
	"maintify/core/pkg/config"
	"maintify/core/pkg/logger"
	"net/http"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
)

// HealthStatus represents the health state of a component
type HealthStatus struct {
	Status    string                 `json:"status"` // "healthy", "unhealthy", "degraded"
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status     string                  `json:"status"`
	Timestamp  time.Time               `json:"timestamp"`
	Uptime     string                  `json:"uptime"`
	Version    string                  `json:"version"`
	Components map[string]HealthStatus `json:"components"`
	System     map[string]interface{}  `json:"system"`
}

var (
	startTime = time.Now()
	rdb       *redis.Client
	// Dependencies for testing
	redisChecker RedisChecker
	httpClient   HTTPClient = &http.Client{Timeout: 5 * time.Second}
	builderURL              = "http://builder:8081/health"
)

// RedisChecker abstracts the Redis ping operation
type RedisChecker interface {
	Ping(ctx context.Context) error
}

// HTTPClient abstracts the HTTP client
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// defaultRedisChecker implements RedisChecker using go-redis
type defaultRedisChecker struct {
	client *redis.Client
}

func (c *defaultRedisChecker) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Initialize sets up health monitoring with Redis client
func Initialize() {
	// Initialize Redis client for health checks
	if config.Current == nil {
		logger.Warn("[health] Config not loaded")
		return
	}
	redisURL := config.Current.RedisURL
	if redisURL == "" {
		redisURL = "redis:6379"
	}
	if len(redisURL) < 8 || redisURL[0:8] != "redis://" {
		redisURL = "redis://" + redisURL
	}

	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		logger.Error("[health] Failed to parse Redis URL for health checks", err)
		return
	}

	rdb = redis.NewClient(opt)
	redisChecker = &defaultRedisChecker{client: rdb}
}

// CheckRedis checks Redis connectivity
func CheckRedis() HealthStatus {
	if redisChecker == nil {
		return HealthStatus{
			Status:    "unhealthy",
			Message:   "Redis client not initialized",
			Timestamp: time.Now(),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := redisChecker.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		return HealthStatus{
			Status:    "unhealthy",
			Message:   fmt.Sprintf("Redis ping failed: %v", err),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"latency_ms": latency.Milliseconds(),
			},
		}
	}

	status := "healthy"
	if latency > 100*time.Millisecond {
		status = "degraded"
	}

	return HealthStatus{
		Status:    status,
		Message:   "Redis connection healthy",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"latency_ms": latency.Milliseconds(),
			"url":        config.Current.RedisURL,
		},
	}
}

// CheckBuilder checks builder service connectivity
func CheckBuilder() HealthStatus {
	start := time.Now()
	resp, err := httpClient.Get(builderURL)
	latency := time.Since(start)

	if err != nil {
		return HealthStatus{
			Status:    "unhealthy",
			Message:   fmt.Sprintf("Builder service unreachable: %v", err),
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"latency_ms": latency.Milliseconds(),
			},
		}
	}
	defer resp.Body.Close()

	status := "healthy"
	if resp.StatusCode != http.StatusOK {
		status = "unhealthy"
	} else if latency > 200*time.Millisecond {
		status = "degraded"
	}

	return HealthStatus{
		Status:    status,
		Message:   fmt.Sprintf("Builder service responded with status %d", resp.StatusCode),
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
			"latency_ms":  latency.Milliseconds(),
		},
	}
}

// GetSystemMetrics returns system resource metrics
func GetSystemMetrics() map[string]interface{} {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return map[string]interface{}{
		"goroutines":   runtime.NumGoroutine(),
		"memory_alloc": mem.Alloc,
		"memory_sys":   mem.Sys,
		"memory_heap":  mem.HeapAlloc,
		"gc_cycles":    mem.NumGC,
		"cpu_cores":    runtime.NumCPU(),
		"go_version":   runtime.Version(),
	}
}

// GetOverallHealth returns comprehensive system health
func GetOverallHealth() SystemHealth {
	components := map[string]HealthStatus{
		"redis":   CheckRedis(),
		"builder": CheckBuilder(),
	}

	// Determine overall status
	overallStatus := "healthy"
	for _, component := range components {
		if component.Status == "unhealthy" {
			overallStatus = "unhealthy"
			break
		} else if component.Status == "degraded" && overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	}

	return SystemHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(startTime).String(),
		Version:    "1.0.0-dev",
		Components: components,
		System:     GetSystemMetrics(),
	}
}

// HealthHandler provides a comprehensive health endpoint
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := GetOverallHealth()

	w.Header().Set("Content-Type", "application/json")

	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if health.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if health.Status == "degraded" {
		statusCode = http.StatusPartialContent
	}

	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(health); err != nil {
		logger.Error("[health] Failed to encode health response", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// LivenessHandler provides a simple liveness check
func LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"uptime":    time.Since(startTime).String(),
	})
}

// ReadinessHandler checks if the service is ready to serve traffic
func ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	redisHealth := CheckRedis()
	builderHealth := CheckBuilder()

	ready := redisHealth.Status != "unhealthy" && builderHealth.Status != "unhealthy"

	w.Header().Set("Content-Type", "application/json")

	if ready {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ready",
			"timestamp": time.Now(),
			"checks": map[string]string{
				"redis":   redisHealth.Status,
				"builder": builderHealth.Status,
			},
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "not_ready",
			"timestamp": time.Now(),
			"checks": map[string]string{
				"redis":   redisHealth.Status,
				"builder": builderHealth.Status,
			},
		})
	}
}
