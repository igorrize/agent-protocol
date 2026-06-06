package harness

import "strings"

// disallowedTools are native capabilities a locked worker must never have:
// spawning subagents (Agent/Task), shell access (Bash), or native orchestration
// (Workflow). This is defense-in-depth on top of the allowedTools whitelist —
// the worker can only reach other agents through the proxy.
const disallowedTools = "Agent,Task,Bash,Workflow"

// Claude is the Claude Code harness (the reference). It runs headless with a
// single locked MCP config, a whitelisted tool set, and an explicit deny-list.
type Claude struct{}

// Command builds the claude argv. listen/complete are always appended to the
// agent's own work tools; the deny-list is always applied.
func (Claude) Command(configPath, prompt string, allowedTools []string) []string {
	tools := append(append([]string{}, allowedTools...),
		"mcp__agent-protocol__listen",
		"mcp__agent-protocol__complete",
	)
	return []string{
		"claude", "-p", prompt,
		"--strict-mcp-config",
		"--mcp-config", configPath,
		"--allowedTools", strings.Join(tools, ","),
		"--disallowedTools", disallowedTools,
	}
}
