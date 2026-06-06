package queries

import (
	"context"

	"agent-protocol/internal/app/ports"
)

// AuditRequest reads recent proxy events with optional filters.
type AuditRequest struct {
	Last   int
	Event  string
	TaskID string
}

// AuditResponse carries the matching events.
type AuditResponse struct {
	Status string        `json:"status"` // "ok"
	Events []ports.Event `json:"events"`
}

// Audit returns recent proxy events filtered by the request.
func (q *Queries) Audit(_ context.Context, req AuditRequest) (*AuditResponse, error) {
	events := q.audit.Recent(req.Last, req.Event, req.TaskID)
	return &AuditResponse{Status: "ok", Events: events}, nil
}
