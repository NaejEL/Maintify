//go:build integration

package pluginmgr

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShellComposeRunner_Run exercises the real docker-compose binary.
// This test only runs with: go test -tags integration ./pkg/pluginmgr/...
// It requires docker-compose to be installed and available on PATH.
func TestShellComposeRunner_Run_WithRealDockerCompose(t *testing.T) {
	// Write a minimal compose file that docker-compose can validate/parse
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")
	content := []byte(`version: "3.8"
services:
  test:
    image: hello-world
`)
	assert.NoError(t, os.WriteFile(composePath, content, 0o644))

	runner := &ShellComposeRunner{}
	// "config" validates the compose file without starting containers
	err := runner.Run("test-plugin", composePath, "config", "--quiet")
	assert.NoError(t, err)
}
