package metrics

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/noop"
)

// Metrics wraps OTEL histogram and counter instruments for application observability.
type (
	Metrics struct {
		latency metric.Float64Histogram
		counter metric.Int64Counter
	}
)

// NewMetrics creates a Metrics instance backed by the given MeterProvider.
// The namespace is used as the OTEL meter (instrument scope) name.
func NewMetrics(namespace string, provider metric.MeterProvider) (*Metrics, error) {
	meter := provider.Meter(namespace)

	latency, err := meter.Float64Histogram(
		fmt.Sprintf("%s.operation.duration", namespace),
		metric.WithDescription("Duration of operations in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, fmt.Errorf("creating latency histogram: %w", err)
	}

	counter, err := meter.Int64Counter(
		fmt.Sprintf("%s.operations.total", namespace),
		metric.WithDescription("Total number of operations by status."),
	)
	if err != nil {
		return nil, fmt.Errorf("creating operations counter: %w", err)
	}

	return &Metrics{
		latency: latency,
		counter: counter,
	}, nil
}

// RecordLatency observes the duration for the given operation.
func (m *Metrics) RecordLatency(operation string, durationMs float64) {
	m.latency.Record(context.Background(),
		durationMs,
		metric.WithAttributes(attribute.String("operation", operation)),
	)
}

// IncrementCounter increments the counter for the given operation and status.
func (m *Metrics) IncrementCounter(operation, status string) {
	m.counter.Add(context.Background(), 1,
		metric.WithAttributes(
			attribute.String("operation", operation),
			attribute.String("status", status),
		),
	)
}

// NewNoopMetrics returns a Metrics backed by noop instruments.
// Suitable for unit tests where metric collection is not needed.
func NewNoopMetrics() *Metrics {
	provider := noop.NewMeterProvider()

	m, _ := NewMetrics("noop", provider)

	return m
}
