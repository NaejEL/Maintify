// shell_runner.go contains ShellComposeRunner.Run — the only function in this
// package that requires a real docker-compose binary. It is tested via the
// integration build tag (lifecycle_integration_test.go). Unit tests inject
// fakeComposeRunner instead, so this file is intentionally excluded from the
// unit-coverage profile with:
//
//	go test -coverprofile=... -coverpkg=... -run . -covermode=atomic \
//	    $(go list ./... | grep -v 'shell_runner')
//
// or by acknowledging the one 0% function in the CI gate.

package pluginmgr

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

// Run executes docker-compose with the given sub-command against the compose
// file at composePath. It is the production implementation of ComposeRunner.
func (r *ShellComposeRunner) Run(pluginName string, composePath string, args ...string) error {
	cmdArgs := append([]string{"-f", composePath}, args...)
	cmd := exec.Command("docker-compose", cmdArgs...) // #nosec G204 -- composePath is generated internally, args are lifecycle constants (up/down/restart)
	cmd.Dir = filepath.Dir(composePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker-compose failed for plugin %s: %w: %s", pluginName, err, string(output))
	}
	return nil
}
