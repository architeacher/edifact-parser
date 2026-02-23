package decorator

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MetricsRecorder abstracts the metrics operations required by decorators.
// This enables dependency inversion — the decorator package does not depend on a concrete metrics implementation.
type (
	MetricsRecorder interface {
		RecordLatency(operation string, durationMs float64)
		IncrementCounter(operation, status string)
	}
)

type (
	commandMetricsDecorator[C any, R any] struct {
		base    CommandHandler[C, R]
		metrics MetricsRecorder
	}

	queryMetricsDecorator[Q any, R any] struct {
		base    QueryHandler[Q, R]
		metrics MetricsRecorder
	}
)

func (d *commandMetricsDecorator[C, R]) Handle(ctx context.Context, cmd C) (result R, err error) {
	start := time.Now()

	defer func() {
		if d.metrics == nil {
			return
		}

		actionName := strings.ToLower(generateActionName(cmd))
		durationMs := float64(time.Since(start).Milliseconds())

		d.metrics.RecordLatency(fmt.Sprintf("commands.%s", actionName), durationMs)

		if err == nil {
			d.metrics.IncrementCounter(fmt.Sprintf("commands.%s", actionName), "success")
		} else {
			d.metrics.IncrementCounter(fmt.Sprintf("commands.%s", actionName), "error")
		}
	}()

	return d.base.Handle(ctx, cmd)
}

func (d *queryMetricsDecorator[Q, R]) Handle(ctx context.Context, query Q) (result R, err error) {
	start := time.Now()

	defer func() {
		if d.metrics == nil {
			return
		}

		actionName := strings.ToLower(generateActionName(query))
		durationMs := float64(time.Since(start).Milliseconds())

		d.metrics.RecordLatency(fmt.Sprintf("queries.%s", actionName), durationMs)

		if err == nil {
			d.metrics.IncrementCounter(fmt.Sprintf("queries.%s", actionName), "success")
		} else {
			d.metrics.IncrementCounter(fmt.Sprintf("queries.%s", actionName), "error")
		}
	}()

	return d.base.Handle(ctx, query)
}
