// Command dev is a smoke check: it wires the infra packages and prints a tick.
package main

import (
	"fmt"

	"agent-protocol/pkg/infra/config"
	"agent-protocol/pkg/infra/log"
)

func main() {
	cfg := config.Load()
	logger := log.New()
	logger.Info("dev smoke",
		"port", cfg.Port,
		"health_port", cfg.HealthPort,
		"proxy_url", cfg.ProxyURL,
		"spawn_workers", cfg.SpawnWorkers,
	)
	fmt.Println("✓ config + log OK")
}
