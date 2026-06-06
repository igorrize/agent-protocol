# agent-protocol

> A typed-contract **MCP proxy** for AI agents — the single channel through which agents call each other, with **server-side argument validation** and **physically locked-down** worker harnesses.

AI agents talk to each other in plain text: no typed contracts, no input/output validation, no enforcement beyond "please follow the prompt". `agent-protocol` makes agent-to-agent calls **systemic, not prompt-based** — agents can only reach one another *through the proxy*, where every argument is checked against a contract, and workers are launched so locked-down they physically can't go anywhere else.

Single Go binary · MCP over HTTP · in-memory · no database.
<img width="1489" height="380" alt="Screenshot 2026-06-06 at 20 58 55" src="https://github.com/user-attachments/assets/bcf5e5c4-90a2-4e72-8753-770fe8ba3079" />
<img width="1475" height="190" alt="Screenshot 2026-06-06 at 20 59 08" src="https://github.com/user-attachments/assets/f1ec70e4-de52-4027-91e6-15e0040f1f37" />

---

## Why

| Plain agent-to-agent | With agent-protocol |
|---|---|
| "please send the right fields" (prompt) | arguments **validated** against a JSON-Schema contract, reject/accept (infrastructure) |
| a worker can call anything / anyone | worker is launched with a **single MCP server (the proxy)**, a **whitelist of tools**, no shell, no native sub-agent spawn — it *can't* bypass |
| no visibility | every `dispatched` / `rejected` / `completed` / `blocked` event is **audited** |

The guarantee lives in the **architecture**, not the language or the prompt: an agent can't call another agent directly — the only path is the proxy's `dispatch`, and that's where the contract is enforced.

---

## How it works

```
  parent (orchestrator)
        │  dispatch(agent, params, prompt)
        ▼
   ┌─────────────┐   validate params ✗ → rejected (parent fixes & retries)
   │    proxy    │   validate params ✓ → mint worker token + spawn LOCKED worker
   └─────────────┘
        │  (locked worker: ONLY this proxy, whitelisted tools, no shell/spawn)
        ▼
     worker ── listen() → gets assignment → does the work → complete(output)
        │
   ┌─────────────┐   validate output ✗ → rejected (worker retries)
   │    proxy    │   validate output ✓ → task completed
   └─────────────┘
        ▲
        │  parent listen()  (long-poll) → result
  parent (orchestrator)
```

---

## Install

```sh
brew install igorrize/tap/agent-protocol
```

## Run the proxy

```sh
agent-protocol serve                    # MCP on :4321 (/mcp), health on :8080 (/healthz)
SPAWN_WORKERS=1 agent-protocol serve     # also spawn a locked worker on each dispatch
```

| env | default | meaning |
|---|---|---|
| `PORT` | `4321` | MCP HTTP port (`/mcp`) |
| `HEALTH_PORT` | `8080` | health port (`/healthz`) |
| `PROXY_URL` | `http://localhost:4321/mcp` | URL spawned workers use to reach the proxy |
| `SPAWN_WORKERS` | `false` | whether `dispatch` spawns a locked worker harness |

## Use it

```sh
agent-protocol orchestrate               # launches a LOCKED Claude Code orchestrator
```
Then, in its chat:
```
Have the clj-reviewer agent review internal/app/schema/schema.go and show its findings.
```
The orchestrator can **only** `dispatch` → the proxy spawns a locked `clj-reviewer` worker (with just `Read/Glob/Grep`) → the worker reviews the file and `complete`s → the orchestrator gets the result via `listen`.
<img width="1286" height="255" alt="Screenshot 2026-06-06 at 21 00 34" src="https://github.com/user-attachments/assets/e8eb066e-fb40-4388-b8ff-838b65ffa1c5" />
<img width="1475" height="190" alt="Screenshot 2026-06-06 at 20 59 08" src="https://github.com/user-attachments/assets/1c1e1d78-5042-40d4-a367-4c666c4f53b1" />

<!-- screenshot: locked orchestrator delegating + result -->
<!-- ![live run](docs/live-run.png) -->

