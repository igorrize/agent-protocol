package ports

// Store persists contracts, tasks and role-scoped tokens.
type Store interface {
	PutContract(c Contract)
	GetContract(agent string) (Contract, bool)

	// CreateTask records a new task with status "dispatched" and a generated id.
	CreateTask(agent string, params map[string]any, prompt string) Task
	GetTask(id string) (Task, bool)
	// CompleteTask marks the task "completed" with output; ok is false if absent.
	CompleteTask(id string, output map[string]any) (Task, bool)

	CreateToken(role Role, taskID string) string
	TokenInfo(tok string) (Token, bool)
}
