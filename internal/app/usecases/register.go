package usecases

import (
	"context"

	"agent-protocol/internal/app/ports"
)

// RegisterRequest registers an agent contract.
type RegisterRequest struct {
	AgentName    string
	InputSchema  map[string]any
	OutputSchema map[string]any
	AllowedTools []string
}

// RegisterResponse is the reply for a successful register.
type RegisterResponse struct {
	Status string `json:"status"` // "registered"
	Agent  string `json:"agent"`
}

// Register stores an agent's input/output contract and allowed tools.
func (c *Commands) Register(_ context.Context, req RegisterRequest) (*RegisterResponse, error) {
	c.store.PutContract(ports.Contract{
		AgentName:    req.AgentName,
		Input:        req.InputSchema,
		Output:       req.OutputSchema,
		AllowedTools: req.AllowedTools,
	})
	c.audit.Log(ports.Event{Event: "registered", Tool: "register", Agent: req.AgentName})
	return &RegisterResponse{Status: "registered", Agent: req.AgentName}, nil
}
