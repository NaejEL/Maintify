package pluginmgr

import (
	"strings"
	"testing"
)

func TestGenerateComposeFile(t *testing.T) {
	plugin := PluginMeta{
		Name: "test-plugin",
		Resources: struct {
			MemoryMB      int64 `yaml:"memory_mb" json:"memory_mb"`
			CPUMilliCores int64 `yaml:"cpu_milli_cores" json:"cpu_milli_cores"`
		}{
			MemoryMB:      512,
			CPUMilliCores: 1000,
		},
	}

	imageName := "test-plugin:latest"
	networkName := "maintify_net"

	content, err := GenerateComposeFile(plugin, imageName, networkName)
	if err != nil {
		t.Fatalf("GenerateComposeFile failed: %v", err)
	}

	yamlStr := string(content)

	// Check for key elements in the generated YAML
	expectedStrings := []string{
		"version: \"3.8\"",
		"services:",
		"plugin-test-plugin:",
		"image: test-plugin:latest",
		"container_name: maintify-plugin-test-plugin",
		"networks:",
		"- maintify_net",
		"cpus: \"1.0\"",
		"memory: 512M",
		"PLUGIN_NAME: test-plugin",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(yamlStr, s) {
			t.Errorf("Generated YAML missing expected string: %s\nGot:\n%s", s, yamlStr)
		}
	}
}
