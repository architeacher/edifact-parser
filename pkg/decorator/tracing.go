package decorator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type (
	commandTracingDecorator[C any, R any] struct {
		base   CommandHandler[C, R]
		tracer trace.Tracer
	}

	queryTracingDecorator[Q any, R any] struct {
		base   QueryHandler[Q, R]
		tracer trace.Tracer
	}
)

//nolint:dupl // Command and query tracing share structure but are distinct types.
func (d *commandTracingDecorator[C, R]) Handle(ctx context.Context, cmd C) (result R, err error) {
	actionName := strings.ToLower(generateActionName(cmd))

	ctx, span := d.tracer.Start(ctx, fmt.Sprintf("command.%s", actionName),
		trace.WithAttributes(attribute.String("command.name", actionName)),
	)
	defer span.End()

	start := time.Now()

	defer func() {
		span.SetAttributes(attribute.String("duration", time.Since(start).String()))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.AddEvent(fmt.Sprintf("command.%s.failure", actionName))

			return
		}

		span.SetStatus(codes.Ok, "")
		span.AddEvent(fmt.Sprintf("command.%s.success", actionName))
	}()

	return d.base.Handle(ctx, cmd)
}

func (d *queryTracingDecorator[Q, R]) Handle(ctx context.Context, query Q) (result R, err error) {
	actionName := strings.ToLower(generateActionName(query))

	ctx, span := d.tracer.Start(ctx, fmt.Sprintf("query.%s", actionName),
		trace.WithAttributes(attribute.String("query.name", actionName)),
	)
	defer span.End()

	start := time.Now()

	defer func() {
		span.SetAttributes(attribute.String("duration", time.Since(start).String()))

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			span.AddEvent(fmt.Sprintf("query.%s.failure", actionName))

			return
		}

		span.SetStatus(codes.Ok, "")
		span.AddEvent(fmt.Sprintf("query.%s.success", actionName))
	}()

	return d.base.Handle(ctx, query)
}
