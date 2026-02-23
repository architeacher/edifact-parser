package infrastructure

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
)

const serviceVersion = "1.0.0"

// NewResource creates a shared OTEL resource with service identity attributes.
// All three providers (traces, metrics, logs) reference the same resource
// so that every signal carries consistent service.name and service.version.
func NewResource(ctx context.Context, cfg config.OTelConfig) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(serviceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTel resource: %w", err)
	}

	return res, nil
}
