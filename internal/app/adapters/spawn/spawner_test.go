package spawn

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"agent-protocol/internal/app/ports"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// fakeHarness records its inputs and returns a preset command.
type fakeHarness struct {
	configPath string
	prompt     string
	allowed    []string
	command    []string
}

func (h *fakeHarness) Command(configPath, prompt string, allowedTools []string) []string {
	h.configPath = configPath
	h.prompt = prompt
	h.allowed = allowedTools
	return h.command
}

func TestSpawnWritesLockedConfigAndStarts(t *testing.T) {
	h := &fakeHarness{command: []string{"true"}} // harmless, exits 0
	s := New(h, "http://localhost:4321/mcp", noopLogger{})
	task := ports.Task{ID: "task_abc12345", Agent: "researcher"}

	if err := s.Spawn(task, "tok_secret", []string{"Read", "Grep"}); err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	t.Cleanup(func() {
		os.Remove(h.configPath)
		os.Remove("/tmp/ap-worker-task_abc12345.log")
	})

	// harness received the task prompt and work tools
	if !strings.Contains(h.prompt, "task_abc12345") {
		t.Errorf("prompt missing task id: %q", h.prompt)
	}
	if len(h.allowed) != 2 || h.allowed[0] != "Read" {
		t.Errorf("allowed tools = %v", h.allowed)
	}

	// locked config content: only our proxy, authed with the worker token
	data, err := os.ReadFile(h.configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	servers := cfg["mcpServers"].(map[string]any)
	if len(servers) != 1 {
		t.Errorf("want exactly one mcp server, got %d", len(servers))
	}
	srv := servers["agent-protocol"].(map[string]any)
	if srv["type"] != "http" || srv["url"] != "http://localhost:4321/mcp" {
		t.Errorf("server config = %v", srv)
	}
	if auth := srv["headers"].(map[string]any)["Authorization"]; auth != "Bearer tok_secret" {
		t.Errorf("auth header = %v", auth)
	}
}

func TestSpawnEmptyCommandErrors(t *testing.T) {
	h := &fakeHarness{command: nil}
	s := New(h, "http://x", noopLogger{})
	err := s.Spawn(ports.Task{ID: "task_x"}, "tok", nil)
	if err == nil {
		t.Error("empty command should error")
	}
	t.Cleanup(func() { os.Remove("/tmp/ap-worker-task_x.log") })
}
