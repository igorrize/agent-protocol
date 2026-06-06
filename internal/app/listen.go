package app

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-protocol/pkg/infra/config"
	"agent-protocol/pkg/infra/log"
)

// shutdownTimeout bounds graceful shutdown of the HTTP servers.
const shutdownTimeout = 10 * time.Second

// listenAndServe starts the MCP and health HTTP servers and blocks until an
// interrupt/terminate signal (or a fatal listen error), then gracefully shuts
// both servers down.
func listenAndServe(cfg config.Config, logger *log.Logger, mcpHandler http.Handler) error {
	mcpSrv := &http.Server{Addr: ":" + cfg.Port, Handler: mcpHandler}
	healthSrv := &http.Server{Addr: ":" + cfg.HealthPort, Handler: healthMux()}

	errc := make(chan error, 2)
	go serve(mcpSrv, "mcp", logger, errc)
	go serve(healthSrv, "health", logger, errc)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errc:
		return err
	case sig := <-stop:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	err := errors.Join(mcpSrv.Shutdown(ctx), healthSrv.Shutdown(ctx))
	logger.Info("agent-protocol stopped")
	return err
}

// serve runs one HTTP server, reporting only unexpected (non-shutdown) errors.
func serve(srv *http.Server, name string, logger *log.Logger, errc chan<- error) {
	logger.Info("listening", "server", name, "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		errc <- err
	}
}

// healthMux serves GET /healthz with a plain "ok" body.
func healthMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return mux
}
