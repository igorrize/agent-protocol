# Agent Protocol — Go Port Plan (spec for the implementing agent)

> Это самодостаточная спецификация. Реализуй Go-сервис строго по ней.
> Идёшь маленькими шагами по §10, после каждого этапа прогоняй §11 quality gates.

## 0. Контекст и источники

- **Что переписываем:** рабочий MVP на Clojure → Go. Clojure — ЭТАЛОН поведения, бери логику оттуда 1:1.
  - Эталон: `/Users/igorlobazov/model_validation_project/src/agent_protocol/*.clj`
    (`core.clj` = MCP/JSON-RPC + роли; `validation.clj` = валидатор; `registry.clj` = стор/типы; `tools.clj` = usecases; `queries` listen/audit; `worker.clj` = спавн; `audit.clj` = ring buffer)
  - Smoke-скрипты эталона: `smoke.sh`, `b2.sh`, `listen.sh` (там же) — целевое поведение по HTTP.
- **Архитектурный гайд:** `/Users/igorlobazov/medi_drive/SERVICE_GUIDE.md` — следуем структуре/принципам **Pipeline-профиля**.

## 1. Что строим (домен в двух словах)

`agent-protocol` — **MCP-прокси**: единственная точка межагентных вызовов. Агенты (и оркестратор, и воркеры) ходят ТОЛЬКО через него. Прокси:
1. хранит **контракты** агентов (input/output JSON-Schema + список разрешённых тулов);
2. на `dispatch` валидирует **params** против input-схемы; на `complete` валидирует **output** против output-схемы; нет контракта → pass-through;
3. различает **роли** по токену: `orchestrator` (все тулы) vs `worker` (только `listen`+`complete`);
4. на `dispatch` создаёт задачу + worker-токен и (под флагом) **спавнит запертого воркера** — харнес (`claude` и др.) с урезанным конфигом, где единственный MCP = наш прокси и доступны только нужные тулы;
5. ведёт **audit** (ring buffer) всех событий.

5 MCP-тулов: `register`, `dispatch`, `listen`, `complete`, `audit`.

## 2. Профиль и жёсткие правила

**Профиль — Pipeline** (как эластик-индексатор: данные/вызовы текут, нет aggregate с мутациями). Слои: `adapters → usecases/queries → ports` (+ pure-пакет `schema/`). **Нет `domain/`-слоя** (нет aggregate.Base, Field-enum, change-tracking, domain-events, committer).

**DO (применяем из гайда):** структуру директорий §2, правило зависимостей §3, usecase `Execute(ctx, req)` §10, queries §11, composition root §15, pkg infra §16, entry points §17, graceful shutdown §18 (HTTP вместо gRPC), naming §22, unit-тесты §23, quality gates §24, error-pattern §13 (адаптированный — см. ниже).

**DON'T (это НЕ наш случай — НЕ делать):**
- ❌ gRPC / proto / `nemt-proto` — транспорт у нас **MCP over HTTP (JSON-RPC)**, только JSON, никаких `.proto`.
- ❌ Spanner / `nemt-objects` / facades / committer / mutations / outbox — persistence **in-memory**.
- ❌ `domain/` aggregate, `aggregate.Base`, Field-enum, change-tracking.
- ❌ multi-module §E — сервис маленький, **flat** (один «домен», ~5 usecases/queries).

**Error-pattern (адаптация §13):** ошибки — это типы, реализующие интерфейс, который знает свою категорию, но мапим НЕ в gRPC-коды, а в **MCP-семантику**: `ValidationError → status "rejected"`, `ForbiddenError(роль) → "blocked"`, `NotFound/прочее → "error"`. Прочие ошибки оборачиваем `fmt.Errorf("ctx: %w", err)`.

## 3. Структура директорий

