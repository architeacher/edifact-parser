package decorator

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

// QueryHandler processes a query and returns a result.
type (
	QueryHandler[Q any, R any] interface {
		Handle(ctx context.Context, query Q) (R, error)
	}

	// QueryHandlerFunc is a function adapter for QueryHandler.
	QueryHandlerFunc[Q any, R any] func(ctx context.Context, query Q) (R, error)
)

// Handle implements QueryHandler.
func (f QueryHandlerFunc[Q, R]) Handle(ctx context.Context, query Q) (R, error) {
	return f(ctx, query)
}

// ApplyQueryDecorators wraps a handler with tracing (innermost), metrics, and logging (outermost).
// Decorator order (outermost to innermost): Logging -> Metrics -> Tracing -> Handler.
func ApplyQueryDecorators[Q any, R any](
	handler QueryHandler[Q, R],
	log logger.Logger,
	metrics MetricsRecorder,
	tracer trace.Tracer,
) QueryHandler[Q, R] {
	return &queryLoggingDecorator[Q, R]{
		base: &queryMetricsDecorator[Q, R]{
			base: &queryTracingDecorator[Q, R]{
				base:   handler,
				tracer: tracer,
			},
			metrics: metrics,
		},
		logger: log,
	}
}
