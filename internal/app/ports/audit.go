package ports

// AuditLog records proxy events in a bounded ring buffer.
type AuditLog interface {
	// Log appends an event, stamping it with the current time.
	Log(Event)
	// Recent returns up to last events (default applied when last <= 0),
	// optionally filtered by event name and/or task id, newest last.
	Recent(last int, event, taskID string) []Event
}