```
agent-protocol/
├── go.mod                     # module agent-protocol (временно; namespace заменим при пуше в git)
├── Makefile  .air.toml  .gitignore
├── cmd/
│   ├── server/main.go         # import _ "go.uber.org/automaxprocs"; func main(){ app.Run() }
│   └── dev/main.go            # smoke: поднять pkg, напечатать ✓
├── internal/app/
│   ├── app.go                 # composition root (flat, §15): собрать store/audit/spawner → usecases/queries → mcp server → Listen
│   ├── listen.go              # MCP HTTP :4321 (/mcp) + health :8080 (/healthz) + graceful shutdown (§18)
│   ├── schema/                # PURE-валидация (transform пайплайна, не domain)
│   │   ├── schema.go
│   │   └── schema_test.go
│   ├── ports/
│   │   ├── types.go           # Contract, Task, Token, Role, Event (плоские структуры)
│   │   ├── store.go           # Store interface
│   │   ├── spawner.go         # Spawner interface
│   │   ├── audit.go           # AuditLog interface
│   │   ├── clock.go           # Clock interface (Now)
│   │   └── logger.go          # Logger interface
│   ├── usecases/
│   │   ├── commands.go        # Commands facade (New(...) + поля Register/Dispatch/Complete)
│   │   ├── register.go
│   │   ├── dispatch.go
│   │   └── complete.go
│   ├── queries/
│   │   ├── queries.go         # Queries facade
│   │   ├── listen.go
│   │   └── audit.go
│   └── adapters/
│       ├── mcp/               # INBOUND транспорт (вместо grpc)
│       │   ├── server.go      # http.Handler: маршрутизация /mcp, /healthz; чтение тела
│       │   ├── rpc.go         # JSON-RPC: initialize / ping / notifications/initialized / tools/list / tools/call
│       │   ├── tools.go       # каталог tool-defs + фильтр по роли
│       │   ├── route.go       # tools/call name → usecase/query
│       │   └── errmap.go      # DomainError → MCP {status:"rejected"/"blocked"/"error"}
│       ├── store/memory.go    # in-memory Store (map + sync.RWMutex)
│       ├── audit/ring.go      # ring buffer (max 1000)
│       └── spawn/
│           ├── spawner.go     # os/exec: записать locked .mcp.json, запустить харнес, лог в /tmp
│           └── harness/
│               ├── harness.go # Harness interface
│               ├── claude.go  # Claude Code (эталон) — флаги запирания
│               ├── gemini.go opencode.go codex.go   # заглушки на этап 8
└── pkg/infra/
    ├── config/config.go       # PORT (4321), HEALTH_PORT (8080), PROXY_URL, SPAWN_WORKERS (env)
    └── log/log.go             # простой логгер (stdlib log/slog)
```

## 4. go.mod / зависимости

`module agent-protocol`, Go 1.22+. Стандартная либа максимально. Внешние — минимум:
- `github.com/google/uuid` (id/токены)
- `go.uber.org/automaxprocs` (§17)
- JSON — `encoding/json` (stdlib). HTTP — `net/http` (stdlib). Спавн — `os/exec` (stdlib).

НЕ тянуть: spanner, grpc, proto, nemt-*.

## 5. Слои и правило зависимостей (строго)

```
adapters → usecases/queries → ports        (+ schema — pure, без зависимостей)
```
`ports` ничего из app не импортирует (только stdlib + uuid). `schema` — чистая, без зависимостей. `usecases/queries` зависят только от `ports` и `schema`. `adapters` реализуют `ports` и/или вызывают `usecases/queries`. Никаких импортов транспорта/стора в usecases.

## 6. Спецификация компонентов

### 6.1 `schema/schema.go` — валидатор (порт `validation.clj`)

```go
// Validate возвращает map ошибок {field: reason}; пустой map = валидно.
func Validate(schema map[string]any, data map[string]any) map[string]string
```
Поведение (точно как Clojure + ИСПРАВИТЬ 2 бага, найденные ревью):
- если `schema` пустая/nil → `{}` (pass-through);
- **required**: для каждого имени из `schema["required"]` ([]string), которого нет в `data` → `field: "missing required key"`;
- **type** (для полей из `schema["properties"]`, присутствующих в `data`): проверить тип значения против `properties[field]["type"]`:
  - `"string"`→string, `"number"`→любое число, `"integer"`→**целое, НО принимать `1.0` как integer** (JSON-числа часто float64; проверка: число И дробная часть == 0), `"boolean"`→bool, `"array"`→слайс, `"object"`→map, `"null"`→nil;
  - **`type` может быть массивом** (`["string","null"]`) → валидно, если значение подходит под ЛЮБОЙ из типов (any-of);
  - неизвестный/отсутствующий type → не валить;
  - несовпадение → `field: "expected <type>"`.
