package pluginmgr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"maintify/core/pkg/config"
)

// BuilderClient abstracts calls to the builder service so the logic is
// independently testable without a real Docker or builder process.
type BuilderClient interface {
	BuildImage(pluginName, contextDir string) (imageName string, err error)
}

// HTTPBuilderClient is the production implementation that calls the real service.
type HTTPBuilderClient struct {
	URL    string
	APIKey string
	client *http.Client
}

func NewHTTPBuilderClient() *HTTPBuilderClient {
	apiKey := ""
	if config.Current != nil {
		apiKey = config.Current.BuilderAPIKey
	}
	return &HTTPBuilderClient{
		URL:    "http://builder:8081/build",
		APIKey: apiKey,
		client: &http.Client{},
	}
}

// NewHTTPBuilderClientWithKey constructs a client with an explicit API key,
// useful when config has already been resolved by the caller.
func NewHTTPBuilderClientWithKey(apiKey string) *HTTPBuilderClient {
	return &HTTPBuilderClient{
		URL:    "http://builder:8081/build",
		APIKey: apiKey,
		client: &http.Client{},
	}
}

func (b *HTTPBuilderClient) BuildImage(pluginName, contextDir string) (string, error) {
	req := BuildRequest{PluginName: pluginName, ContextDir: contextDir}
	body, _ := json.Marshal(req) // BuildRequest contains only strings; Marshal never fails

	httpReq, _ := http.NewRequest("POST", b.URL, bytes.NewBuffer(body)) // static URL; NewRequest never fails
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+b.APIKey)

	resp, err := b.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call builder service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("builder returned HTTP %d: %s", resp.StatusCode, string(raw))
	}

	var buildResp BuildResponse
	if err := json.NewDecoder(resp.Body).Decode(&buildResp); err != nil {
		return "", fmt.Errorf("decode builder response: %w", err)
	}
	if buildResp.Status != "success" {
		return "", fmt.Errorf("builder reported failure for plugin %s", pluginName)
	}
	return buildResp.Image, nil
}

// LaunchOptions groups the runtime parameters needed to launch plugin containers.
type LaunchOptions struct {
	PluginDir   string
	NetworkName string
}

// DefaultLaunchOptions returns production-appropriate defaults.
func DefaultLaunchOptions() LaunchOptions {
	return LaunchOptions{
		PluginDir:   "/plugins",
		NetworkName: "newmaintify_default",
	}
}

// LaunchPluginContainers builds images via builderClient and starts each plugin
// container with composeRunner. This function is fully testable via fake impls.
func LaunchPluginContainers(
	discovered []PluginMeta,
	builderClient BuilderClient,
	composeRunner ComposeRunner,
	opts LaunchOptions,
) {
	log.Printf("[pluginmgr] Launching %d plugin(s) using Docker Compose…", len(discovered))

	for _, plugin := range discovered {
		backendDir := filepath.Join(opts.PluginDir, plugin.Name, "backend")
		dockerfilePath := filepath.Join(backendDir, "Dockerfile")

		if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
			log.Printf("[pluginmgr] No Dockerfile for %s, skipping", plugin.Name)
			continue
		}

		imageName, err := builderClient.BuildImage(plugin.Name, backendDir)
		if err != nil {
			log.Printf("[pluginmgr] Build failed for %s: %v", plugin.Name, err)
			continue
		}
		log.Printf("[pluginmgr] Built image: %s", imageName)

		if config.Current != nil && !config.Current.EnableContainerMode {
			log.Printf("[pluginmgr] Container mode disabled — image %s ready for external orchestration", imageName)
			continue
		}

		composeContent, _ := GenerateComposeFile(plugin, imageName, opts.NetworkName) // yaml.Marshal on a plain struct never fails

		composePath := filepath.Join(opts.PluginDir, plugin.Name, "docker-compose.yml")
		if err := os.WriteFile(composePath, composeContent, 0o640); err != nil { // #nosec G306 -- 0640 is intentional: compose files should not be world-readable
			log.Printf("[pluginmgr] Failed to write compose file for %s: %v", plugin.Name, err)
			continue
		}

		if err := composeRunner.Run(plugin.Name, composePath, "up", "-d"); err != nil {
			log.Printf("[pluginmgr] Failed to start %s: %v", plugin.Name, err)
			continue
		}
		log.Printf("[pluginmgr] Started plugin %s", plugin.Name)
	}
}
