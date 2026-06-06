package mcp

import (
	"context"
	"encoding/json"

	"agent-protocol/internal/app/ports"
)

const protocolVersion = "2025-06-18"

// rpcRequest is an incoming JSON-RPC message. A missing id marks a notification.
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

// handleRPC dispatches one JSON-RPC message to a response map, or nil for a
// notification (which the caller acks with 202 and no body).
func (s *Server) handleRPC(ctx context.Context, role ports.Role, msg rpcRequest) map[string]any {
	switch msg.Method {
	case "initialize":
		return rpcResult(msg.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "agent-protocol", "version": "0.1.0"},
		})
	case "notifications/initialized":
		return nil
	case "ping":
		return rpcResult(msg.ID, map[string]any{})
	case "tools/list":
		return rpcResult(msg.ID, map[string]any{"tools": toolsForRole(role)})
	case "tools/call":
		return s.handleToolsCall(ctx, role, msg)
	default:
		return rpcError(msg.ID, -32601, "Method not found: "+msg.Method)
	}
}

// handleToolsCall enforces the role's tool set, routes to the usecase/query,
// and wraps the result in the MCP content envelope.
func (s *Server) handleToolsCall(ctx context.Context, role ports.Role, msg rpcRequest) map[string]any {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	_ = json.Unmarshal(msg.Params, &params)

	var (
		result  any
		isError bool
	)
	if !roleAllows(role, params.Name) {
		s.audit.Log(ports.Event{Event: "blocked", Tool: params.Name, Role: role})
		result, isError = toResult(nil, &ports.ForbiddenError{Tool: params.Name, Role: role})
	} else {
		result, isError = s.route(ctx, role, params.Name, params.Arguments)
	}

	text, _ := json.Marshal(result)
	return rpcResult(msg.ID, map[string]any{
		"content": []map[string]any{{"type": "text", "text": string(text)}},
		"isError": isError,
	})
}

func rpcResult(id json.RawMessage, result any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": rawID(id), "result": result}
}

func rpcError(id json.RawMessage, code int, message string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      rawID(id),
		"error":   map[string]any{"code": code, "message": message},
	}
}

// rawID returns the raw id, or nil (JSON null) when absent.
func rawID(id json.RawMessage) any {
	if len(id) == 0 {
		return nil
	}
	return id
}