- тесты `schema_test.go` (как наши REPL-кейсы): missing repo → `{"repo":"missing required key"}`; всё на месте → `{}`; `{"ticket":42}` при type string → `{"ticket":"expected string"}`; `1.0` при integer → `{}`; union type `["string","null"]` со string и с nil → `{}`.

### 6.2 `ports/types.go`

```go
type Role string
const (RoleOrchestrator Role = "orchestrator"; RoleWorker Role = "worker")

type Contract struct {
    AgentName    string
    Input        map[string]any   // JSON-Schema
    Output       map[string]any
    AllowedTools []string         // рабочие тулы воркера (без listen/complete — их добавляет spawner)
}
type Task struct {
    ID, Agent string
    Params    map[string]any
    Prompt    string
    Status    string             // "dispatched" | "completed" | "rejected"? (см. usecases) | "timed_out"(later)
    Output    map[string]any
}
type Token struct { Value string; Role Role; TaskID string }
type Event struct {                // audit
    Event, Tool, Agent, TaskID string
    Role  Role
    Params, Output map[string]any
    Errors map[string]string
    TS    int64
}
```

### 6.3 `ports/*.go` — интерфейсы

```go
type Store interface {
    PutContract(c Contract)
    GetContract(agent string) (Contract, bool)
    CreateTask(agent string, params map[string]any, prompt string) Task   // генерит ID "task_<8hex>", status "dispatched"
    GetTask(id string) (Task, bool)
    CompleteTask(id string, output map[string]any) (Task, bool)           // status "completed"
    CreateToken(role Role, taskID string) string                          // "tok_<12hex>"
    TokenInfo(tok string) (Token, bool)
}
type Spawner interface { Spawn(task Task, workerToken string, allowedTools []string) error } // fire-and-forget
type AuditLog interface { Log(Event); Recent(last int, event, taskID string) []Event }       // ring 1000
type Clock interface { Now() int64 }                                                          // unix ms
type Logger interface { Info(msg string, kv ...any); Error(msg string, kv ...any) }
```

### 6.4 `usecases/` (commands; порт `tools.clj`)

- **Register(ctx, req{AgentName, InputSchema, OutputSchema, AllowedTools})** → `store.PutContract`; audit `registered`; reply `{Agent}`.
- **Dispatch(ctx, req{AgentName, Params, Prompt})**:
  1. `c, ok := store.GetContract`; если ok → `errs := schema.Validate(c.Input, Params)`; иначе errs пуст (pass-through);
  2. если `len(errs)>0` → audit `rejected` → вернуть `ValidationError{errs}`;
  3. иначе `task := store.CreateTask(...)`; `wtok := store.CreateToken(Worker, task.ID)`; audit `dispatched`;
  4. если `config.SpawnWorkers` → `spawner.Spawn(task, wtok, c.AllowedTools)` (под флагом, fire-and-forget, ошибку спавна только логировать);
  5. reply `{TaskID: task.ID, WorkerToken: wtok}`.
- **Complete(ctx, req{TaskID, Output})**:
  1. `task, ok := store.GetTask`; нет → `NotFoundError`;
  2. контракт агента задачи → `errs := schema.Validate(c.Output, Output)` (если контракт есть);
  3. errs>0 → audit `rejected` → `ValidationError`; иначе `store.CompleteTask` → audit `completed` → reply `{Accepted, TaskID}`.
- `commands.go` — facade `Commands{Register,Dispatch,Complete}` + `NewCommands(store, audit, spawner, clock, cfg)`.

### 6.5 `queries/` (reads; порт `tools.clj` listen/`audit.clj`)

- **Listen(ctx, req{TaskID})** → `store.GetTask`; нет → error; есть → `{Status, Agent, Params, Prompt, Output}`.
- **Audit(ctx, req{Last, Event, TaskID})** → `audit.Recent(...)` → `{Events}`.

### 6.6 `adapters/mcp/` (порт `core.clj`) — главный транспорт

