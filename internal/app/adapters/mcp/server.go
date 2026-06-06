// Package mcp is the inbound MCP-over-HTTP (JSON-RPC) transport adapter.
package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"

	"agent-protocol/internal/app/ports"
	"agent-protocol/internal/app/queries"
	"agent-protocol/internal/app/usecases"
)

// Server is the MCP adapter. It implements http.Handler.
type Server struct {
	store    ports.Store
	audit    ports.AuditLog
	commands *usecases.Commands
	queries  *queries.Queries
	log      ports.Logger
}

// NewServer builds the MCP adapter.
func NewServer(store ports.Store, audit ports.AuditLog, commands *usecases.Commands, queries *queries.Queries, logger ports.Logger) *Server {
	return &Server{store: store, audit: audit, commands: commands, queries: queries, log: logger}
}

// ServeHTTP routes POST /mcp (JSON-RPC) and GET /healthz; everything else 404s.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/healthz":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	case r.Method == http.MethodPost && r.URL.Path == "/mcp":
		s.handleMCP(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
	}
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	role := s.roleOf(r)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, rpcError(nil, -32700, "read error"))
		return
	}
	var msg rpcRequest
	if err := json.Unmarshal(body, &msg); err != nil {
		writeJSON(w, http.StatusBadRequest, rpcError(nil, -32700, "parse error"))
		return
	}

	resp := s.handleRPC(r.Context(), role, msg)
	if resp == nil {
		w.WriteHeader(http.StatusAccepted) // notification: ack, no body
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

var bearerRE = regexp.MustCompile(`(?i)bearer\s+(\S+)`)

// roleOf resolves the role from the Bearer token; no/unknown token defaults to
// orchestrator (local default).
func (s *Server) roleOf(r *http.Request) ports.Role {
	m := bearerRE.FindStringSubmatch(r.Header.Get("Authorization"))
	if m == nil {
		return ports.RoleOrchestrator
	}
	if info, ok := s.store.TokenInfo(m[1]); ok {
		return info.Role
	}
	return ports.RoleOrchestrator
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
