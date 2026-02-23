package decorator

import (
	"context"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel/trace"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

// CommandHandler processes a command and returns a result.
type (
	CommandHandler[C any, R any] interface {
		Handle(ctx context.Context, cmd C) (R, error)
	}

	// CommandHandlerFunc is a function adapter for CommandHandler.
	CommandHandlerFunc[C any, R any] func(ctx context.Context, cmd C) (R, error)
)

// Handle implements CommandHandler.
func (f CommandHandlerFunc[C, R]) Handle(ctx context.Context, cmd C) (R, error) {
	return f(ctx, cmd)
}

// ApplyCommandDecorators wraps a handler with tracing (innermost), metrics, and logging (outermost).
// Decorator order (outermost to innermost): Logging -> Metrics -> Tracing -> Handler.
func ApplyCommandDecorators[C any, R any](
	handler CommandHandler[C, R],
	log logger.Logger,
	metrics MetricsRecorder,
	tracer trace.Tracer,
) CommandHandler[C, R] {
	return &commandLoggingDecorator[C, R]{
		base: &commandMetricsDecorator[C, R]{
			base: &commandTracingDecorator[C, R]{
				base:   handler,
				tracer: tracer,
			},
			metrics: metrics,
		},
		logger: log,
	}
}

func generateActionName(handler any) string {
	return strings.Split(fmt.Sprintf("%T", handler), ".")[1]
}
