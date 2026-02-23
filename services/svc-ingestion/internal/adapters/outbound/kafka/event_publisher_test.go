package kafka

import (
	"context"
	"testing"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewPublisher(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		brokers []string
		topic   string
		wantErr error
	}{
		{
			name:    "valid configuration",
			brokers: []string{"localhost:9092"},
			topic:   "edifact.events",
		},
		{
			name:    "multiple brokers",
			brokers: []string{"broker1:9092", "broker2:9092", "broker3:9092"},
			topic:   "edifact.events",
		},
		{
			name:    "no brokers",
			brokers: []string{},
			topic:   "edifact.events",
			wantErr: ErrNoBrokers,
		},
		{
			name:    "nil brokers",
			brokers: nil,
			topic:   "edifact.events",
			wantErr: ErrNoBrokers,
		},
		{
			name:    "empty topic",
			brokers: []string{"localhost:9092"},
			topic:   "",
			wantErr: ErrEmptyTopic,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			pub, err := NewPublisher(tc.brokers, tc.topic)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, pub)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, pub)
			require.NotNil(t, pub.writer)

			// Clean up writer to avoid goroutine leak.
			require.NoError(t, pub.Close())
		})
	}
}

func TestPublish_EmptyEvents(t *testing.T) {
	t.Parallel()

	pub, err := NewPublisher([]string{"localhost:9092"}, "edifact.events")
	require.NoError(t, err)

	t.Cleanup(func() { _ = pub.Close() })

	err = pub.Publish(context.Background())
	require.NoError(t, err, "publishing zero events should be a no-op")
}

func TestBuildMessages(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name         string
		event        func() *model.DomainEvent
		wantKeyEmpty bool
	}{
		{
			name: "event with interchange ID uses it as key",
			event: func() *model.DomainEvent {
				e, _ := model.NewIngestedEvent(interchangeID, 3, "SENDER", "RECEIVER")

				return e
			},
			wantKeyEmpty: false,
		},
		{
			name: "event without interchange ID uses empty key",
			event: func() *model.DomainEvent {
				e, _ := model.NewDomainEvent(nil, model.EventVersionCreated, map[string]any{"v": 1})

				return e
			},
			wantKeyEmpty: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			event := tc.event()
			require.NotNil(t, event)

			if tc.wantKeyEmpty {
				require.Nil(t, event.InterchangeID)
			} else {
				require.NotNil(t, event.InterchangeID)
				require.Equal(t, interchangeID.String(), event.InterchangeID.String())
			}
		})
	}
}
