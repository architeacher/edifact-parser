package usecases

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/architeacher/nomos-technical-challenge/pkg/decorator"
	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
	"github.com/architeacher/nomos-technical-challenge/pkg/metrics"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/commands"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/usecases/queries"
)

func TestNewApplication(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	m := metrics.NewNoopMetrics()
	tracer := noop.NewTracerProvider().Tracer("test")

	deps := newTestDeps(log, m, tracer)
	app := NewApplication(deps)

	require.NotNil(t, app)
	require.NotNil(t, app.Commands.IngestFile)
	require.NotNil(t, app.Queries.GetInterchange)
	require.NotNil(t, app.Queries.GetRawFile)
	require.NotNil(t, app.Queries.FetchHealth)
	require.NotNil(t, app.Queries.FetchLiveness)
	require.NotNil(t, app.Queries.FetchReadiness)
	require.NotNil(t, app.Queries.ListMessages)
}

func TestApplication_CommandDecoratorChain(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	m := metrics.NewNoopMetrics()
	tracer := noop.NewTracerProvider().Tracer("test")

	cases := []struct {
		name    string
		handler decorator.CommandHandlerFunc[commands.IngestFileCommand, *commands.IngestFileResult]
		wantErr bool
		wantMsg int
	}{
		{
			name: "success flows through all decorators",
			handler: func(_ context.Context, _ commands.IngestFileCommand) (*commands.IngestFileResult, error) {
				return &commands.IngestFileResult{MessageCount: 3}, nil
			},
			wantMsg: 3,
		},
		{
			name: "error propagates through all decorators",
			handler: func(_ context.Context, _ commands.IngestFileCommand) (*commands.IngestFileResult, error) {
				return nil, errors.New("ingest failed")
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(log, m, tracer)
			deps.IngestFile = tc.handler

			app := NewApplication(deps)

			result, err := app.Commands.IngestFile.Handle(context.Background(), commands.IngestFileCommand{
				FileData:    []byte("test"),
				ContentType: "application/edifact",
			})

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantMsg, result.MessageCount)
		})
	}
}

func TestApplication_QueryDecoratorChain(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	m := metrics.NewNoopMetrics()
	tracer := noop.NewTracerProvider().Tracer("test")

	cases := []struct {
		name    string
		handler decorator.QueryHandlerFunc[queries.GetInterchangeQuery, queries.GetInterchangeResult]
		wantErr bool
		wantID  string
	}{
		{
			name: "success flows through all decorators",
			handler: func(_ context.Context, _ queries.GetInterchangeQuery) (queries.GetInterchangeResult, error) {
				return queries.GetInterchangeResult{SenderID: "SENDER001"}, nil
			},
			wantID: "SENDER001",
		},
		{
			name: "error propagates through all decorators",
			handler: func(_ context.Context, _ queries.GetInterchangeQuery) (queries.GetInterchangeResult, error) {
				return queries.GetInterchangeResult{}, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(log, m, tracer)
			deps.GetInterchange = tc.handler

			app := NewApplication(deps)

			result, err := app.Queries.GetInterchange.Handle(context.Background(), queries.GetInterchangeQuery{})

			if tc.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantID, result.SenderID)
		})
	}
}

func TestApplication_ContextPropagation(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	m := metrics.NewNoopMetrics()
	tracer := noop.NewTracerProvider().Tracer("test")

	type ctxKey string

	const testKey ctxKey = "test_key"

	var capturedValue string

	deps := newTestDeps(log, m, tracer)
	deps.GetInterchange = decorator.QueryHandlerFunc[queries.GetInterchangeQuery, queries.GetInterchangeResult](
		func(ctx context.Context, _ queries.GetInterchangeQuery) (queries.GetInterchangeResult, error) {
			if v, ok := ctx.Value(testKey).(string); ok {
				capturedValue = v
			}

			return queries.GetInterchangeResult{}, nil
		},
	)

	app := NewApplication(deps)

	ctx := context.WithValue(context.Background(), testKey, "propagated")
	_, err := app.Queries.GetInterchange.Handle(ctx, queries.GetInterchangeQuery{})

	require.NoError(t, err)
	require.Equal(t, "propagated", capturedValue)
}

func TestApplication_AllQueriesCallable(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	m := metrics.NewNoopMetrics()
	tracer := noop.NewTracerProvider().Tracer("test")

	deps := newTestDeps(log, m, tracer)
	app := NewApplication(deps)

	ctx := context.Background()

	t.Run("GetInterchange", func(t *testing.T) {
		t.Parallel()

		_, err := app.Queries.GetInterchange.Handle(ctx, queries.GetInterchangeQuery{})
		require.NoError(t, err)
	})

	t.Run("GetRawFile", func(t *testing.T) {
		t.Parallel()

		_, err := app.Queries.GetRawFile.Handle(ctx, queries.GetRawFileQuery{})
		require.NoError(t, err)
	})

	t.Run("FetchHealth", func(t *testing.T) {
		t.Parallel()

		_, err := app.Queries.FetchHealth.Handle(ctx, queries.FetchHealthQuery{})
		require.NoError(t, err)
	})

	t.Run("FetchLiveness", func(t *testing.T) {
		t.Parallel()

		_, err := app.Queries.FetchLiveness.Handle(ctx, queries.FetchLivenessQuery{})
		require.NoError(t, err)
	})

	t.Run("FetchReadiness", func(t *testing.T) {
		t.Parallel()

		_, err := app.Queries.FetchReadiness.Handle(ctx, queries.FetchReadinessQuery{})
		require.NoError(t, err)
	})

	t.Run("ListMessages", func(t *testing.T) {
		t.Parallel()

		_, err := app.Queries.ListMessages.Handle(ctx, queries.ListMessagesQuery{})
		require.NoError(t, err)
	})
}

