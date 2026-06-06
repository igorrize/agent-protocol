package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-protocol/internal/app/adapters/audit"
	"agent-protocol/internal/app/adapters/store"
	"agent-protocol/internal/app/ports"
)

type noopLogger struct{}

func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

type fakeClock struct{}

func (fakeClock) Now() int64 { return 0 }

func newQueries() (*Queries, *store.Memory, *audit.Ring) {
	st := store.NewMemory(noopLogger{})
	au := audit.NewRing(fakeClock{})
	return NewQueries(st, au), st, au
}

func TestListenDispatched(t *testing.T) {
	q, st, _ := newQueries()
	task := st.CreateTask("researcher", map[string]any{"ticket": "X"}, "find bug")

	resp, err := q.Listen(context.Background(), ListenRequest{TaskID: task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "dispatched" || resp.Agent != "researcher" || resp.Prompt != "find bug" {
		t.Errorf("response = %+v", resp)
	}
	if resp.Params["ticket"] != "X" {
		t.Errorf("params = %v", resp.Params)
	}
}

func TestListenAfterComplete(t *testing.T) {
	q, st, _ := newQueries()
	task := st.CreateTask("researcher", nil, "go")
	st.CompleteTask(task.ID, map[string]any{"bug_file": "x.go"})

	resp, err := q.Listen(context.Background(), ListenRequest{TaskID: task.ID})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "completed" || resp.Output["bug_file"] != "x.go" {
		t.Errorf("response = %+v", resp)
	}
}

func TestListenUnknown(t *testing.T) {
	q, _, _ := newQueries()
	_, err := q.Listen(context.Background(), ListenRequest{TaskID: "task_nope"})
	var nf *ports.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("want NotFoundError, got %v", err)
	}
}

// TestListenWaitReturnsOnComplete: an orchestrator long-poll unblocks as soon as
// the task is completed by someone else.
func TestListenWaitReturnsOnComplete(t *testing.T) {
	q, st, _ := newQueries()
	task := st.CreateTask("researcher", nil, "go")
	go func() {
		time.Sleep(100 * time.Millisecond)
		st.CompleteTask(task.ID, map[string]any{"bug_file": "x.go"})
	}()

	resp, err := q.Listen(context.Background(), ListenRequest{TaskID: task.ID, Wait: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "completed" || resp.Output["bug_file"] != "x.go" {
		t.Errorf("response = %+v", resp)
	}
}

// TestListenWaitRespectsContext: a long-poll must return promptly when the
// context is cancelled rather than hanging for longPollTimeout.
func TestListenWaitRespectsContext(t *testing.T) {
	q, st, _ := newQueries()
	task := st.CreateTask("researcher", nil, "go") // stays dispatched

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	resp, err := q.Listen(ctx, ListenRequest{TaskID: task.ID, Wait: true})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "dispatched" {
		t.Errorf("status = %v, want dispatched", resp.Status)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("waited %v; context not respected", elapsed)
	}
}

// TestListenWorkerNoWait: with Wait=false the call returns the assignment
// immediately even while the task is still dispatched (the worker must not block
// on the task it is about to run).
func TestListenWorkerNoWait(t *testing.T) {
	q, st, _ := newQueries()
	task := st.CreateTask("researcher", map[string]any{"ticket": "X"}, "go")

	start := time.Now()
	resp, err := q.Listen(context.Background(), ListenRequest{TaskID: task.ID, Wait: false})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "dispatched" || resp.Params["ticket"] != "X" {
		t.Errorf("response = %+v", resp)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("Wait=false must return immediately, took %v", elapsed)
	}
}

func TestAudit(t *testing.T) {
	q, _, au := newQueries()
	au.Log(ports.Event{Event: "dispatched", TaskID: "t1"})
	au.Log(ports.Event{Event: "completed", TaskID: "t1"})
	au.Log(ports.Event{Event: "dispatched", TaskID: "t2"})

	resp, err := q.Audit(context.Background(), AuditRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != "ok" || len(resp.Events) != 3 {
		t.Errorf("response = %+v", resp)
	}

	filtered, _ := q.Audit(context.Background(), AuditRequest{TaskID: "t1"})
	if len(filtered.Events) != 2 {
		t.Errorf("taskID filter: want 2, got %d", len(filtered.Events))
	}
}
