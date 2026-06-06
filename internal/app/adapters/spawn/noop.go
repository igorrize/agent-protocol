// Package spawn launches locked worker harnesses. The real os/exec spawner
// lands in a later stage; Noop is the placeholder used until then.
package spawn

import "agent-protocol/internal/app/ports"

// Noop is a Spawner that does nothing.
type Noop struct{}

// Spawn satisfies ports.Spawner and performs no action.
func (Noop) Spawn(ports.Task, string, []string) error { return nil }
