#!/usr/bin/env bash
# Poll a task's state/result. Run: bash listen.sh <task_id>
curl -sX POST http://localhost:4321/mcp -H 'content-type: application/json' \
  -d "{\"jsonrpc\":\"2.0\",\"id\":9,\"method\":\"tools/call\",\"params\":{\"name\":\"listen\",\"arguments\":{\"task_id\":\"$1\"}}}"
echo
