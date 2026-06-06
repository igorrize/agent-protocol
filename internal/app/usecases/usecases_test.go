package usecases

import (
	"context"
	"errors"
	"testing"

	"agent-protocol/internal/app/adapters/audit"
	"agent-protocol/internal/app/adapters/store"
	"agent-protocol/internal/app/ports"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type fakeClock struct{}

func (fakeClock) Now() int64 { return 0 }

type recordingSpawner struct {
	calls []spawnCall
}

type spawnCall struct {
	task    ports.Task
	token   string
	allowed []string
}

func (s *recordingSpawner) Spawn(task ports.Task, token string, allowed []string) error {
	s.calls = append(s.calls, spawnCall{task, token, allowed})
	return nil
}

type fixture struct {
	cmd     *Commands
	store   *store.Memory
	audit   *audit.Ring
	spawner *recordingSpawner
}

func newFixture(spawnWorkers bool) fixture {
	st := store.NewMemory(noopLogger{})
	au := audit.NewRing(fakeClock{})
	sp := &recordingSpawner{}
	return fixture{
		cmd:     NewCommands(st, au, sp, spawnWorkers, noopLogger{}),
		store:   st,
		audit:   au,
		spawner: sp,
	}
}

const researcher = "researcher"

// registerResearcher registers a contract requiring ticket+repo in / bug_file out.
func (f fixture) registerResearcher(t *testing.T) {
	t.Helper()
	_, err := f.cmd.Register(context.Background(), RegisterRequest{
		AgentName:    researcher,
		InputSchema:  map[string]any{"required": []any{"ticket", "repo"}},
		OutputSchema: map[string]any{"required": []any{"bug_file"}},
		AllowedTools: []string{"Read", "Grep"},
	})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
}

func TestRegister(t *testing.T) {
	f := newFixture(false)
	resp, err := f.cmd.Register(context.Background(), RegisterRequest{
		AgentName:   researcher,
		InputSchema: map[string]any{"required": []any{"ticket"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "registered" || resp.Agent != researcher {
		t.Errorf("response = %+v", resp)
	}
	if _, ok := f.store.GetContract(researcher); !ok {
		t.Error("contract not stored")
	}
	if ev := f.audit.Recent(0, "registered", ""); len(ev) != 1 {
		t.Errorf("want 1 registered event, got %d", len(ev))
	}
}

func TestDispatchValid(t *testing.T) {
	f := newFixture(false)
	f.registerResearcher(t)

	resp, err := f.cmd.Dispatch(context.Background(), DispatchRequest{
		AgentName: researcher,
		Params:    map[string]any{"ticket": "MED2-5322", "repo": "broker"},
		Prompt:    "find the bug",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "dispatched" || resp.TaskID == "" || resp.WorkerToken == "" {
		t.Errorf("response = %+v", resp)
	}
	if _, ok := f.store.GetTask(resp.TaskID); !ok {
		t.Error("task not created")
	}
	if info, ok := f.store.TokenInfo(resp.WorkerToken); !ok || info.Role != ports.RoleWorker {
		t.Errorf("worker token = %+v, %v", info, ok)
	}
	if len(f.spawner.calls) != 0 {
		t.Error("spawner should not be called when spawnWorkers=false")
	}
	if ev := f.audit.Recent(0, "dispatched", ""); len(ev) != 1 {
		t.Errorf("want 1 dispatched event, got %d", len(ev))
	}
}

func TestDispatchRejected(t *testing.T) {
	f := newFixture(false)
	f.registerResearcher(t)

	_, err := f.cmd.Dispatch(context.Background(), DispatchRequest{
		AgentName: researcher,
		Params:    map[string]any{"ticket": "MED2-5322"}, // missing repo
	})
	var ve *ports.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	if ve.Errors["repo"] != "missing required key" {
		t.Errorf("errors = %v", ve.Errors)
	}
	if ev := f.audit.Recent(0, "rejected", ""); len(ev) != 1 {
		t.Errorf("want 1 rejected event, got %d", len(ev))
	}
}

func TestDispatchPassThrough(t *testing.T) {
	f := newFixture(false)
	// no contract registered for "ghost"
	resp, err := f.cmd.Dispatch(context.Background(), DispatchRequest{
		AgentName: "ghost",
		Params:    map[string]any{"anything": 1},
	})
	if err != nil {
		t.Fatalf("pass-through should succeed, got %v", err)
	}
	if resp.Status != "dispatched" {
		t.Errorf("response = %+v", resp)
	}
}

func TestDispatchSpawns(t *testing.T) {
	f := newFixture(true)
	f.registerResearcher(t)

	resp, err := f.cmd.Dispatch(context.Background(), DispatchRequest{
		AgentName: researcher,
		Params:    map[string]any{"ticket": "X", "repo": "Y"},
		Prompt:    "go",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.spawner.calls) != 1 {
		t.Fatalf("want 1 spawn call, got %d", len(f.spawner.calls))
	}
	call := f.spawner.calls[0]
	if call.task.ID != resp.TaskID || call.token != resp.WorkerToken {
		t.Errorf("spawn call = %+v, resp = %+v", call, resp)
	}
	if len(call.allowed) != 2 || call.allowed[0] != "Read" {
		t.Errorf("spawn allowed tools = %v", call.allowed)
	}
}

func TestComplete(t *testing.T) {
	f := newFixture(false)
	f.registerResearcher(t)
	disp, _ := f.cmd.Dispatch(context.Background(), DispatchRequest{
		AgentName: researcher,
		Params:    map[string]any{"ticket": "X", "repo": "Y"},
	})

	resp, err := f.cmd.Complete(context.Background(), CompleteRequest{
		TaskID: disp.TaskID,
		Output: map[string]any{"bug_file": "broker/handler.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "accepted" || resp.TaskID != disp.TaskID {
		t.Errorf("response = %+v", resp)
	}
	task, _ := f.store.GetTask(disp.TaskID)
	if task.Status != "completed" || task.Output["bug_file"] != "broker/handler.go" {
		t.Errorf("task = %+v", task)
	}
	if ev := f.audit.Recent(0, "completed", ""); len(ev) != 1 {
		t.Errorf("want 1 completed event, got %d", len(ev))
	}
}

func TestCompleteUnknownTask(t *testing.T) {
	f := newFixture(false)
	_, err := f.cmd.Complete(context.Background(), CompleteRequest{TaskID: "task_nope"})
	var nf *ports.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want NotFoundError, got %v", err)
	}
}

func TestCompleteRejected(t *testing.T) {
	f := newFixture(false)
	f.registerResearcher(t)
	disp, _ := f.cmd.Dispatch(context.Background(), DispatchRequest{
		AgentName: researcher,
		Params:    map[string]any{"ticket": "X", "repo": "Y"},
	})

	_, err := f.cmd.Complete(context.Background(), CompleteRequest{
		TaskID: disp.TaskID,
		Output: map[string]any{}, // missing bug_file
	})
	var ve *ports.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
	if ve.Errors["bug_file"] != "missing required key" {
		t.Errorf("errors = %v", ve.Errors)
	}
	// task must remain not completed
	if task, _ := f.store.GetTask(disp.TaskID); task.Status != "dispatched" {
		t.Errorf("task should stay dispatched, got %q", task.Status)
	}
}
