package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	// shutdownTimeout is the maximum duration allowed for graceful shutdown before forcing exit.
	shutdownTimeout = 10 * time.Second
)

// ServiceCtx holds the service lifecycle state, including dependencies and shutdown coordination.
type ServiceCtx struct {
	deps            *dependencies
	shutdownChannel chan os.Signal
	serverCtx       context.Context
	serverStopFunc  context.CancelFunc
	serverReady     chan struct{}
}

// New creates a ServiceCtx with the given options applied.
func New(opts ...ServiceOption) *ServiceCtx {
	ctx := &ServiceCtx{
		shutdownChannel: make(chan os.Signal, 1),
	}

	for _, opt := range opts {
		opt(ctx)
	}

	return ctx
}

// Run builds dependencies, starts the HTTP server, and blocks until a shutdown signal is received.
func (c *ServiceCtx) Run() {
	if err := c.build(); err != nil {
		log.Fatalf("failed to build service: %v", err)
	}

	c.startService()
	c.shutdownHook()

	select {
	case <-c.serverCtx.Done():
	case <-c.shutdownChannel:
		defer close(c.shutdownChannel)
	}

	c.shutdown()
}

func (c *ServiceCtx) build(opts ...DependencyOption) error {
	c.serverCtx, c.serverStopFunc = context.WithCancel(context.Background())

	var err error

	c.deps, err = initializeDependencies(c.serverCtx, opts...)
	if err != nil {
		return fmt.Errorf("initializing dependencies: %w", err)
	}

	return nil
}

// Build initializes all dependencies without starting the server.
func (c *ServiceCtx) Build(opts ...DependencyOption) error {
	return c.build(opts...)
}

func (c *ServiceCtx) startService() {
	addr := fmt.Sprintf(":%d", c.deps.config.Server.Port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen on %s: %v", addr, err)
	}

	go func() {
		c.deps.infra.logger.Info().Str("addr", listener.Addr().String()).Msg("starting HTTP server")

		if c.serverReady != nil {
			close(c.serverReady)
		}

		if err := c.deps.infra.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			c.deps.infra.logger.Error().Err(err).Msg("HTTP server failed")
			c.serverStopFunc()
		}
	}()
}

func (c *ServiceCtx) shutdownHook() {
	signal.Notify(c.shutdownChannel, syscall.SIGINT, syscall.SIGTERM)
}

func (c *ServiceCtx) shutdown() {
	c.deps.infra.logger.Info().Msg("shutting down service...")

	c.serverStopFunc()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)

	go func() {
		<-shutdownCtx.Done()

		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			c.deps.infra.logger.Error().Msg("graceful shutdown timed out.. forcing exit.")
			cancel()
			os.Exit(1)
		}
	}()

	c.cleanup(shutdownCtx)
	cancel()

	c.deps.infra.logger.Info().Msg("service shutdown complete")
}

// WaitForServer blocks until the HTTP server signals it is ready to accept connections.
func (c *ServiceCtx) WaitForServer() {
	if c.serverReady != nil {
		<-c.serverReady
	}
}

func (c *ServiceCtx) cleanup(shutdownCtx context.Context) {
	c.deps.infra.logger.Info().Msg("cleaning up resources...")

	// Iterate in reverse (LIFO) so the last-registered resource (e.g., HTTP server) shuts down first,
	// before its dependencies (database, Kafka) are closed.
	for idx := len(c.deps.cleanupFuncs) - 1; idx >= 0; idx-- {
		entry := c.deps.cleanupFuncs[idx]
		if err := entry.fn(shutdownCtx); err != nil {
			c.deps.infra.logger.Error().Err(err).Str("resource", entry.name).Msg("failed to shutdown resource gracefully")
		}
	}

	c.deps.infra.logger.Info().Msg("cleanup completed")
}

// HTTPHandler returns the HTTP handler for testing purposes.
func (c *ServiceCtx) HTTPHandler() http.Handler {
	return c.deps.infra.httpServer.Handler
}