- **server.go**: `http.Handler`. `POST /mcp`: определить роль (см. ниже), прочитать тело, распарсить JSON-RPC, вызвать `rpc.Handle(role, msg)`, ответить `application/json`. Notification (нет `id`) → `202`, пустое тело. `GET /healthz` → `200 "ok"`. Иначе `404`.
- **роль из заголовка**: `Authorization: Bearer <tok>` → `store.TokenInfo(tok)`; нет токена/неизвестен → `RoleOrchestrator` (локальный дефолт).
- **rpc.go** методы:
  - `initialize` → `{protocolVersion:"2025-06-18", capabilities:{tools:{}}, serverInfo:{name:"agent-protocol", version:"0.1.0"}}`
  - `notifications/initialized` → nil (→202)
  - `ping` → `{}`
  - `tools/list` → массив tool-defs, **отфильтрованный по роли**
  - `tools/call` → `{content:[{type:"text", text:<JSON результата>}], isError:<bool>}` — результат usecase/query сериализуется в JSON-строку в `text`; `isError=true` только для протокольных/`error` (не для `rejected`).
- **роли → тулы** (`tools.go`):
  - orchestrator: `{register, dispatch, listen, complete, audit}`
  - worker: `{listen, complete}`
  - на `tools/call`: если тул не в наборе роли → audit `blocked` → `{error: "tool '<t>' not available for role '<role>'"}` (isError=true).
- **tool-defs** (inputSchema каждого): `register{agent_name*, input_schema, output_schema, allowed_tools}`, `dispatch{agent_name*, params*, prompt}`, `listen{task_id*}`, `complete{task_id*, output*}`, `audit{last, event, task_id}`.
- **route.go**: name → вызов соответствующего usecase/query, маппинг arguments (string-keyed JSON) в req-структуры.
- **errmap.go**: `ValidationError→{status:"rejected", errors}`, `ForbiddenError→{error:...}`(isError), `NotFound→{status:"error", error}`. Успех usecase → его reply как `{status:"dispatched"/"accepted"/"registered"/...}`.

> ВАЖНО: JSON парсить со **string-keyed** map (`map[string]any`), не структуры, для `params`/`arguments` — ключи произвольные (как в Clojure: без keywordize).

### 6.7 `adapters/store/memory.go`
In-memory реализация `Store`: три `map` под `sync.RWMutex`. ID: `"task_"+hex8`, токен: `"tok_"+hex12` (uuid). Метод `LoadContracts(dir)` — прочитать все `<dir>/*/contract.json` (`{agent_name, allowed_tools, input_schema, output_schema}`) → `PutContract`. Логировать `loaded contract <name>`.

### 6.8 `adapters/audit/ring.go`
`AuditLog`: слайс под мьютексом, при `>1000` — отрезать старые. `Log` проставляет `TS` (clock). `Recent(last,event,taskID)` — фильтр по event/taskID, последние `last` (дефолт 20).

### 6.9 `adapters/spawn/` (порт `worker.clj`)
- **spawner.go**: `Spawn(task, workerToken, allowedTools)`:
  1. собрать **locked MCP-конфиг** (JSON): единственный сервер `agent-protocol` = `{type:"http", url: <PROXY_URL>, headers:{Authorization:"Bearer "+workerToken}}`; записать во временный файл;
  2. `harness.Command(configPath, prompt, allowedTools)` — взять у выбранного харнеса argv;
  3. `exec.Command(argv...)`, stdout+stderr → `/tmp/ap-worker-<task.ID>.log`, `cmd.Start()` (fire-and-forget, env наследуется);
  4. лог `spawned worker for <task.ID> tools <allowedTools>`.
  - **protocol prompt** воркеру: «Вызови `listen` с `{task_id:"<id>"}` → выполни (можешь использовать данные тулы на указанных файлах) → вызови `complete` с `{task_id, output}` по контракту».
- **harness/harness.go**: `type Harness interface { Command(configPath, prompt string, allowedTools []string) []string }`.
- **harness/claude.go** (эталон, делать первым):
  ```
  claude -p <prompt> --strict-mcp-config --mcp-config <configPath>
    --allowedTools "<join(allowedTools)>,mcp__agent-protocol__listen,mcp__agent-protocol__complete"
  ```
  (`listen`/`complete` добавляются всегда; `Task`/`Bash` НЕ в whitelist).