---

## The five MCP tools

| Tool | Role | What it does |
|---|---|---|
| `register` | orchestrator | register an agent contract (input/output JSON-Schema + `allowed_tools`) |
| `dispatch` | orchestrator | send a task; proxy **validates `params`**, mints a worker token, spawns a locked worker |
| `listen` | both | worker fetches its assignment; parent **long-polls** for the result |
| `complete` | worker | submit output; proxy **validates `output`** against the contract |
| `audit` | orchestrator | recent events (`dispatched` / `rejected` / `completed` / `blocked`) |

`tools/list` is **role-scoped**: a `worker` token sees only `listen` + `complete`; an `orchestrator` sees the rest.

---

## Rules = contracts

A "rule" is a `contract.json` placed next to an agent (under `examples/<agent>/`). It declares the agent's input/output JSON-Schema **and** the tools its locked worker is allowed. The proxy enforces it on **both ends** and auto-registers every `examples/*/contract.json` at startup.

**Example — `examples/clj-reviewer/contract.json`:**

```json
{
  "agent_name": "clj-reviewer",
  "allowed_tools": ["Read", "Glob", "Grep"],
  "input_schema": {
    "type": "object",
    "required": ["files", "rubric"],
    "properties": {
      "files":  { "type": "array"  },
      "rubric": { "type": "string" },
      "focus":  { "type": "string" }
    }
  },
  "output_schema": {
    "type": "object",
    "required": ["findings"],
    "properties": {
      "findings": { "type": "array"  },
      "summary":  { "type": "string" }
    }
  }
}
```

| Field | Meaning |
|---|---|
| `agent_name` | id used in `dispatch` |
| `allowed_tools` | exact built-in tools the locked worker may use (here `Read/Glob/Grep`). `listen`/`complete` are always added; `Bash`/`Agent`/etc. are **not** — they're physically removed |
| `input_schema` | validated on `dispatch(params)` — `required` keys + `type` |
| `output_schema` | validated on `complete(output)` |

**Validation in action** — the orchestrator dispatches without the required `rubric`, the proxy rejects it, and the orchestrator self-corrects on retry:

```jsonc
// dispatch without "rubric"
→ { "status": "rejected", "errors": { "rubric": "missing required key" } }
// dispatch with files + rubric
→ { "status": "dispatched", "task_id": "task_417d008e", "worker_token": "tok_…" }
```

No contract for an agent → **pass-through** (no validation), so the proxy never breaks an un-contracted flow.

<!-- screenshot: rejected → retry -->
<!-- ![rejected then retry](docs/rejected-retry.png) -->

---

## Locking (the guarantee)

Both orchestrator and workers are launched so they can **only** go through the proxy:

- **single MCP server** — the proxy is the *only* place to send a tool call;
- **tool WHITELIST, not a blocklist** — a blocklist always leaks (e.g. the `Monitor` tool can run a shell). The orchestrator gets **zero** built-ins + `dispatch/listen/audit`; a worker gets its contract's `allowed_tools` + `listen/complete`;
- **no native sub-agent spawn / no shell** → the only way to call another agent is the proxy's `dispatch`, where the contract is enforced.

`agent-protocol orchestrate` bakes the lock in — equivalent to launching Claude Code with:

```sh
claude --strict-mcp-config --mcp-config <locked.json> \
  --tools "" \
  --allowedTools "mcp__agent-protocol__dispatch,mcp__agent-protocol__listen,mcp__agent-protocol__audit"
```

The proxy spawns **workers** the same way (single MCP, `--tools "<contract allowed_tools>"`, `--allowedTools "mcp__agent-protocol__listen,complete"`).

<!-- screenshot: locked orchestrator can only delegate -->
<!-- ![locked orchestrator](docs/locked-orchestrator.png) -->

---

## Build from source

```sh
git clone git@github.com:igorrize/agent-protocol.git
cd agent-protocol
make run          # requires Go 1.26+ ; run `bash smoke.sh` to test the endpoint
```

---

## License

TBD.
