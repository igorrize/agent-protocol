package usecases

import (
	"context"

	"agent-protocol/internal/app/ports"
	"agent-protocol/internal/app/schema"
)

// DispatchRequest dispatches a task to an agent.
type DispatchRequest struct {
	AgentName string
	Params    map[string]any
	Prompt    string
}

// DispatchResponse is the reply for a successful dispatch.
type DispatchResponse struct {
	Status      string `json:"status"` // "dispatched"
	TaskID      string `json:"task_id"`
	WorkerToken string `json:"worker_token"`
}

// Dispatch validates params against the agent's input contract (pass-through if
// none), creates a task and worker token, optionally spawns a locked worker,
// and returns the task id and token.
func (c *Commands) Dispatch(_ context.Context, req DispatchRequest) (*DispatchResponse, error) {
	contract, hasContract := c.store.GetContract(req.AgentName)
	if hasContract {
		if errs := schema.Validate(contract.Input, req.Params); len(errs) > 0 {
			c.audit.Log(ports.Event{Event: "rejected", Tool: "dispatch", Agent: req.AgentName, Errors: errs})
			return nil, &ports.ValidationError{Errors: errs}
		}
	}

	task := c.store.CreateTask(req.AgentName, req.Params, req.Prompt)
	wtok := c.store.CreateToken(ports.RoleWorker, task.ID)
	c.audit.Log(ports.Event{Event: "dispatched", Tool: "dispatch", Agent: req.AgentName, TaskID: task.ID})

	if c.spawnWorkers {
		// Fire-and-forget; a spawn failure is logged, not surfaced to the caller.
		if err := c.spawner.Spawn(task, wtok, contract.AllowedTools); err != nil {
			c.log.Error("spawn worker", "task", task.ID, "err", err)
		}
	}

	return &DispatchResponse{Status: "dispatched", TaskID: task.ID, WorkerToken: wtok}, nil
}
