package infrastructure

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
)

// NewLoggerProvider creates a LoggerProvider that exports log records
// to the OTEL Collector via OTLP gRPC.
// When cfg.LogsEnabled is false a noop provider is returned.
func NewLoggerProvider(ctx context.Context, cfg config.OTelConfig, res *resource.Resource) (*log.LoggerProvider, error) {
	if !cfg.Enabled || !cfg.LogsEnabled {
		lp := log.NewLoggerProvider()
		global.SetLoggerProvider(lp)

		return lp, nil
	}

	exporter, err := otlploggrpc.New(ctx,
		otlploggrpc.WithEndpoint(cfg.Endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP log exporter: %w", err)
	}

	lp := log.NewLoggerProvider(
		log.WithResource(res),
		log.WithProcessor(log.NewBatchProcessor(exporter)),
	)

	global.SetLoggerProvider(lp)

	return lp, nil
}
