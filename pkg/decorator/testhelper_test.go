package decorator_test

import (
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

type testCommand struct {
	Value string
}

type testResult struct {
	Name string
}

type testQuery struct {
	ID string
}

type testQueryResult struct {
	Name string
}

// noopMetrics satisfies decorator.MetricsRecorder for tests.
type noopMetrics struct{}

func (n *noopMetrics) RecordLatency(_ string, _ float64) {}
func (n *noopMetrics) IncrementCounter(_, _ string)      {}

func noopTracer() trace.Tracer {
	return noop.NewTracerProvider().Tracer("test")
}
