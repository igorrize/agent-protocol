package mcp

import (
	"context"

	"agent-protocol/internal/app/ports"
	"agent-protocol/internal/app/queries"
	"agent-protocol/internal/app/usecases"
)

// route maps a tools/call name to the matching usecase/query, translating the
// string-keyed arguments into the request struct. It returns the result value
// (response struct or error map) and whether it is a transport error.
//
// The role decides listen's wait mode: an orchestrator long-polls for the
// result; a worker fetches its assignment immediately (see ListenRequest).
func (s *Server) route(ctx context.Context, role ports.Role, name string, args map[string]any) (any, bool) {
	switch name {
	case "register":
		resp, err := s.commands.Register(ctx, usecases.RegisterRequest{
			AgentName:    asString(args["agent_name"]),
			InputSchema:  asMap(args["input_schema"]),
			OutputSchema: asMap(args["output_schema"]),
			AllowedTools: asStringSlice(args["allowed_tools"]),
		})
		return toResult(resp, err)
	case "dispatch":
		resp, err := s.commands.Dispatch(ctx, usecases.DispatchRequest{
			AgentName: asString(args["agent_name"]),
			Params:    asMap(args["params"]),
			Prompt:    asString(args["prompt"]),
		})
		return toResult(resp, err)
	case "listen":
		resp, err := s.queries.Listen(ctx, queries.ListenRequest{
			TaskID: asString(args["task_id"]),
			Wait:   role == ports.RoleOrchestrator,
		})
		return toResult(resp, err)
	case "complete":
		resp, err := s.commands.Complete(ctx, usecases.CompleteRequest{
			TaskID: asString(args["task_id"]),
			Output: asMap(args["output"]),
		})
		return toResult(resp, err)
	case "audit":
		resp, err := s.queries.Audit(ctx, queries.AuditRequest{
			Last:   asInt(args["last"]),
			Event:  asString(args["event"]),
			TaskID: asString(args["task_id"]),
		})
		return toResult(resp, err)
	default:
		return map[string]any{"error": "unknown tool " + name}, true
	}
}

// toResult converts a usecase (response, error) pair into the value to serialize
// and an isError flag. Success serializes the response; an error is mapped to an
// MCP status map, with isError true only when the map carries an "error" key.
func toResult(resp any, err error) (any, bool) {
	if err == nil {
		return resp, false
	}
	m := errmap(err)
	_, hasError := m["error"]
	return m, hasError
}

// --- argument coercion (arguments arrive as a string-keyed JSON map) ---

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
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

func asInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	}
	return 0
}
