// Package clock provides a system implementation of ports.Clock.
package clock

import "time"

// System is a Clock backed by the wall clock.
type System struct{}

// Now returns the current time in unix milliseconds.
func (System) Now() int64 { return time.Now().UnixMilli() }
