// Package harness builds the per-CLI argv for launching a locked worker agent.
package harness

// Harness builds the command line to launch a locked worker.
type Harness interface {
	// Command returns the argv to run, given the locked MCP config path, the
	// worker prompt, and the agent's work tools. The harness always also grants
	// the proxy's listen/complete tools.
	Command(configPath, prompt string, allowedTools []string) []string
}
