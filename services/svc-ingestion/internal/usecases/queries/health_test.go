package queries

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFetchHealthHandler_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		dbErr      error
		kafkaErr   error
		wantStatus string
		wantDBStat string
		wantKafka  string
	}{
		{
			name:       "all healthy",
			wantStatus: HealthStatusHealthy,
			wantDBStat: HealthStatusHealthy,
			wantKafka:  HealthStatusHealthy,
		},
		{
			name:       "database failure",
			dbErr:      errSentinel,
			wantStatus: HealthStatusDegraded,
			wantDBStat: HealthStatusUnhealthy,
			wantKafka:  HealthStatusHealthy,
		},
		{
			name:       "kafka failure",
			kafkaErr:   errSentinel,
			wantStatus: HealthStatusDegraded,
			wantDBStat: HealthStatusHealthy,
			wantKafka:  HealthStatusUnhealthy,
		},
		{
			name:       "all unhealthy",
			dbErr:      errSentinel,
			kafkaErr:   errSentinel,
			wantStatus: HealthStatusUnhealthy,
			wantDBStat: HealthStatusUnhealthy,
			wantKafka:  HealthStatusUnhealthy,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := NewFetchHealthHandler(
				&mockHealthChecker{err: tc.dbErr},
				&mockHealthChecker{err: tc.kafkaErr},
			)

			result, err := handler.Handle(context.Background(), FetchHealthQuery{})

			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, result.Status)
			require.Equal(t, tc.wantDBStat, result.Database.Status)
			require.Equal(t, tc.wantKafka, result.Kafka.Status)
			require.False(t, result.Timestamp.IsZero())
		})
	}
}

func TestFetchLivenessHandler_Handle(t *testing.T) {
	t.Parallel()

	handler := NewFetchLivenessHandler()

	result, err := handler.Handle(context.Background(), FetchLivenessQuery{})

	require.NoError(t, err)
	require.Equal(t, HealthStatusHealthy, result.Status)
	require.False(t, result.Timestamp.IsZero())
}

func TestFetchReadinessHandler_Handle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		dbErr      error
		kafkaErr   error
		wantStatus string
	}{
		{
			name:       "all ready",
			wantStatus: HealthStatusHealthy,
		},
		{
			name:       "database not ready",
			dbErr:      errSentinel,
			wantStatus: HealthStatusUnhealthy,
		},
		{
			name:       "kafka not ready",
			kafkaErr:   errSentinel,
			wantStatus: HealthStatusUnhealthy,
		},
		{
			name:       "nothing ready",
			dbErr:      errSentinel,
			kafkaErr:   errSentinel,
			wantStatus: HealthStatusUnhealthy,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := NewFetchReadinessHandler(
				&mockHealthChecker{err: tc.dbErr},
				&mockHealthChecker{err: tc.kafkaErr},
			)

			result, err := handler.Handle(context.Background(), FetchReadinessQuery{})

			require.NoError(t, err)
			require.Equal(t, tc.wantStatus, result.Status)
			require.False(t, result.Timestamp.IsZero())
		})
	}
}
