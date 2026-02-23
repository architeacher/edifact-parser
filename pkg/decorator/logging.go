package decorator

import (
	"context"
	"time"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

type (
	commandLoggingDecorator[C any, R any] struct {
		base   CommandHandler[C, R]
		logger logger.Logger
	}

	queryLoggingDecorator[Q any, R any] struct {
		base   QueryHandler[Q, R]
		logger logger.Logger
	}
)

func (d *commandLoggingDecorator[C, R]) Handle(ctx context.Context, cmd C) (result R, err error) {
	start := time.Now()
	actionName := generateActionName(cmd)

	log := d.logger.WithContext(ctx)
	log.Info().Str("command", actionName).Msg("executing command")

	defer func() {
		duration := time.Since(start)

		if err != nil {
			log.Error().Err(err).
				Str("command", actionName).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("command failed")

			return
		}

		log.Info().
			Str("command", actionName).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("command completed")
	}()

	return d.base.Handle(ctx, cmd)
}

func (d *queryLoggingDecorator[Q, R]) Handle(ctx context.Context, query Q) (result R, err error) {
	start := time.Now()
	actionName := generateActionName(query)

	log := d.logger.WithContext(ctx)
	log.Debug().Str("query", actionName).Msg("executing query")

	defer func() {
		duration := time.Since(start)

		if err != nil {
			log.Error().Err(err).
				Str("query", actionName).
				Int64("duration_ms", duration.Milliseconds()).
				Msg("query failed")

			return
		}

		log.Debug().
			Str("query", actionName).
			Int64("duration_ms", duration.Milliseconds()).
			Msg("query completed")
	}()

	return d.base.Handle(ctx, query)
}
