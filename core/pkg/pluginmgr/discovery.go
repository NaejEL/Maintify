package pluginmgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	IssueTypeFilesystem = "filesystem"
	IssueTypeMetadata   = "metadata"
	IssueTypeConflict   = "conflict"
)

type DiscoveryIssue struct {
	Type    string `json:"type"`
	Plugin  string `json:"plugin,omitempty"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

func discoverPluginsInDir(pluginDir string) ([]PluginMeta, []DiscoveryIssue) {
	found := make([]PluginMeta, 0)
	issues := make([]DiscoveryIssue, 0)

	entries, err := os.ReadDir(pluginDir)
	if err != nil {
		issues = append(issues, DiscoveryIssue{
			Type:    IssueTypeFilesystem,
			Path:    pluginDir,
			Message: fmt.Sprintf("failed to read plugin directory: %v", err),
		})
		return found, issues
	}

	nameIndex := make(map[string]struct{})
	routeIndex := make(map[string]struct{})

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginPath := filepath.Join(pluginDir, entry.Name())
		metaPath := filepath.Join(pluginPath, "plugin.yaml")
		data, readErr := os.ReadFile(metaPath) // #nosec G304 -- metaPath constructed from admin-controlled plugin directory
		if readErr != nil {
			issues = append(issues, DiscoveryIssue{
				Type:    IssueTypeMetadata,
				Plugin:  entry.Name(),
				Path:    metaPath,
				Message: fmt.Sprintf("invalid metadata: %v", readErr),
			})
			continue
		}

		var meta PluginMeta
		if unmarshalErr := yaml.Unmarshal(data, &meta); unmarshalErr != nil {
			issues = append(issues, DiscoveryIssue{
				Type:    IssueTypeMetadata,
				Plugin:  entry.Name(),
				Path:    metaPath,
				Message: fmt.Sprintf("invalid metadata: %v", unmarshalErr),
			})
			continue
		}

		if strings.TrimSpace(meta.Name) == "" || strings.TrimSpace(meta.Version) == "" || strings.TrimSpace(meta.Route) == "" {
			issues = append(issues, DiscoveryIssue{
				Type:    IssueTypeMetadata,
				Plugin:  entry.Name(),
				Path:    metaPath,
				Message: "invalid metadata: required fields name, version, and route must be set",
			})
			continue
		}

		if _, exists := nameIndex[meta.Name]; exists {
			issues = append(issues, DiscoveryIssue{
				Type:    IssueTypeConflict,
				Plugin:  meta.Name,
				Path:    metaPath,
				Message: fmt.Sprintf("plugin name conflict: %s", meta.Name),
			})
			continue
		}

		if _, exists := routeIndex[meta.Route]; exists {
			issues = append(issues, DiscoveryIssue{
				Type:    IssueTypeConflict,
				Plugin:  meta.Name,
				Path:    metaPath,
				Message: fmt.Sprintf("plugin route conflict: %s", meta.Route),
			})
			continue
		}

		nameIndex[meta.Name] = struct{}{}
		routeIndex[meta.Route] = struct{}{}
		found = append(found, meta)
	}

	return found, issues
}
