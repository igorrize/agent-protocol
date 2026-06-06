package mcp

import (
	"errors"

	"agent-protocol/internal/app/ports"
)

// errmap maps a domain error to its MCP result map:
//
//	ValidationError -> {status:"rejected", errors}
//	ForbiddenError  -> {error}                       (isError via the "error" key)
//	NotFoundError   -> {status:"error", error}
//	other           -> {status:"error", error}
func errmap(err error) map[string]any {
	var ve *ports.ValidationError
	if errors.As(err, &ve) {
		return map[string]any{"status": string(ports.CategoryRejected), "errors": ve.Errors}
	}
	var fe *ports.ForbiddenError
	if errors.As(err, &fe) {
		return map[string]any{"error": fe.Error()}
	}
	var nf *ports.NotFoundError
	if errors.As(err, &nf) {
		return map[string]any{"status": string(ports.CategoryError), "error": nf.Error()}
	}
	return map[string]any{"status": string(ports.CategoryError), "error": err.Error()}
}
