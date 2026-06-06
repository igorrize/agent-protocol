#!/usr/bin/env bash
# Smoke test for the Go agent-protocol MCP endpoint (roles + audit).
# Server must be up on :4321 (make run). Run: bash smoke.sh
URL=http://localhost:4321/mcp
H='content-type: application/json'
call() { curl -sX POST "$URL" -H "$H" "$@"; }

echo "== register =="
call -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"register","arguments":{"agent_name":"researcher","input_schema":{"required":["ticket","repo"]},"output_schema":{"required":["bug_file"]}}}}'
echo; echo

echo "== dispatch (orchestrator) -> task_id + worker_token =="
resp=$(call -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dispatch","arguments":{"agent_name":"researcher","params":{"ticket":"MED2-5322","repo":"broker"},"prompt":"find the bug"}}}')
echo "$resp"
tok=$(echo "$resp" | grep -oE 'tok_[A-Za-z0-9-]+' | head -1)
echo ">> worker_token = $tok"
echo

echo "== worker tools/list (Bearer worker) -> ONLY listen+complete =="
call -H "Authorization: Bearer $tok" -d '{"jsonrpc":"2.0","id":3,"method":"tools/list"}'
echo; echo

echo "== worker tries dispatch (Bearer worker) -> DENIED =="
call -H "Authorization: Bearer $tok" -d '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"dispatch","arguments":{"agent_name":"x","params":{}}}}'
echo; echo

echo "== audit (last 10) =="
call -d '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"audit","arguments":{"last":10}}}'
echo
