package ports

// Logger is a minimal structured logger. pkg/infra/log.Logger satisfies it.
type Logger interface {
	Info(msg string, kv ...any)
	Error(msg string, kv ...any)
}