### 6.10 app.go / listen.go / pkg
- **app.go** `Run()`: создать config+logger → in-memory store (+`LoadContracts("examples")`) → audit ring → spawner (harness=claude по умолчанию) → `Commands`/`Queries` → `mcp.Server` → `Listen`.
- **listen.go**: HTTP-сервер на `:4321` (`/mcp`) и health на `:8080` (`/healthz`); `signal.Notify` SIGTERM/SIGINT → graceful shutdown.
- **pkg/infra/config**: env `PORT`(4321), `HEALTH_PORT`(8080), `PROXY_URL`(http://localhost:4321/mcp), `SPAWN_WORKERS`(bool).

## 7. Поведение MCP (резюме, по которому проверять)

`POST /mcp` JSON-RPC. Оркестратор без токена видит 5 тулов; worker-токен → только `listen`+`complete`, `dispatch` ему → `blocked`. `dispatch` валидного → `{status:"dispatched", task_id, worker_token}`; без обязательного поля → `{status:"rejected", errors:{...}}`; без контракта → pass-through dispatched. `complete` с кривым типом output → `rejected`. `audit` → события `registered/dispatched/rejected/completed/blocked` с `ts`.

## 8. Контракты агентов (examples + авто-регистрация)
Создать `examples/clj-reviewer/contract.json` и `examples/clj-test-writer/contract.json` (формат: `agent_name`, `allowed_tools`, `input_schema`, `output_schema`). `clj-reviewer`: `allowed_tools:["Read","Glob","Grep"]`, input `required:["files","rubric"]`. На старте `LoadContracts("examples")`.

## 9. Harness-адаптеры (флаги запирания)
Делать в порядке: **Claude Code (эталон)** → Gemini CLI → opencode → Codex (Oz позже, ему нужна сетевая тюрьма — вне этого плана). Флаги per-harness (whitelist тулов + единственный MCP + запрет нативного спавна + запрет shell) — единый паттерн, реализуется методом `Command(...)` каждого харнеса. (Точные флаги Claude см. §6.9; остальные адаптеры — заглушки на потом, интерфейс уже готов.)

## 10. Этапы (маленькими шагами, после каждого — §11)
1. Скелет: go.mod, cmd/{server,dev}, pkg/infra/{config,log}, `listen.go` (MCP HTTP + health), пустой `app.go`. Проверка: `curl /healthz` → ok.
2. `schema/` + тесты (порт `validate`, с фиксами integer/union).
3. `ports/` (types + интерфейсы) + `adapters/store/memory.go` + `adapters/audit/ring.go`.
4. `usecases/` + `queries/` (register/dispatch/complete/listen/audit) — без спавна.
5. `adapters/mcp/` (server/rpc/tools/route/errmap) + роли. Проверка: воспроизвести `smoke.sh` эталона (tools/list, register, dispatch valid/rejected/pass-through, worker tools/list=2, worker dispatch=blocked, audit).
6. `adapters/spawn/` + `harness/claude.go` + `examples/` + `LoadContracts`. Проверка: `SPAWN_WORKERS=1`, dispatch спавнит запертый `claude`, воркер делает `listen→complete`, `listen` отдаёт результат (как `b2.sh`).
7. Остальные harness-адаптеры (заглушки → реализация Gemini/opencode/Codex).

## 11. Quality gates (после каждого этапа)
```
gofmt -s -w .   &&  goimports -w .  &&  go vet ./...  &&  go test ./...  &&  go build ./...
```
(golangci-lint если установлен.) Naming §22 гайда: `XxxRequest`/`XxxResponse`, sentinel `Err<Reason>`, packages lower-case.

## 12. Acceptance (целевое поведение, как у Clojure-MVP)
- `bash`-smoke (аналог `smoke.sh`): register → dispatch valid (`dispatched`+task_id+worker_token) → worker `tools/list` (только listen+complete) → worker `dispatch` (`blocked`) → `audit` (события видны).
- live (аналог `b2.sh`): `SPAWN_WORKERS=1` сервер; запертый оркестратор (`claude --strict-mcp-config --mcp-config orchestrator.mcp.json --allowedTools "...dispatch,listen,audit"`) делает dispatch к `clj-reviewer`; первый раз без `rubric` → `rejected` → retry с `rubric` → спавн настоящего ревьюера (Read/Grep) → `complete` → `listen` отдаёт результат.

---
КОНЕЦ. Реализуй по этапам §10, поведение сверяй с Clojure-эталоном (§0) и §12.
