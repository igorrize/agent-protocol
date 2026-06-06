// Package ports defines the domain types and interfaces (the hexagonal "ports").
// It depends only on the standard library; adapters and usecases depend on it.
package ports

// Role is the privilege level resolved from a request's bearer token.
type Role string

const (
	RoleOrchestrator Role = "orchestrator"
	RoleWorker       Role = "worker"
)

// Contract is a registered agent's input/output JSON Schemas plus its work
// tools. AllowedTools holds the agent's own tools; the spawner additionally
// grants listen/complete.
type Contract struct {
	AgentName    string
	Input        map[string]any // input JSON Schema
	Output       map[string]any // output JSON Schema
	AllowedTools []string
}

// Task is a dispatched unit of work.
type Task struct {
	ID     string
	Agent  string
	Params map[string]any
	Prompt string
	Status string // "dispatched" | "completed"
	Output map[string]any
}

// Token is a role-scoped access credential bound to a task.
type Token struct {
	Value  string
	Role   Role
	TaskID string
}

// Event is one audit-log record. Empty fields are omitted from the JSON view.
type Event struct {
	Event  string            `json:"event"`
	Tool   string            `json:"tool,omitempty"`
	Agent  string            `json:"agent,omitempty"`
	TaskID string            `json:"task_id,omitempty"`
	Role   Role              `json:"role,omitempty"`
	Params map[string]any    `json:"params,omitempty"`
	Output map[string]any    `json:"output,omitempty"`
	Errors map[string]string `json:"errors,omitempty"`
	TS     int64             `json:"ts"`
}
