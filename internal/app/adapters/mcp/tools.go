package mcp

import "agent-protocol/internal/app/ports"

// toolDef is an advertised MCP tool (catalogue entry for tools/list).
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// toolDefs is the full tool catalogue.
var toolDefs = []toolDef{
	{
		Name:        "register",
		Description: "Register an agent contract (input/output JSON Schema + allowed tools).",
		InputSchema: schemaObj([]string{"agent_name"}, map[string]any{
			"agent_name":    prop("string"),
			"input_schema":  prop("object"),
			"output_schema": prop("object"),
			"allowed_tools": prop("array"),
		}),
	},
	{
		Name:        "dispatch",
		Description: "Dispatch a task to an agent. Proxy validates params against the input contract.",
		InputSchema: schemaObj([]string{"agent_name", "params"}, map[string]any{
			"agent_name": prop("string"),
			"params":     prop("object"),
			"prompt":     prop("string"),
		}),
	},
	{
		Name:        "listen",
		Description: "Read a task's state: child fetches its assignment, parent polls for the result.",
		InputSchema: schemaObj([]string{"task_id"}, map[string]any{
			"task_id": prop("string"),
		}),
	},
	{
		Name:        "complete",
		Description: "Complete a task with output. Proxy validates output against the output contract.",
		InputSchema: schemaObj([]string{"task_id", "output"}, map[string]any{
			"task_id": prop("string"),
			"output":  prop("object"),
		}),
	},
	{
		Name:        "audit",
		Description: "Recent proxy events. Filters: last (N), event, task_id.",
		InputSchema: schemaObj(nil, map[string]any{
			"last":    prop("integer"),
			"event":   prop("string"),
			"task_id": prop("string"),
		}),
	},
}

// roleTools maps each role to the set of tools it may see and call.
var roleTools = map[ports.Role]map[string]bool{
	ports.RoleOrchestrator: {"register": true, "dispatch": true, "listen": true, "complete": true, "audit": true},
	ports.RoleWorker:       {"listen": true, "complete": true},
}

func roleAllows(role ports.Role, tool string) bool {
	return roleTools[role][tool]
}

// toolsForRole returns the tool catalogue filtered to the role's set.
func toolsForRole(role ports.Role) []toolDef {
	allowed := roleTools[role]
	out := make([]toolDef, 0, len(toolDefs))
	for _, td := range toolDefs {
		if allowed[td.Name] {
			out = append(out, td)
		}
	}
	return out
}

func schemaObj(required []string, properties map[string]any) map[string]any {
	m := map[string]any{"type": "object", "properties": properties}
	if len(required) > 0 {
		m["required"] = required
	}
	return m
}

func prop(t string) map[string]any { return map[string]any{"type": t} }