func TestApplication_NilHandlerPanics(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	m := metrics.NewNoopMetrics()
	tracer := noop.NewTracerProvider().Tracer("test")

	cases := []struct {
		name    string
		mutate  func(*Dependencies)
		wantMsg string
	}{
		{
			name:    "nil IngestFile handler",
			mutate:  func(d *Dependencies) { d.IngestFile = nil },
			wantMsg: "IngestFile handler must not be nil",
		},
		{
			name:    "nil GetInterchange handler",
			mutate:  func(d *Dependencies) { d.GetInterchange = nil },
			wantMsg: "GetInterchange handler must not be nil",
		},
		{
			name:    "nil GetRawFile handler",
			mutate:  func(d *Dependencies) { d.GetRawFile = nil },
			wantMsg: "GetRawFile handler must not be nil",
		},
		{
			name:    "nil FetchHealth handler",
			mutate:  func(d *Dependencies) { d.FetchHealth = nil },
			wantMsg: "FetchHealth handler must not be nil",
		},
		{
			name:    "nil FetchLiveness handler",
			mutate:  func(d *Dependencies) { d.FetchLiveness = nil },
			wantMsg: "FetchLiveness handler must not be nil",
		},
		{
			name:    "nil FetchReadiness handler",
			mutate:  func(d *Dependencies) { d.FetchReadiness = nil },
			wantMsg: "FetchReadiness handler must not be nil",
		},
		{
			name:    "nil ListMessages handler",
			mutate:  func(d *Dependencies) { d.ListMessages = nil },
			wantMsg: "ListMessages handler must not be nil",
		},
		{
			name:    "nil Metrics",
			mutate:  func(d *Dependencies) { d.Metrics = nil },
			wantMsg: "Metrics must not be nil",
		},
		{
			name:    "nil Tracer",
			mutate:  func(d *Dependencies) { d.Tracer = nil },
			wantMsg: "Tracer must not be nil",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := newTestDeps(log, m, tracer)
			tc.mutate(&deps)

			require.PanicsWithValue(t, tc.wantMsg, func() {
				NewApplication(deps)
			})
		})
	}
}

// newTestDeps returns a Dependencies struct with noop handlers for all fields.
func newTestDeps(log logger.Logger, m *metrics.Metrics, tracer trace.Tracer) Dependencies {
	return Dependencies{
		IngestFile: decorator.CommandHandlerFunc[commands.IngestFileCommand, *commands.IngestFileResult](
			func(_ context.Context, _ commands.IngestFileCommand) (*commands.IngestFileResult, error) {
				return &commands.IngestFileResult{}, nil
			},
		),
		GetInterchange: decorator.QueryHandlerFunc[queries.GetInterchangeQuery, queries.GetInterchangeResult](
			func(_ context.Context, _ queries.GetInterchangeQuery) (queries.GetInterchangeResult, error) {
				return queries.GetInterchangeResult{}, nil
			},
		),
		GetRawFile: decorator.QueryHandlerFunc[queries.GetRawFileQuery, queries.GetRawFileResult](
			func(_ context.Context, _ queries.GetRawFileQuery) (queries.GetRawFileResult, error) {
				return queries.GetRawFileResult{}, nil
			},
		),
		FetchHealth: decorator.QueryHandlerFunc[queries.FetchHealthQuery, queries.HealthResponse](
			func(_ context.Context, _ queries.FetchHealthQuery) (queries.HealthResponse, error) {
				return queries.HealthResponse{}, nil
			},
		),
		FetchLiveness: decorator.QueryHandlerFunc[queries.FetchLivenessQuery, queries.HealthResponse](
			func(_ context.Context, _ queries.FetchLivenessQuery) (queries.HealthResponse, error) {
				return queries.HealthResponse{}, nil
			},
		),
		FetchReadiness: decorator.QueryHandlerFunc[queries.FetchReadinessQuery, queries.HealthResponse](
			func(_ context.Context, _ queries.FetchReadinessQuery) (queries.HealthResponse, error) {
				return queries.HealthResponse{}, nil
			},
		),
		ListMessages: decorator.QueryHandlerFunc[queries.ListMessagesQuery, queries.ListMessagesResult](
			func(_ context.Context, _ queries.ListMessagesQuery) (queries.ListMessagesResult, error) {
				return queries.ListMessagesResult{}, nil
			},
		),
		Logger:  log,
		Metrics: m,
		Tracer:  tracer,
	}
}
