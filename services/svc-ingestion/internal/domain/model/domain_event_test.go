package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewDomainEvent(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name          string
		interchangeID *uuid.UUID
		eventType     EventType
		payload       map[string]any
		wantErr       string
	}{
		{
			name:          "valid event with interchange",
			interchangeID: &interchangeID,
			eventType:     EventInterchangeIngested,
			payload:       map[string]any{"message_count": 3},
		},
		{
			name:          "valid event without interchange",
			interchangeID: nil,
			eventType:     EventVersionCreated,
			payload:       map[string]any{},
		},
		{
			name:          "nil payload initialized to empty map",
			interchangeID: &interchangeID,
			eventType:     EventInterchangeFailed,
			payload:       nil,
		},
		{
			name:          "empty event type",
			interchangeID: &interchangeID,
			eventType:     "",
			payload:       nil,
			wantErr:       "event type",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewDomainEvent(tc.interchangeID, tc.eventType, tc.payload)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.NotEqual(t, uuid.Nil, got.ID)
			require.Equal(t, tc.interchangeID, got.InterchangeID)
			require.Equal(t, tc.eventType, got.EventType)
			require.NotNil(t, got.Payload)
			require.Contains(t, got.Payload, "event_id")
			require.Contains(t, got.Payload, "event_type")
			require.Contains(t, got.Payload, "occurred_at")
			require.False(t, got.CreatedAt.IsZero())
		})
	}
}

func TestNewIngestedEvent(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewIngestedEvent(interchangeID, 5, "SENDER1", "RECEIVER1")

	require.NoError(t, err)
	require.Equal(t, EventInterchangeIngested, got.EventType)
	require.Equal(t, 5, got.Payload["message_count"])
	require.Equal(t, "SENDER1", got.Payload["sender"])
	require.Equal(t, "RECEIVER1", got.Payload["receiver"])
}

func TestNewFailedEvent(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewFailedEvent(interchangeID, "parse error")

	require.NoError(t, err)
	require.Equal(t, EventInterchangeFailed, got.EventType)
	require.Equal(t, "parse error", got.Payload["error"])
}

func TestNewVersionCreatedEvent(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())

	got, err := NewVersionCreatedEvent(interchangeID, versionID, 2, "1.1.0")

	require.NoError(t, err)
	require.Equal(t, EventVersionCreated, got.EventType)
	require.Equal(t, versionID.String(), got.Payload["version_id"])
	require.Equal(t, 2, got.Payload["version_number"])
	require.Equal(t, "1.1.0", got.Payload["parser_version"])
}

func TestNewVersionActivatedEvent(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())

	got, err := NewVersionActivatedEvent(interchangeID, versionID, 1)

	require.NoError(t, err)
	require.Equal(t, EventVersionActivated, got.EventType)
	require.Equal(t, versionID.String(), got.Payload["version_id"])
	require.Equal(t, 1, got.Payload["version_number"])
}

func TestNewVersionRolledBackEvent(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewVersionRolledBackEvent(interchangeID, 2, 1)

	require.NoError(t, err)
	require.Equal(t, EventVersionRolledBack, got.EventType)
	require.Equal(t, 2, got.Payload["from_version"])
	require.Equal(t, 1, got.Payload["to_version"])
}

func TestNewDomainEvent_PayloadEnrichment(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	got, err := NewDomainEvent(
		&interchangeID,
		EventInterchangeIngested,
		map[string]any{"custom_key": "custom_value"},
	)

	require.NoError(t, err)
	require.Equal(t, "custom_value", got.Payload["custom_key"])
	require.Equal(t, got.ID.String(), got.Payload["event_id"])
	require.Equal(t, string(EventInterchangeIngested), got.Payload["event_type"])
	require.Contains(t, got.Payload, "occurred_at")
	require.Equal(t, interchangeID.String(), got.Payload["interchange_id"])
}

func TestNewDomainEvent_NilInterchangeIDExcludesPayloadKey(t *testing.T) {
	t.Parallel()

	got, err := NewDomainEvent(nil, EventVersionCreated, map[string]any{})

	require.NoError(t, err)
	require.NotContains(t, got.Payload, "interchange_id")
	require.Nil(t, got.InterchangeID)
}

func TestNewDomainEvent_UniqueIDs(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())

	e1, err := NewDomainEvent(&interchangeID, EventInterchangeIngested, nil)
	require.NoError(t, err)

	e2, err := NewDomainEvent(&interchangeID, EventInterchangeIngested, nil)
	require.NoError(t, err)

	require.NotEqual(t, e1.ID, e2.ID)
}
