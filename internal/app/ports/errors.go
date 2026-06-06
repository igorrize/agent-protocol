package ports

import "fmt"

// Category classifies a domain error for transport (MCP) mapping. Its values
// are the MCP status strings the errmap adapter emits.
type Category string

const (
	CategoryRejected Category = "rejected" // schema validation failure
	CategoryBlocked  Category = "blocked"  // role/permission denied
	CategoryError    Category = "error"    // not found / other
)

// DomainError is an error that knows its transport category.
type DomainError interface {
	error
	Category() Category
}

// ValidationError reports schema validation failures keyed by field.
type ValidationError struct {
	Errors map[string]string
}

func (e *ValidationError) Error() string      { return fmt.Sprintf("validation failed: %v", e.Errors) }
func (e *ValidationError) Category() Category { return CategoryRejected }

// ForbiddenError reports that a role may not use a tool.
type ForbiddenError struct {
	Tool string
	Role Role
}

func (e *ForbiddenError) Error() string {
	return fmt.Sprintf("tool '%s' not available for role '%s'", e.Tool, e.Role)
}
func (e *ForbiddenError) Category() Category { return CategoryBlocked }

// NotFoundError reports a missing entity (e.g. an unknown task).
type NotFoundError struct {
	Kind string // e.g. "task"
	ID   string
}

func (e *NotFoundError) Error() string      { return fmt.Sprintf("unknown %s %s", e.Kind, e.ID) }
func (e *NotFoundError) Category() Category { return CategoryError }

// Compile-time checks that the error types satisfy DomainError.
var (
	_ DomainError = (*ValidationError)(nil)
	_ DomainError = (*ForbiddenError)(nil)
	_ DomainError = (*NotFoundError)(nil)
)
