// Package queries holds the read operations (listen/audit). It depends only on
// ports.
package queries

import "agent-protocol/internal/app/ports"

// Queries groups the read operations behind one facade.
type Queries struct {
	store ports.Store
	audit ports.AuditLog
}

// NewQueries wires the read queries.
func NewQueries(store ports.Store, audit ports.AuditLog) *Queries {
	return &Queries{store: store, audit: audit}
}
