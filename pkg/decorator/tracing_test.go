package decorator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/decorator"
	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

func TestTracingCommandDecorator_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler decorator.CommandHandlerFunc[testCommand, testResult]
		wantErr bool
	}{
		{
			name: "success creates span",
			handler: func(_ context.Context, _ testCommand) (testResult, error) {
				return testResult{Name: "ok"}, nil
			},
			wantErr: false,
		},
		{
			name: "error sets span status",
			handler: func(_ context.Context, _ testCommand) (testResult, error) {
				return testResult{}, errors.New("trace-fail")
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

func TestTracingQueryDecorator_Handle(t *testing.T) {
	t.Parallel()

	handler := decorator.ApplyQueryDecorators[testQuery, testQueryResult](
		decorator.QueryHandlerFunc[testQuery, testQueryResult](func(_ context.Context, _ testQuery) (testQueryResult, error) {
			return testQueryResult{Name: "traced"}, nil
		}),
		logger.NewTestLogger(),
		&noopMetrics{},
		noopTracer(),
	)

	result, err := handler.Handle(context.Background(), testQuery{ID: "1"})

	require.NoError(t, err)
	require.Equal(t, "traced", result.Name)
}
