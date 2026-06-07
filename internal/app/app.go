// Package app is the composition root: it wires infrastructure, adapters,
// usecases and queries together and runs the HTTP servers.
package app

import (
	"fmt"

	"go.uber.org/automaxprocs/maxprocs"

	"agent-protocol/internal/app/adapters/audit"
	"agent-protocol/internal/app/adapters/mcp"
	"agent-protocol/internal/app/adapters/spawn"
	"agent-protocol/internal/app/adapters/spawn/harness"
	"agent-protocol/internal/app/adapters/store"
	"agent-protocol/internal/app/queries"
	"agent-protocol/internal/app/usecases"
	"agent-protocol/pkg/infra/clock"
	"agent-protocol/pkg/infra/config"
	"agent-protocol/pkg/infra/log"
)

// Run wires the application together and serves until shutdown.
func Run() error {
	cfg := config.Load()
	logger := log.New()

	// Match GOMAXPROCS to the cgroup CPU quota (server path only), routing the
	// notice through our logger.
	if _, err := maxprocs.Set(maxprocs.Logger(func(format string, args ...any) {
		logger.Info(fmt.Sprintf(format, args...))
	})); err != nil {
		logger.Error("set maxprocs", "err", err)
	}

	logger.Info("agent-protocol starting",
		"port", cfg.Port,
		"health_port", cfg.HealthPort,
		"spawn_workers", cfg.SpawnWorkers,
	)

	st := store.NewMemory(logger)
	if err := st.LoadContracts("examples"); err != nil {
		logger.Error("load contracts", "err", err)
	}
	au := audit.NewRing(clock.System{})
	sp := spawn.New(harness.Claude{}, cfg.ProxyURL, logger)

	commands := usecases.NewCommands(st, au, sp, cfg.SpawnWorkers, logger)
	qs := queries.NewQueries(st, au)
	mcpServer := mcp.NewServer(st, au, commands, qs, logger)

	return listenAndServe(cfg, logger, mcpServer)
}
