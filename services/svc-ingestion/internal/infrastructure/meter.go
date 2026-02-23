package infrastructure

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
)

// NewMeterProvider creates a MeterProvider with two readers:
//   - OTLP periodic reader → pushes metrics to the Collector
//   - Prometheus exporter   → backs the /metrics scrape endpoint
//
// When cfg.MetricsEnabled is false a noop provider is registered.
func NewMeterProvider(ctx context.Context, cfg config.OTelConfig, res *resource.Resource) (*metric.MeterProvider, error) {
	if !cfg.Enabled || !cfg.MetricsEnabled {
		mp := metric.NewMeterProvider()
		otel.SetMeterProvider(mp)

		return mp, nil
	}

	otlpExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.Endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP metric exporter: %w", err)
	}

	promExporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("creating Prometheus metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(otlpExporter)),
		metric.WithReader(promExporter),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}
