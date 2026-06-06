package audit

import (
	"testing"

	"agent-protocol/internal/app/ports"
)

type fakeClock struct{ t int64 }

func (c fakeClock) Now() int64 { return c.t }

func TestLogStampsTimestamp(t *testing.T) {
	r := NewRing(fakeClock{t: 12345})
	r.Log(ports.Event{Event: "registered", Agent: "a"})

	got := r.Recent(0, "", "")
	if len(got) != 1 {
		t.Fatalf("want 1 event, got %d", len(got))
	}
	if got[0].TS != 12345 {
		t.Errorf("ts = %d, want 12345 (from clock)", got[0].TS)
	}
}

func TestRecentFilters(t *testing.T) {
	r := NewRing(fakeClock{})
	r.Log(ports.Event{Event: "dispatched", TaskID: "t1"})
	r.Log(ports.Event{Event: "completed", TaskID: "t1"})
	r.Log(ports.Event{Event: "dispatched", TaskID: "t2"})

	if got := r.Recent(0, "dispatched", ""); len(got) != 2 {
		t.Errorf("event filter: want 2, got %d", len(got))
	}
	if got := r.Recent(0, "", "t1"); len(got) != 2 {
		t.Errorf("taskID filter: want 2, got %d", len(got))
	}
	if got := r.Recent(0, "dispatched", "t2"); len(got) != 1 {
		t.Errorf("combined filter: want 1, got %d", len(got))
	}
}

func TestRecentLastLimit(t *testing.T) {
	r := NewRing(fakeClock{})
	for range 5 {
		r.Log(ports.Event{Event: "e", TaskID: "t"})
	}
	if got := r.Recent(3, "", ""); len(got) != 3 {
		t.Errorf("last=3: want 3, got %d", len(got))
	}
}

func TestRingCapsAtMax(t *testing.T) {
	r := NewRing(fakeClock{})
	for range maxEvents + 50 {
		r.Log(ports.Event{Event: "e"})
	}
	if got := r.Recent(maxEvents+100, "", ""); len(got) != maxEvents {
		t.Errorf("ring cap: want %d, got %d", maxEvents, len(got))
	}
}
