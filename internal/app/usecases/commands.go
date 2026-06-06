// Package usecases holds the write commands (register/dispatch/complete). It
// depends only on ports and schema.
package usecases

import "agent-protocol/internal/app/ports"

// Commands groups the write usecases behind one facade.
type Commands struct {
	store        ports.Store
	audit        ports.AuditLog
	spawner      ports.Spawner
	spawnWorkers bool
	log          ports.Logger
}

// NewCommands wires the command usecases. spawnWorkers gates worker spawning
// (the SpawnWorkers config flag is passed as a primitive to keep this layer
// free of infra dependencies).
func NewCommands(store ports.Store, audit ports.AuditLog, spawner ports.Spawner, spawnWorkers bool, logger ports.Logger) *Commands {
	return &Commands{
		store:        store,
		audit:        audit,
		spawner:      spawner,
		spawnWorkers: spawnWorkers,
		log:          logger,
	}
}
