// Package harness builds the per-CLI argv for launching a locked agent. The
// lock-down flags for every profile live here, in one place.
package harness

// Profile selects the lock-down flag set for a launched agent.
type Profile int

const (
	// Worker runs headless (-p prompt): the contract's work tools plus the
	// proxy's listen/complete.
	Worker Profile = iota
	// Orchestrator runs interactively: no built-in tools, plus the proxy's
	// dispatch/listen/audit.
	Orchestrator
)

// Harness builds the command line to launch a locked agent for a profile,
// given the locked MCP config path, a prompt (worker only), and the agent's
// built-in work tools (worker only).
type Harness interface {
	Command(profile Profile, configPath, prompt string, tools []string) []string
}
