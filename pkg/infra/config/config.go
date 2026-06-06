// Package config loads runtime configuration from the environment.
package config

import (
	"os"
	"strconv"
)

// Config holds runtime configuration for the agent-protocol proxy.
type Config struct {
	Port         string // MCP HTTP port (JSON-RPC endpoint /mcp)
	HealthPort   string // health endpoint port (/healthz)
	ProxyURL     string // URL spawned workers use to reach this proxy
	SpawnWorkers bool   // whether dispatch spawns a locked worker harness
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:         env("PORT", "4321"),
		HealthPort:   env("HEALTH_PORT", "8080"),
		ProxyURL:     env("PROXY_URL", "http://localhost:4321/mcp"),
		SpawnWorkers: envBool("SPAWN_WORKERS", false),
	}
}

// env returns the value of key, or def when unset/empty.
func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envBool parses key as a boolean, falling back to def on unset/invalid input.
func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
