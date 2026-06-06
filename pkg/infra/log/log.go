// Package log provides a thin structured logger over the standard log/slog.
package log

import (
	"log/slog"
	"os"
)

// Logger wraps slog with an Info/Error key-value API. It structurally
// satisfies the ports.Logger interface introduced in later stages.
type Logger struct {
	l *slog.Logger
}

// New returns a Logger writing structured text logs to stderr.
func New() *Logger {
	return &Logger{l: slog.New(slog.NewTextHandler(os.Stderr, nil))}
}

// Info logs at INFO level with optional key-value pairs.
func (lg *Logger) Info(msg string, kv ...any) { lg.l.Info(msg, kv...) }

// Error logs at ERROR level with optional key-value pairs.
func (lg *Logger) Error(msg string, kv ...any) { lg.l.Error(msg, kv...) }
