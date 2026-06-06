// Package audit provides an in-memory ring-buffer ports.AuditLog.
package audit

import (
	"sync"

	"agent-protocol/internal/app/ports"
)

const (
	maxEvents   = 1000 // ring capacity
	defaultLast = 20   // Recent default when last <= 0
)

// Ring is a bounded in-memory ring buffer of recent events (newest last).
type Ring struct {
	mu     sync.Mutex
	events []ports.Event
	clock  ports.Clock
}

// NewRing returns an empty audit ring buffer.
func NewRing(clock ports.Clock) *Ring {
	return &Ring{clock: clock}
}

// Log appends an event stamped with the current time, dropping the oldest
// events beyond maxEvents.
func (r *Ring) Log(e ports.Event) {
	e.TS = r.clock.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	if len(r.events) > maxEvents {
		r.events = r.events[len(r.events)-maxEvents:]
	}
}

// Recent returns up to last events (defaultLast when last <= 0), optionally
// filtered by event name and/or task id, newest last.
func (r *Ring) Recent(last int, event, taskID string) []ports.Event {
	if last <= 0 {
		last = defaultLast
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]ports.Event, 0, len(r.events))
	for _, e := range r.events {
		if event != "" && e.Event != event {
			continue
		}
		if taskID != "" && e.TaskID != taskID {
			continue
		}
		out = append(out, e)
	}
	if len(out) > last {
		out = out[len(out)-last:]
	}
	return out
}
