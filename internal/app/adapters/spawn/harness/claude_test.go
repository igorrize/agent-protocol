package harness

import (
	"strings"
	"testing"
)

// argvValue returns the argument following flag, or "".
func argvValue(argv []string, flag string) string {
	for i, a := range argv {
		if a == flag && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return ""
}

func TestClaudeCommand(t *testing.T) {
	argv := Claude{}.Command("/tmp/cfg.json", "do the thing", []string{"Read", "Grep"})

	if argv[0] != "claude" {
		t.Fatalf("argv[0] = %q, want claude", argv[0])
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{"-p", "do the thing", "--strict-mcp-config", "--mcp-config", "/tmp/cfg.json", "--allowedTools"} {
		if !strings.Contains(joined, want) {
			t.Errorf("argv missing %q: %v", want, argv)
		}
	}

	tools := argvValue(argv, "--allowedTools")
	if tools != "Read,Grep,mcp__agent-protocol__listen,mcp__agent-protocol__complete" {
		t.Errorf("allowedTools = %q", tools)
	}
	if strings.Contains(tools, "Task") || strings.Contains(tools, "Bash") {
		t.Errorf("whitelist must not include Task/Bash: %q", tools)
	}

	// deny-list must explicitly block native subagent spawning and shell
	disallowed := argvValue(argv, "--disallowedTools")
	for _, want := range []string{"Agent", "Task", "Bash", "Workflow"} {
		if !strings.Contains(disallowed, want) {
			t.Errorf("--disallowedTools missing %q: %q", want, disallowed)
		}
	}
}

func TestClaudeCommandNoWorkTools(t *testing.T) {
	argv := Claude{}.Command("/tmp/cfg.json", "p", nil)
	if got := argvValue(argv, "--allowedTools"); got != "mcp__agent-protocol__listen,mcp__agent-protocol__complete" {
		t.Errorf("allowedTools = %q", got)
	}
}
