// Package store provides an in-memory ports.Store.
package store

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"agent-protocol/internal/app/ports"
)

// Memory is an in-memory ports.Store backed by maps under an RWMutex.
// State is lost on restart (MVP).
type Memory struct {
	mu        sync.RWMutex
	contracts map[string]ports.Contract
	tasks     map[string]ports.Task
	tokens    map[string]ports.Token
	log       ports.Logger
}

// NewMemory returns an empty in-memory store.
func NewMemory(logger ports.Logger) *Memory {
	return &Memory{
		contracts: map[string]ports.Contract{},
		tasks:     map[string]ports.Task{},
		tokens:    map[string]ports.Token{},
		log:       logger,
	}
}

// --- contracts ---

func (m *Memory) PutContract(c ports.Contract) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.contracts[c.AgentName] = c
}

func (m *Memory) GetContract(agent string) (ports.Contract, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.contracts[agent]
	return c, ok
}

// LoadContracts registers every <dir>/*/contract.json file. Missing dir or a
// bad file is logged and skipped, not fatal.
func (m *Memory) LoadContracts(dir string) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*", "contract.json"))
	if err != nil {
		return err
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			m.log.Error("read contract", "path", path, "err", err)
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			m.log.Error("parse contract", "path", path, "err", err)
			continue
		}
		c := contractFromJSON(raw)
		m.PutContract(c)
		m.log.Info("loaded contract", "agent", c.AgentName)
	}
	return nil
}

// contractFromJSON builds a Contract from a decoded contract.json map.
func contractFromJSON(raw map[string]any) ports.Contract {
	c := ports.Contract{AgentName: asString(raw["agent_name"])}
	if in, ok := raw["input_schema"].(map[string]any); ok {
		c.Input = in
	}
	if out, ok := raw["output_schema"].(map[string]any); ok {
		c.Output = out
	}
	c.AllowedTools = asStringSlice(raw["allowed_tools"])
	return c
}

// --- tasks ---

func (m *Memory) CreateTask(agent string, params map[string]any, prompt string) ports.Task {
	t := ports.Task{
		ID:     newID("task_", 8),
		Agent:  agent,
		Params: params,
		Prompt: prompt,
		Status: "dispatched",
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[t.ID] = t
	return t
}

func (m *Memory) GetTask(id string) (ports.Task, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	return t, ok
}

func (m *Memory) CompleteTask(id string, output map[string]any) (ports.Task, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.tasks[id]
	if !ok {
		return ports.Task{}, false
	}
	t.Status = "completed"
	t.Output = output
	m.tasks[id] = t
	return t, true
}

// --- tokens ---

func (m *Memory) CreateToken(role ports.Role, taskID string) string {
	tok := newID("tok_", 12)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tokens[tok] = ports.Token{Value: tok, Role: role, TaskID: taskID}
	return tok
}

func (m *Memory) TokenInfo(tok string) (ports.Token, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tokens[tok]
	return t, ok
}

// --- helpers ---

// newID returns prefix followed by n hex chars from a fresh random UUID.
func newID(prefix string, n int) string {
	u := uuid.New()
	return prefix + hex.EncodeToString(u[:])[:n]
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asStringSlice(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, e := range arr {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
