package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-protocol/internal/app/ports"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

func TestPutGetContract(t *testing.T) {
	m := NewMemory(noopLogger{})
	if _, ok := m.GetContract("x"); ok {
		t.Fatal("empty store should miss")
	}
	m.PutContract(ports.Contract{AgentName: "x", AllowedTools: []string{"Read"}})
	c, ok := m.GetContract("x")
	if !ok || len(c.AllowedTools) != 1 || c.AllowedTools[0] != "Read" {
		t.Fatalf("GetContract = %+v, %v", c, ok)
	}
}

func TestCreateAndCompleteTask(t *testing.T) {
	m := NewMemory(noopLogger{})
	task := m.CreateTask("researcher", map[string]any{"ticket": "X"}, "find bug")
	if !strings.HasPrefix(task.ID, "task_") || len(task.ID) != len("task_")+8 {
		t.Errorf("task id = %q", task.ID)
	}
	if task.Status != "dispatched" {
		t.Errorf("status = %q, want dispatched", task.Status)
	}

	got, ok := m.GetTask(task.ID)
	if !ok || got.Agent != "researcher" || got.Prompt != "find bug" {
		t.Errorf("GetTask = %+v, %v", got, ok)
	}

	done, ok := m.CompleteTask(task.ID, map[string]any{"bug_file": "x.go"})
	if !ok || done.Status != "completed" || done.Output["bug_file"] != "x.go" {
		t.Errorf("CompleteTask = %+v, %v", done, ok)
	}

	if _, ok := m.CompleteTask("task_nope", nil); ok {
		t.Error("CompleteTask on unknown id should report ok=false")
	}
}

func TestTokens(t *testing.T) {
	m := NewMemory(noopLogger{})
	tok := m.CreateToken(ports.RoleWorker, "task_123")
	if !strings.HasPrefix(tok, "tok_") || len(tok) != len("tok_")+12 {
		t.Errorf("token = %q", tok)
	}
	info, ok := m.TokenInfo(tok)
	if !ok || info.Role != ports.RoleWorker || info.TaskID != "task_123" {
		t.Errorf("TokenInfo = %+v, %v", info, ok)
	}
	if _, ok := m.TokenInfo("tok_nope"); ok {
		t.Error("unknown token should report ok=false")
	}
}

func TestLoadContracts(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "clj-reviewer")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	contract := `{"agent_name":"clj-reviewer","allowed_tools":["Read","Glob","Grep"],` +
		`"input_schema":{"required":["files","rubric"]},"output_schema":{"required":["summary"]}}`
	if err := os.WriteFile(filepath.Join(agentDir, "contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatal(err)
	}

	m := NewMemory(noopLogger{})
	if err := m.LoadContracts(dir); err != nil {
		t.Fatalf("LoadContracts: %v", err)
	}

	c, ok := m.GetContract("clj-reviewer")
	if !ok {
		t.Fatal("contract not loaded")
	}
	if len(c.AllowedTools) != 3 || c.AllowedTools[0] != "Read" {
		t.Errorf("allowed tools = %v", c.AllowedTools)
	}
	if _, ok := c.Input["required"]; !ok {
		t.Errorf("input schema not preserved: %v", c.Input)
	}
}

func TestLoadContractsMissingDir(t *testing.T) {
	m := NewMemory(noopLogger{})
	if err := m.LoadContracts(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Errorf("missing dir should be a no-op, got %v", err)
	}
}
