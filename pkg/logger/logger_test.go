package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/pkg/logger"
)

func TestNew(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		level  string
		format string
	}{
		{
			name:   "creates logger with debug level",
			level:  logger.LogLevelDebug,
			format: logger.ConsoleLoggingFormat,
		},
		{
			name:   "creates logger with info level",
			level:  logger.LogLevelInfo,
			format: logger.ConsoleLoggingFormat,
		},
		{
			name:   "creates logger with json format",
			level:  logger.LogLevelInfo,
			format: logger.JSONLoggingFormat,
		},
		{
			name:   "creates logger with default level for unknown",
			level:  "unknown",
			format: logger.ConsoleLoggingFormat,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			log := logger.New(tc.level, tc.format)
			require.NotNil(t, log)
		})
	}
}

func TestWithContext(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name             string
		setupContext     func() context.Context
		expectedField    string
		expectedValue    string
		hasExpectedField bool
	}{
		{
			name: "adds correlation ID to logger",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), logger.ContextKeyCorrelationID, "corr-123")
			},
			expectedField:    "correlation_id",
			expectedValue:    "corr-123",
			hasExpectedField: true,
		},
		{
			name: "adds request ID to logger",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), logger.ContextKeyRequestID, "req-456")
			},
			expectedField:    "request_id",
			expectedValue:    "req-456",
			hasExpectedField: true,
		},
		{
			name: "handles empty context",
			setupContext: func() context.Context {
				return context.Background()
			},
			hasExpectedField: false,
		},
		{
			name: "handles empty correlation ID",
			setupContext: func() context.Context {
				return context.WithValue(context.Background(), logger.ContextKeyCorrelationID, "")
			},
			expectedField:    "correlation_id",
			hasExpectedField: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var buf bytes.Buffer
			log := logger.NewWithWriter(logger.LogLevelInfo, logger.JSONLoggingFormat, &buf)

			ctx := tc.setupContext()
			ctxLogger := log.WithContext(ctx)

			ctxLogger.Info().Msg("test message")

			if tc.hasExpectedField {
				var logEntry map[string]any
				err := json.Unmarshal(buf.Bytes(), &logEntry)
				require.NoError(t, err)
				require.Equal(t, tc.expectedValue, logEntry[tc.expectedField])
			} else if tc.expectedField != "" {
				var logEntry map[string]any
				err := json.Unmarshal(buf.Bytes(), &logEntry)
				require.NoError(t, err)
				_, exists := logEntry[tc.expectedField]
				require.False(t, exists, "field %q should not be present", tc.expectedField)
			}
		})
	}
}

func TestNewWithWriter_LevelSuppression(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	log := logger.NewWithWriter(logger.LogLevelInfo, logger.JSONLoggingFormat, &buf)

	log.Debug().Msg("should not appear")

	require.Empty(t, buf.String())
}

func TestNewTestLogger(t *testing.T) {
	t.Parallel()

	log := logger.NewTestLogger()
	// Should not panic or produce output.
	log.Info().Msg("noop test")
	log.Error().Err(nil).Msg("noop error")
}
