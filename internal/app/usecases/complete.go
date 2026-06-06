package usecases

import (
	"context"

	"agent-protocol/internal/app/ports"
	"agent-protocol/internal/app/schema"
)

// CompleteRequest completes a task with its output.
type CompleteRequest struct {
	TaskID string
	Output map[string]any
}

// CompleteResponse is the reply for a successful complete.
type CompleteResponse struct {
	Status string `json:"status"` // "accepted"
	TaskID string `json:"task_id"`
}

// Complete validates output against the task's agent contract (pass-through if
// none), marks the task completed, and acknowledges.
func (c *Commands) Complete(_ context.Context, req CompleteRequest) (*CompleteResponse, error) {
	task, ok := c.store.GetTask(req.TaskID)
	if !ok {
		return nil, &ports.NotFoundError{Kind: "task", ID: req.TaskID}
	}

	if contract, ok := c.store.GetContract(task.Agent); ok {
		if errs := schema.Validate(contract.Output, req.Output); len(errs) > 0 {
			c.audit.Log(ports.Event{Event: "rejected", Tool: "complete", Agent: task.Agent, TaskID: req.TaskID, Errors: errs})
			return nil, &ports.ValidationError{Errors: errs}
		}
	}

	c.store.CompleteTask(req.TaskID, req.Output)
	c.audit.Log(ports.Event{Event: "completed", Tool: "complete", Agent: task.Agent, TaskID: req.TaskID})
	return &CompleteResponse{Status: "accepted", TaskID: req.TaskID}, nil
}
