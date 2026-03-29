package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"maintify/core/pkg/logger"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

type BuildRequest struct {
	PluginName string `json:"plugin_name"`
	ContextDir string `json:"context_dir"`
}

var apiKey string

func init() {
	// Generate or load API key
	apiKey = os.Getenv("BUILDER_API_KEY")
	if apiKey == "" {
		// Generate a random API key if none provided
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			log.Fatalf("Failed to generate API key: %v", err)
		}
		apiKey = hex.EncodeToString(bytes)
		log.Printf("Generated API key: %s", apiKey)
		log.Println("Set BUILDER_API_KEY environment variable to use a specific key")
	} else {
		log.Println("Using API key from environment variable")
	}
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		// Support both "Bearer token" and "token" formats
		token := authHeader
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if token != apiKey {
			log.Printf("Invalid API key attempt from %s", r.RemoteAddr)
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "maintify-builder",
	})
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	// Initialize logging system
	logConfig := logger.Config{
		Level:       getEnvWithDefault("LOG_LEVEL", "INFO"),
		Component:   "builder",
		Structured:  getEnvWithDefault("LOG_STRUCTURED", "true") == "true",
		LogDir:      getEnvWithDefault("LOG_DIR", "/var/log/maintify"),
		MaxFileSize: 10 * 1024 * 1024, // 10MB
		MaxFiles:    5,
		Console:     true,
	}

	err := logger.InitDefaultLogger(logConfig)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger.LogSystemEvent("service_start", "Maintify Builder service starting", map[string]interface{}{
		"version": "1.0.0",
		"port":    8081,
	})

	log.Println("Starting Maintify Builder Service on :8081")

	// Health check endpoint (no auth required)
	http.HandleFunc("/health", healthHandler)

	// Protected build endpoint
	http.HandleFunc("/build", authMiddleware(buildHandler))

	// Protected cleanup endpoint
	http.HandleFunc("/cleanup", authMiddleware(cleanupHandler))

	srv := &http.Server{
		Addr:              ":8081",
		Handler:           nil,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func buildHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error("Invalid build request body", err, map[string]interface{}{
			"endpoint": "/build",
			"method":   r.Method,
		})
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	logger.LogSystemEvent("build_start", "Plugin build started", map[string]interface{}{
		"plugin_name": req.PluginName,
		"context_dir": req.ContextDir,
	})

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, "Failed to create Docker client", http.StatusInternalServerError)
		return
	}

	buildCtx, err := archive.TarWithOptions(req.ContextDir, &archive.TarOptions{})
	if err != nil {
		logger.Error("Failed to create build context", err, map[string]interface{}{
			"plugin_name": req.PluginName,
			"context_dir": req.ContextDir,
		})
		http.Error(w, "Failed to create build context", http.StatusInternalServerError)
		return
	}

	imageName := "maintify-plugin-" + req.PluginName
	buildOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{imageName},
		Remove:     true,
	}

	buildResp, err := cli.ImageBuild(context.Background(), buildCtx, buildOptions)
	if err != nil {
		logger.Error("Failed to build image", err, map[string]interface{}{
			"plugin_name": req.PluginName,
			"image_name":  imageName,
		})
		http.Error(w, "Failed to build image", http.StatusInternalServerError)
		return
	}
	defer buildResp.Body.Close()

	// Stream build output to builder logs
	_, err = io.Copy(os.Stdout, buildResp.Body)
	if err != nil {
		logger.Error("Error streaming build output", err, map[string]interface{}{
			"plugin_name": req.PluginName,
		})
	}

	logger.LogSystemEvent("build_success", "Successfully built plugin image", map[string]interface{}{
		"plugin_name": req.PluginName,
		"image_name":  imageName,
	})

	// Clean up old images for this plugin to prevent disk space issues
	go cleanupOldImages(req.PluginName)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success", "image": imageName})
}

// cleanupOldImages removes old images for a specific plugin to save disk space.
// It retains the 3 most recent builds and deletes the rest.
func cleanupOldImages(pluginName string) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Failed to create Docker client for cleanup: %v", err)
		return
	}

	images, err := cli.ImageList(context.Background(), image.ListOptions{All: true})
	if err != nil {
		log.Printf("Failed to list images for cleanup: %v", err)
		return
	}

	targetPrefix := fmt.Sprintf("maintify-plugin-%s", pluginName)
	var pluginImages []image.Summary
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if strings.HasPrefix(tag, targetPrefix) {
				pluginImages = append(pluginImages, img)
				break
			}
		}
	}

	const maxImages = 3
	if len(pluginImages) <= maxImages {
		log.Printf("Image cleanup: %d images for %s, no cleanup needed", len(pluginImages), pluginName)
		return
	}

	sort.Slice(pluginImages, func(i, j int) bool {
		return pluginImages[i].Created > pluginImages[j].Created
	})

	removedCount := 0
	for i := maxImages; i < len(pluginImages); i++ {
		img := pluginImages[i]
		_, err := cli.ImageRemove(context.Background(), img.ID, image.RemoveOptions{
			Force:         true,
			PruneChildren: true,
		})
		if err != nil {
			log.Printf("Failed to remove image %s: %v", img.ID[:12], err)
		} else {
			removedCount++
			log.Printf("Removed old image: %s (created: %s)", img.ID[:12], time.Unix(img.Created, 0).Format(time.RFC3339))
		}
	}

	if removedCount > 0 {
		log.Printf("Image cleanup for %s: removed %d old image(s)", pluginName, removedCount)
	}
}

// cleanupHandler provides an endpoint for manual image cleanup.
func cleanupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	pluginName := r.URL.Query().Get("plugin")
	go func() {
		if pluginName != "" {
			log.Printf("Manual cleanup triggered for plugin: %s", pluginName)
			cleanupOldImages(pluginName)
		} else {
			log.Printf("Manual cleanup triggered for all plugins")
			cleanupAllImages()
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "cleanup_started",
		"plugin": pluginName,
	})
}

// cleanupAllImages cleans up images for all known plugins.
func cleanupAllImages() {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		log.Printf("Failed to create Docker client for full cleanup: %v", err)
		return
	}

	images, err := cli.ImageList(context.Background(), image.ListOptions{All: true})
	if err != nil {
		log.Printf("Failed to list all images for cleanup: %v", err)
		return
	}

	pluginImageMap := make(map[string]struct{})
	for _, img := range images {
		for _, tag := range img.RepoTags {
			if strings.HasPrefix(tag, "maintify-plugin-") {
				name := strings.TrimPrefix(tag, "maintify-plugin-")
				if idx := strings.Index(name, ":"); idx != -1 {
					name = name[:idx]
				}
				pluginImageMap[name] = struct{}{}
				break
			}
		}
	}

	for name := range pluginImageMap {
		log.Printf("Cleaning up images for plugin: %s", name)
		cleanupOldImages(name)
	}
	log.Printf("Full image cleanup completed for all plugins")
}
