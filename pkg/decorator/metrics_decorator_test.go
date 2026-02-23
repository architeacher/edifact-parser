package decorator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/decorator"
	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

func TestMetricsCommandDecorator_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler decorator.CommandHandlerFunc[testCommand, testResult]
		wantErr bool
	}{
		{
			name: "success increments counter",
			handler: func(_ context.Context, _ testCommand) (testResult, error) {
				return testResult{Name: "ok"}, nil
			},
			wantErr: false,
		},
		{
			name: "error increments error counter",
			handler: func(_ context.Context, _ testCommand) (testResult, error) {
				return testResult{}, errors.New("fail")
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			log := logger.NewTestLogger()
			handler := decorator.ApplyCommandDecorators[testCommand, testResult](
				tc.handler,
				log,
				&noopMetrics{},
				noopTracer(),
			)

			_, err := handler.Handle(context.Background(), testCommand{Value: "x"})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMetricsQueryDecorator_Handle(t *testing.T) {
	t.Parallel()

	handler := decorator.ApplyQueryDecorators[testQuery, testQueryResult](
		decorator.QueryHandlerFunc[testQuery, testQueryResult](func(_ context.Context, _ testQuery) (testQueryResult, error) {
			return testQueryResult{Name: "ok"}, nil
		}),
		logger.NewTestLogger(),
		&noopMetrics{},
		noopTracer(),
	)

	result, err := handler.Handle(context.Background(), testQuery{ID: "1"})

	require.NoError(t, err)
	require.Equal(t, "ok", result.Name)
}

func TestNilMetrics_DoesNotPanic(t *testing.T) {
	t.Parallel()

	handler := decorator.ApplyCommandDecorators[testCommand, testResult](
		decorator.CommandHandlerFunc[testCommand, testResult](func(_ context.Context, _ testCommand) (testResult, error) {
			return testResult{Name: "ok"}, nil
		}),
		logger.NewTestLogger(),
		nil,
		noopTracer(),
	)

	result, err := handler.Handle(context.Background(), testCommand{Value: "x"})

	require.NoError(t, err)
	require.Equal(t, "ok", result.Name)
}
