package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-protocol/internal/app/adapters/audit"
	"agent-protocol/internal/app/adapters/spawn"
	"agent-protocol/internal/app/adapters/store"
	"agent-protocol/internal/app/queries"
	"agent-protocol/internal/app/usecases"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type fakeClock struct{}

func (fakeClock) Now() int64 { return 0 }

func newTestServer() *Server {
	logger := noopLogger{}
	st := store.NewMemory(logger)
	au := audit.NewRing(fakeClock{})
	cmd := usecases.NewCommands(st, au, spawn.Noop{}, false, logger)
	qs := queries.NewQueries(st, au)
	return NewServer(st, au, cmd, qs, logger)
}

// call performs one JSON-RPC POST /mcp and returns the decoded response and the
// raw recorder (for status-code assertions).
func call(t *testing.T, s *Server, token, body string) (map[string]any, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	var out map[string]any
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatalf("decode response: %v\nbody: %s", err, rec.Body.String())
		}
	}
	return out, rec
}

// toolResult unwraps a tools/call response into the parsed inner result and the
// isError flag.
func toolResult(t *testing.T, resp map[string]any) (map[string]any, bool) {
	t.Helper()
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in response: %v", resp)
	}
	isError, _ := result["isError"].(bool)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)

	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		t.Fatalf("decode tool result text: %v\ntext: %s", err, text)
	}
	return parsed, isError
}

func registerResearcher(t *testing.T, s *Server) {
	t.Helper()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"register",`+
		`"arguments":{"agent_name":"researcher","input_schema":{"required":["ticket","repo"]},`+
		`"output_schema":{"required":["bug_file"]}}}}`)
	parsed, isErr := toolResult(t, resp)
	if isErr || parsed["status"] != "registered" || parsed["agent"] != "researcher" {
		t.Fatalf("register = %v (isError=%v)", parsed, isErr)
	}
}

// dispatchValid dispatches a valid task and returns the worker token.
func dispatchValid(t *testing.T, s *Server) string {
	t.Helper()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dispatch",`+
		`"arguments":{"agent_name":"researcher","params":{"ticket":"MED2-5322","repo":"broker"},`+
		`"prompt":"find the bug"}}}`)
	parsed, isErr := toolResult(t, resp)
	if isErr || parsed["status"] != "dispatched" {
		t.Fatalf("dispatch = %v (isError=%v)", parsed, isErr)
	}
	if parsed["task_id"] == "" || parsed["worker_token"] == "" {
		t.Fatalf("dispatch missing ids: %v", parsed)
	}
	return parsed["worker_token"].(string)
}

func TestInitialize(t *testing.T) {
	s := newTestServer()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`)
	result := resp["result"].(map[string]any)
	if result["protocolVersion"] != "2025-06-18" {
		t.Errorf("protocolVersion = %v", result["protocolVersion"])
	}
	info := result["serverInfo"].(map[string]any)
	if info["name"] != "agent-protocol" {
		t.Errorf("serverInfo = %v", info)
	}
}

func TestPing(t *testing.T) {
	s := newTestServer()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	result, ok := resp["result"].(map[string]any)
	if !ok || len(result) != 0 {
		t.Errorf("ping result = %v", resp["result"])
	}
}

func TestNotificationAcks(t *testing.T) {
	s := newTestServer()
	_, rec := call(t, s, "", `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	if rec.Code != http.StatusAccepted {
		t.Errorf("status = %d, want 202", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("notification body = %q, want empty", rec.Body.String())
	}
}

func TestUnknownMethod(t *testing.T) {
	s := newTestServer()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":9,"method":"does/not/exist"}`)
	errObj, ok := resp["error"].(map[string]any)
	if !ok || errObj["code"].(float64) != -32601 {
		t.Errorf("error = %v", resp["error"])
	}
}

func TestOrchestratorToolsList(t *testing.T) {
	s := newTestServer()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	tools := resp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 5 {
		t.Fatalf("orchestrator should see 5 tools, got %d", len(tools))
	}
}

func TestWorkerToolsListOnlyTwo(t *testing.T) {
	s := newTestServer()
	registerResearcher(t, s)
	token := dispatchValid(t, s)

	resp, _ := call(t, s, token, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	tools := resp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 2 {
		t.Fatalf("worker should see 2 tools, got %d", len(tools))
	}
	names := map[string]bool{}
	for _, tl := range tools {
		names[tl.(map[string]any)["name"].(string)] = true
	}
	if !names["listen"] || !names["complete"] {
		t.Errorf("worker tools = %v, want listen+complete", names)
	}
}

func TestWorkerDispatchBlocked(t *testing.T) {
	s := newTestServer()
	registerResearcher(t, s)
	token := dispatchValid(t, s)

	resp, _ := call(t, s, token, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"dispatch",`+
		`"arguments":{"agent_name":"x","params":{}}}}`)
	parsed, isErr := toolResult(t, resp)
	if !isErr {
		t.Error("worker dispatch should be isError=true")
	}
	errMsg, _ := parsed["error"].(string)
	if !strings.Contains(errMsg, "not available for role 'worker'") {
		t.Errorf("error = %q", errMsg)
	}
}

func TestDispatchRejected(t *testing.T) {
	s := newTestServer()
	registerResearcher(t, s)

	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"dispatch",`+
		`"arguments":{"agent_name":"researcher","params":{"ticket":"only-ticket"}}}}`)
	parsed, isErr := toolResult(t, resp)
	if isErr {
		t.Error("rejected (validation) should be isError=false")
	}
	if parsed["status"] != "rejected" {
		t.Fatalf("status = %v, want rejected", parsed["status"])
	}
	errs := parsed["errors"].(map[string]any)
	if errs["repo"] != "missing required key" {
		t.Errorf("errors = %v", errs)
	}
}

