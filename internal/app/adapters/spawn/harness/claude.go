package harness

import "strings"

// disallowedTools are native capabilities a locked agent must never have:
// spawning subagents (Agent/Task), shell access (Bash), or native orchestration
// (Workflow). Defense-in-depth on top of --tools/--allowedTools — agents reach
// each other only through the proxy.
const disallowedTools = "Agent,Task,Bash,Workflow"

// Proxy MCP tool names.
const (
	mcpDispatch = "mcp__agent-protocol__dispatch"
	mcpListen   = "mcp__agent-protocol__listen"
	mcpComplete = "mcp__agent-protocol__complete"
	mcpAudit    = "mcp__agent-protocol__audit"
)

// Claude is the Claude Code harness (the reference). It locks an agent down to
// a single MCP config, a built-in tool set (--tools, "" disables all), an
// allow-list of proxy tools, and an explicit deny-list.
type Claude struct{}

// Command builds the claude argv for the profile.
func (Claude) Command(profile Profile, configPath, prompt string, tools []string) []string {
	argv := []string{"claude"}
	if profile == Worker {
		argv = append(argv, "-p", prompt) // headless; orchestrator stays interactive
	}
	argv = append(argv,
		"--strict-mcp-config",
		"--mcp-config", configPath,
		"--tools", strings.Join(tools, ","), // "" => all built-in tools disabled
	)

	switch profile {
	case Worker:
		argv = append(argv, "--allowedTools", strings.Join([]string{mcpListen, mcpComplete}, ","))
	case Orchestrator:
		argv = append(argv, "--allowedTools", strings.Join([]string{mcpDispatch, mcpListen, mcpAudit}, ","))
	}

	return append(argv, "--disallowedTools", disallowedTools)
}
