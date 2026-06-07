package spawn

import (
	"encoding/json"
	"os"
)

// WriteLockedConfig writes a locked MCP config to a temp file: the proxy is the
// ONLY server. An empty token omits the Authorization header (orchestrator
// role); a non-empty token authenticates as that worker. Returns the path.
func WriteLockedConfig(proxyURL, token string) (string, error) {
	server := map[string]any{"type": "http", "url": proxyURL}
	if token != "" {
		server["headers"] = map[string]any{"Authorization": "Bearer " + token}
	}
	cfg := map[string]any{"mcpServers": map[string]any{"agent-protocol": server}}

	data, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	f, err := os.CreateTemp("", "ap-mcp-*.json")
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	return f.Name(), nil
}