func TestDispatchPassThrough(t *testing.T) {
	s := newTestServer()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"dispatch",`+
		`"arguments":{"agent_name":"ghost","params":{"anything":1}}}}`)
	parsed, isErr := toolResult(t, resp)
	if isErr || parsed["status"] != "dispatched" {
		t.Errorf("pass-through = %v (isError=%v)", parsed, isErr)
	}
}

func TestCompleteAndListenFlow(t *testing.T) {
	s := newTestServer()
	registerResearcher(t, s)

	// dispatch to get a task id
	disp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dispatch",`+
		`"arguments":{"agent_name":"researcher","params":{"ticket":"X","repo":"Y"},"prompt":"go"}}}`)
	dparsed, _ := toolResult(t, disp)
	taskID := dparsed["task_id"].(string)

	// complete with valid output
	comp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"complete",`+
		`"arguments":{"task_id":"`+taskID+`","output":{"bug_file":"broker/handler.go"}}}}`)
	cparsed, isErr := toolResult(t, comp)
	if isErr || cparsed["status"] != "accepted" {
		t.Fatalf("complete = %v (isError=%v)", cparsed, isErr)
	}

	// listen returns completed + output
	lis, _ := call(t, s, "", `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"listen",`+
		`"arguments":{"task_id":"`+taskID+`"}}}`)
	lparsed, _ := toolResult(t, lis)
	if lparsed["status"] != "completed" {
		t.Errorf("listen status = %v, want completed", lparsed["status"])
	}
	out := lparsed["output"].(map[string]any)
	if out["bug_file"] != "broker/handler.go" {
		t.Errorf("listen output = %v", out)
	}
}

// TestWorkerListenDoesNotBlock guards the critical invariant: a worker listening
// for its own (still dispatched) assignment must get params immediately, never
// long-poll — otherwise it would deadlock waiting on the task it must run.
func TestWorkerListenDoesNotBlock(t *testing.T) {
	s := newTestServer()
	registerResearcher(t, s)

	disp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dispatch",`+
		`"arguments":{"agent_name":"researcher","params":{"ticket":"X","repo":"Y"},"prompt":"go"}}}`)
	dparsed, _ := toolResult(t, disp)
	taskID := dparsed["task_id"].(string)
	token := dparsed["worker_token"].(string)

	start := time.Now()
	resp, _ := call(t, s, token, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"listen",`+
		`"arguments":{"task_id":"`+taskID+`"}}}`)
	parsed, isErr := toolResult(t, resp)
	if isErr {
		t.Fatalf("worker listen errored: %v", parsed)
	}
	if parsed["status"] != "dispatched" {
		t.Errorf("status = %v, want dispatched (assignment)", parsed["status"])
	}
	if parsed["params"].(map[string]any)["ticket"] != "X" {
		t.Errorf("params not delivered: %v", parsed)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("worker listen must not block, took %v", elapsed)
	}
}

func TestListenUnknownTask(t *testing.T) {
	s := newTestServer()
	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"listen",`+
		`"arguments":{"task_id":"task_nope"}}}`)
	parsed, isErr := toolResult(t, resp)
	if !isErr {
		t.Error("unknown task should be isError=true")
	}
	if parsed["status"] != "error" || !strings.Contains(parsed["error"].(string), "unknown task") {
		t.Errorf("listen unknown = %v", parsed)
	}
}

func TestAuditShowsEvents(t *testing.T) {
	s := newTestServer()
	registerResearcher(t, s)
	dispatchValid(t, s)

	resp, _ := call(t, s, "", `{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"audit",`+
		`"arguments":{"last":10}}}`)
	parsed, _ := toolResult(t, resp)
	if parsed["status"] != "ok" {
		t.Fatalf("audit status = %v", parsed["status"])
	}
	events := parsed["events"].([]any)
	if len(events) < 2 {
		t.Fatalf("want >=2 events (registered, dispatched), got %d", len(events))
	}
	seen := map[string]bool{}
	for _, e := range events {
		seen[e.(map[string]any)["event"].(string)] = true
	}
	if !seen["registered"] || !seen["dispatched"] {
		t.Errorf("events seen = %v", seen)
	}
}
