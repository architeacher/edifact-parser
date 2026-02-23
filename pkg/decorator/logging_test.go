package decorator_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/decorator"
	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

func TestCommandLoggingDecorator_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler decorator.CommandHandlerFunc[testCommand, testResult]
		wantErr bool
	}{
		{
			name: "success is logged",
			handler: func(_ context.Context, _ testCommand) (testResult, error) {
				return testResult{Name: "ok"}, nil
			},
			wantErr: false,
		},
		{
			name: "error is logged",
			handler: func(_ context.Context, _ testCommand) (testResult, error) {
				return testResult{}, errors.New("boom")
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

func TestQueryLoggingDecorator_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler decorator.QueryHandlerFunc[testQuery, testQueryResult]
		wantErr bool
		want    testQueryResult
	}{
		{
			name: "success returns result",
			handler: func(_ context.Context, _ testQuery) (testQueryResult, error) {
				return testQueryResult{Name: "found"}, nil
			},
			want: testQueryResult{Name: "found"},
		},
		{
			name: "error is propagated",
			handler: func(_ context.Context, _ testQuery) (testQueryResult, error) {
				return testQueryResult{}, errors.New("not found")
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			log := logger.NewTestLogger()
			handler := decorator.ApplyQueryDecorators[testQuery, testQueryResult](
				tc.handler,
				log,
				&noopMetrics{},
				noopTracer(),
			)

			result, err := handler.Handle(context.Background(), testQuery{ID: "1"})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.want, result)
			}
		})
	}
}
