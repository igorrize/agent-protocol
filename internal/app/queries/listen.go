package queries

import (
	"context"
	"time"

	"agent-protocol/internal/app/ports"
)

const (
	longPollTimeout = 25 * time.Second
	pollInterval    = 400 * time.Millisecond
)

// ListenRequest reads a task's current state. When Wait is set and the task is
// still "dispatched", Listen long-polls until the task completes, the long-poll
// timeout elapses, or the context is cancelled.
//
// Only an orchestrator (polling for the result) should set Wait. A worker MUST
// NOT: it calls listen to FETCH the assignment it is about to run, so blocking
// until completion would deadlock its own task.
type ListenRequest struct {
	TaskID string
	Wait   bool
}

// ListenResponse is the task's state: a child fetches its assignment, a parent
// polls for the result.
type ListenResponse struct {
	Status string         `json:"status"`
	Agent  string         `json:"agent"`
	Params map[string]any `json:"params"`
	Prompt string         `json:"prompt"`
	Output map[string]any `json:"output"`
}

// Listen returns the task's state, or NotFoundError if it is unknown.
func (q *Queries) Listen(ctx context.Context, req ListenRequest) (*ListenResponse, error) {
	task, ok := q.store.GetTask(req.TaskID)
	if !ok {
		return nil, &ports.NotFoundError{Kind: "task", ID: req.TaskID}
	}
	if req.Wait && task.Status == "dispatched" {
		task = q.waitForResult(ctx, req.TaskID, task)
	}
	return &ListenResponse{
		Status: task.Status,
		Agent:  task.Agent,
		Params: task.Params,
		Prompt: task.Prompt,
		Output: task.Output,
	}, nil
}

// waitForResult blocks until the task leaves the "dispatched" state, the
// long-poll timeout elapses, or ctx is cancelled, returning the latest state.
func (q *Queries) waitForResult(ctx context.Context, taskID string, task ports.Task) ports.Task {
	deadline := time.NewTimer(longPollTimeout)
	defer deadline.Stop()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return task
		case <-deadline.C:
			return task
		case <-ticker.C:
			if latest, ok := q.store.GetTask(taskID); ok {
				task = latest
				if task.Status != "dispatched" {
					return task
				}
			}
		}
	}
}
