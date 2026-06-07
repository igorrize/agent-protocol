package spawn

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"agent-protocol/internal/app/adapters/spawn/harness"
	"agent-protocol/internal/app/ports"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// fakeHarness records its inputs and returns a preset command.
type fakeHarness struct {
	profile    harness.Profile
	configPath string
	prompt     string
	tools      []string
	command    []string
}

func (h *fakeHarness) Command(profile harness.Profile, configPath, prompt string, tools []string) []string {
	h.profile = profile
	h.configPath = configPath
	h.prompt = prompt
	h.tools = tools
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

	// harness invoked with the worker profile, task prompt, and work tools
	if h.profile != harness.Worker {
		t.Errorf("profile = %v, want Worker", h.profile)
	}
	if !strings.Contains(h.prompt, "task_abc12345") {
		t.Errorf("prompt missing task id: %q", h.prompt)
	}
	if len(h.tools) != 2 || h.tools[0] != "Read" {
		t.Errorf("work tools = %v", h.tools)
	}

	assertLockedConfig(t, h.configPath, "Bearer tok_secret")
}

func TestSpawnEmptyCommandErrors(t *testing.T) {
	h := &fakeHarness{command: nil}
	s := New(h, "http://x", noopLogger{})
	if err := s.Spawn(ports.Task{ID: "task_x"}, "tok", nil); err == nil {
		t.Error("empty command should error")
	}
	t.Cleanup(func() { os.Remove("/tmp/ap-worker-task_x.log") })
}

func TestWriteLockedConfigWorkerHasToken(t *testing.T) {
	path, err := WriteLockedConfig("http://localhost:4321/mcp", "tok_w")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })
	assertLockedConfig(t, path, "Bearer tok_w")
}

func TestWriteLockedConfigOrchestratorHasNoToken(t *testing.T) {
	path, err := WriteLockedConfig("http://localhost:4321/mcp", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(path) })

	srv := readServer(t, path)
	if _, hasHeaders := srv["headers"]; hasHeaders {
		t.Errorf("orchestrator config must omit Authorization header: %v", srv)
	}
	if srv["url"] != "http://localhost:4321/mcp" || srv["type"] != "http" {
		t.Errorf("server config = %v", srv)
	}
}

// assertLockedConfig verifies the file holds exactly one server (the proxy)
// authed with wantAuth.
func assertLockedConfig(t *testing.T, path, wantAuth string) {
	t.Helper()
	srv := readServer(t, path)
	if srv["type"] != "http" {
		t.Errorf("server type = %v", srv["type"])
	}
	auth := srv["headers"].(map[string]any)["Authorization"]
	if auth != wantAuth {
		t.Errorf("auth = %v, want %q", auth, wantAuth)
	}
}

func readServer(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse config: %v", err)
	}
	servers := cfg["mcpServers"].(map[string]any)
	if len(servers) != 1 {
		t.Fatalf("want exactly one mcp server, got %d", len(servers))
	}
	return servers["agent-protocol"].(map[string]any)
}
