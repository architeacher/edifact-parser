package model

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestNewInterchange(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32) // 64 hex chars
	validContent := []byte("UNA:+.? 'UNB+...")
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name              string
		sender            string
		senderQualifier   string
		receiver          string
		receiverQualifier string
		preparedAt        time.Time
		reference         string
		rawContent        []byte
		contentHash       string
		wantErr           string
	}{
		{
			name:              "valid interchange",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        validContent,
			contentHash:       validHash,
		},
		{
			name:              "empty sender",
			sender:            "",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        validContent,
			contentHash:       validHash,
			wantErr:           "sender",
		},
		{
			name:              "empty receiver",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        validContent,
			contentHash:       validHash,
			wantErr:           "receiver",
		},
		{
			name:              "empty reference",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "",
			rawContent:        validContent,
			contentHash:       validHash,
			wantErr:           "reference",
		},
		{
			name:              "nil raw content",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        nil,
			contentHash:       validHash,
			wantErr:           "raw content",
		},
		{
			name:              "empty raw content",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        []byte{},
			contentHash:       validHash,
			wantErr:           "raw content",
		},
		{
			name:              "content hash too short",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        validContent,
			contentHash:       "abc123",
			wantErr:           "content hash",
		},
		{
			name:              "content hash with non-hex chars",
			sender:            "9900000000001",
			senderQualifier:   "14",
			receiver:          "9900000000002",
			receiverQualifier: "14",
			preparedAt:        preparedAt,
			reference:         "REF001",
			rawContent:        validContent,
			contentHash:       strings.Repeat("zz", 32),
			wantErr:           "content hash",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewInterchange(
				tc.sender,
				tc.senderQualifier,
				tc.receiver,
				tc.receiverQualifier,
				tc.preparedAt,
				tc.reference,
				tc.rawContent,
				tc.contentHash,
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
			require.Equal(t, tc.sender, got.SenderID)
			require.Equal(t, tc.senderQualifier, got.SenderQualifier)
			require.Equal(t, tc.receiver, got.ReceiverID)
			require.Equal(t, tc.receiverQualifier, got.ReceiverQualifier)
			require.Equal(t, tc.preparedAt, got.PreparedAt)
			require.Equal(t, tc.reference, got.Reference)
			require.Equal(t, tc.rawContent, got.RawContent)
			require.Equal(t, tc.contentHash, got.ContentHash)
			require.Equal(t, StatusCompleted, got.Status)
			require.Nil(t, got.ErrorDetail)
			require.False(t, got.CreatedAt.IsZero())
			require.False(t, got.UpdatedAt.IsZero())
		})
	}
}

func TestNewInterchange_UniqueIDs(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	i1, err := NewInterchange("S1", "14", "R1", "14", preparedAt, "REF001", []byte("data1"), validHash)
	require.NoError(t, err)

	i2, err := NewInterchange("S1", "14", "R1", "14", preparedAt, "REF002", []byte("data2"), validHash)
	require.NoError(t, err)

	require.NotEqual(t, i1.ID, i2.ID)
}

func TestNewInterchange_SentinelErrors(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name        string
		sender      string
		receiver    string
		reference   string
		rawContent  []byte
		contentHash string
		wantErr     error
	}{
		{
			name:        "empty sender wraps ErrEmptySender",
			sender:      "",
			receiver:    "R1",
			reference:   "REF",
			rawContent:  []byte("data"),
			contentHash: validHash,
			wantErr:     ErrEmptySender,
		},
		{
			name:        "empty receiver wraps ErrEmptyReceiver",
			sender:      "S1",
			receiver:    "",
			reference:   "REF",
			rawContent:  []byte("data"),
			contentHash: validHash,
			wantErr:     ErrEmptyReceiver,
		},
		{
			name:        "empty reference wraps ErrEmptyReference",
			sender:      "S1",
			receiver:    "R1",
			reference:   "",
			rawContent:  []byte("data"),
			contentHash: validHash,
			wantErr:     ErrEmptyReference,
		},
		{
			name:        "empty raw content wraps ErrEmptyRawContent",
			sender:      "S1",
			receiver:    "R1",
			reference:   "REF",
			rawContent:  nil,
			contentHash: validHash,
			wantErr:     ErrEmptyRawContent,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewInterchange(tc.sender, "14", tc.receiver, "14", preparedAt, tc.reference, tc.rawContent, tc.contentHash)

			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestInterchange_MarkFailed(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		reason  string
		wantErr string
	}{
		{
			name:   "valid failure reason",
			reason: "parse error at segment UNH",
		},
		{
			name:    "empty reason",
			reason:  "",
			wantErr: "reason",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			interchange, err := NewInterchange(
				"9900000000001",
				"14",
				"9900000000002",
				"14",
				preparedAt,
				"REF001",
				[]byte("UNA:+.? 'UNB+..."),
				validHash,
			)
			require.NoError(t, err)

			err = interchange.MarkFailed(tc.reason)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)
				require.Equal(t, StatusCompleted, interchange.Status)

				return
			}

			require.NoError(t, err)
			require.Equal(t, StatusFailed, interchange.Status)
			require.NotNil(t, interchange.ErrorDetail)
			require.Equal(t, tc.reason, *interchange.ErrorDetail)
		})
	}
}
