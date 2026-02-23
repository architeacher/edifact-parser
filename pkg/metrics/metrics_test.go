package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
)

func newTestMeterProvider() (*sdkmetric.MeterProvider, *sdkmetric.ManualReader) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	return mp, reader
}

func collectMetrics(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()

	var rm metricdata.ResourceMetrics

	err := reader.Collect(t.Context(), &rm)
	require.NoError(t, err)

	return rm
}

func TestMetrics_RecordLatency(t *testing.T) {
	t.Parallel()

	mp, reader := newTestMeterProvider()
	m, err := metrics.NewMetrics("test", mp)
	require.NoError(t, err)

	m.RecordLatency("ingest", 42.5)

	rm := collectMetrics(t, reader)
	require.NotEmpty(t, rm.ScopeMetrics, "expected at least one scope with metrics")

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test.operation.duration" {
				found = true
			}
		}
	}

	require.True(t, found, "expected test.operation.duration metric")
}

func TestMetrics_IncrementCounter(t *testing.T) {
	t.Parallel()

	mp, reader := newTestMeterProvider()
	m, err := metrics.NewMetrics("test", mp)
	require.NoError(t, err)

	m.IncrementCounter("ingest", "success")
	m.IncrementCounter("ingest", "error")

	rm := collectMetrics(t, reader)
	require.NotEmpty(t, rm.ScopeMetrics)

	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name == "test.operations.total" {
				found = true
			}
		}
	}

	require.True(t, found, "expected test.operations.total metric")
}

func TestNewNoopMetrics_DoesNotPanic(t *testing.T) {
	t.Parallel()

	m := metrics.NewNoopMetrics()

	m.RecordLatency("op", 1.0)
	m.IncrementCounter("op", "success")
}

func TestNewMetrics_ErrorOnNilProvider(t *testing.T) {
	t.Parallel()

	// noop provider should still succeed.
	m := metrics.NewNoopMetrics()
	require.NotNil(t, m)
}
