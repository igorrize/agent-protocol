// Package orchestrate launches a locked, interactive Claude orchestrator that
// can reach other agents ONLY via the proxy's dispatch/listen/audit tools.
package orchestrate

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"agent-protocol/internal/app/adapters/spawn"
	"agent-protocol/internal/app/adapters/spawn/harness"
	"agent-protocol/pkg/infra/config"
)

// Run launches the locked orchestrator, blocking until the claude session exits.
func Run(args []string) error {
	fs := flag.NewFlagSet("orchestrate", flag.ContinueOnError)
	proxyURL := fs.String("proxy-url", "", "proxy URL (overrides PROXY_URL env)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	url := config.Load().ProxyURL
	if *proxyURL != "" {
		url = *proxyURL
	}

	// No token -> orchestrator role at the proxy.
	cfgPath, err := spawn.WriteLockedConfig(url, "")
	if err != nil {
		return fmt.Errorf("write locked config: %w", err)
	}
	defer os.Remove(cfgPath)

	argv := harness.Claude{}.Command(harness.Orchestrator, cfgPath, "", nil)
	cmd := exec.Command(argv[0], argv[1:]...) //nolint:gosec // argv is built from a fixed harness template
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
