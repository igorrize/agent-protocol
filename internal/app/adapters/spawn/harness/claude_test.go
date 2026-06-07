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

// hasFlag reports whether flag appears in argv.
func hasFlag(argv []string, flag string) bool {
	for _, a := range argv {
		if a == flag {
			return true
		}
	}
	return false
}

func TestClaudeWorkerProfile(t *testing.T) {
	argv := Claude{}.Command(Worker, "/tmp/cfg.json", "do the thing", []string{"Read", "Grep"})

	if argv[0] != "claude" {
		t.Fatalf("argv[0] = %q, want claude", argv[0])
	}
	if !hasFlag(argv, "-p") || argvValue(argv, "-p") != "do the thing" {
		t.Errorf("worker must run headless with -p prompt: %v", argv)
	}
	if argvValue(argv, "--mcp-config") != "/tmp/cfg.json" || !hasFlag(argv, "--strict-mcp-config") {
		t.Errorf("locked mcp config missing: %v", argv)
	}
	if got := argvValue(argv, "--tools"); got != "Read,Grep" {
		t.Errorf("--tools = %q, want Read,Grep", got)
	}
	if got := argvValue(argv, "--allowedTools"); got != "mcp__agent-protocol__listen,mcp__agent-protocol__complete" {
		t.Errorf("--allowedTools = %q", got)
	}
	assertDenyList(t, argv)
}

func TestClaudeOrchestratorProfile(t *testing.T) {
	argv := Claude{}.Command(Orchestrator, "/tmp/cfg.json", "", nil)

	if hasFlag(argv, "-p") {
		t.Error("orchestrator must be interactive (no -p)")
	}
	if got := argvValue(argv, "--tools"); got != "" {
		t.Errorf("orchestrator --tools = %q, want empty (all built-in disabled)", got)
	}
	allowed := argvValue(argv, "--allowedTools")
	if allowed != "mcp__agent-protocol__dispatch,mcp__agent-protocol__listen,mcp__agent-protocol__audit" {
		t.Errorf("--allowedTools = %q", allowed)
	}
	if strings.Contains(allowed, "complete") {
		t.Errorf("orchestrator must not have complete: %q", allowed)
	}
	assertDenyList(t, argv)
}

func assertDenyList(t *testing.T, argv []string) {
	t.Helper()
	disallowed := argvValue(argv, "--disallowedTools")
	for _, want := range []string{"Agent", "Task", "Bash", "Workflow"} {
		if !strings.Contains(disallowed, want) {
			t.Errorf("--disallowedTools missing %q: %q", want, disallowed)
		}
	}
}
