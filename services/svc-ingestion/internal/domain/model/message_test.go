package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())
	subscriptionID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name           string
		interchangeID  uuid.UUID
		versionID      uuid.UUID
		subscriptionID *uuid.UUID
		msgType        string
		msgRef         string
		segments       map[string]any
		wantErr        string
	}{
		{
			name:           "valid message with subscription",
			interchangeID:  interchangeID,
			versionID:      versionID,
			subscriptionID: &subscriptionID,
			msgType:        "MSCONS",
			msgRef:         "MSG001",
			segments:       map[string]any{"UNH": "test"},
		},
		{
			name:           "valid message without subscription",
			interchangeID:  interchangeID,
			versionID:      versionID,
			subscriptionID: nil,
			msgType:        "INVOIC",
			msgRef:         "MSG002",
			segments:       map[string]any{},
		},
		{
			name:           "valid message with nil segments",
			interchangeID:  interchangeID,
			versionID:      versionID,
			subscriptionID: nil,
			msgType:        "UTILMD",
			msgRef:         "MSG003",
			segments:       nil,
		},
		{
			name:           "empty message type",
			interchangeID:  interchangeID,
			versionID:      versionID,
			subscriptionID: nil,
			msgType:        "",
			msgRef:         "MSG001",
			segments:       nil,
			wantErr:        "message type",
		},
		{
			name:           "empty message reference",
			interchangeID:  interchangeID,
			versionID:      versionID,
			subscriptionID: nil,
			msgType:        "MSCONS",
			msgRef:         "",
			segments:       nil,
			wantErr:        "message reference",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewMessage(
				tc.interchangeID,
				tc.versionID,
				tc.subscriptionID,
				tc.msgType,
				tc.msgRef,
				tc.segments,
			)

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
			require.Equal(t, tc.versionID, got.ProcessingVersionID)
			require.Equal(t, tc.subscriptionID, got.SubscriptionID)
			require.Equal(t, tc.msgType, got.MessageType)
			require.Equal(t, tc.msgRef, got.MessageReference)
			require.False(t, got.CreatedAt.IsZero())
		})
	}
}

func TestNewMessage_SegmentsPreserved(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())

	segments := map[string]any{
		"UNH": map[string]any{"reference": "MSG001", "type": "MSCONS"},
		"DTM": []string{"2026-01-15", "2026-01-16"},
		"LOC": 42,
	}

	got, err := NewMessage(interchangeID, versionID, nil, "MSCONS", "MSG001", segments)

	require.NoError(t, err)
	require.Len(t, got.Segments, 3)
	require.Equal(t, 42, got.Segments["LOC"])
}

func TestNewMessage_UniqueIDs(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())

	m1, err := NewMessage(interchangeID, versionID, nil, "MSCONS", "MSG001", nil)
	require.NoError(t, err)

	m2, err := NewMessage(interchangeID, versionID, nil, "MSCONS", "MSG002", nil)
	require.NoError(t, err)

	require.NotEqual(t, m1.ID, m2.ID)
}

func TestNewMessage_AllMessageTypes(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name    string
		msgType string
	}{
		{name: "MSCONS type", msgType: "MSCONS"},
		{name: "INVOIC type", msgType: "INVOIC"},
		{name: "UTILMD type", msgType: "UTILMD"},
		{name: "ORDERS type", msgType: "ORDERS"},
		{name: "ORDRSP type", msgType: "ORDRSP"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewMessage(interchangeID, versionID, nil, tc.msgType, "REF001", nil)

			require.NoError(t, err)
			require.Equal(t, tc.msgType, got.MessageType)
		})
	}
}

func TestNewMessage_NilInterchangeID(t *testing.T) {
	t.Parallel()

	versionID := uuid.Must(uuid.NewV7())

	got, err := NewMessage(uuid.Nil, versionID, nil, "MSCONS", "MSG001", nil)

	require.NoError(t, err)
	require.Equal(t, uuid.Nil, got.InterchangeID)
}
