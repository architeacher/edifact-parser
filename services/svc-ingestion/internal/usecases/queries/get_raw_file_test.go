package queries

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

func TestNewGetRawFileQuery(t *testing.T) {
	t.Parallel()

	validID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name: "valid ID",
			id:   validID,
		},
		{
			name:    "nil ID",
			id:      uuid.Nil,
			wantErr: ErrNilInterchangeID,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := NewGetRawFileQuery(tc.id)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tc.id, got.InterchangeID)
		})
	}
}

func TestGetRawFileHandler_Handle(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32)
	rawContent := []byte("UNA:+.? 'UNB+UNOC:3+9900000000001:14+9900000000002:14+260115:1000+REF001'")
	interchangeID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name        string
		svcResult   *ports.RawFile
		svcErr      error
		query       GetRawFileQuery
		wantErr     string
		wantContent []byte
		wantHash    string
	}{
		{
			name: "retrieved verbatim",
			svcResult: &ports.RawFile{
				Content:     rawContent,
				ContentHash: validHash,
			},
			query:       GetRawFileQuery{InterchangeID: interchangeID},
			wantContent: rawContent,
			wantHash:    validHash,
		},
		{
			name:    "not found",
			svcErr:  errors.New("fetching raw file: interchange not found"),
			query:   GetRawFileQuery{InterchangeID: uuid.Must(uuid.NewV7())},
			wantErr: "interchange not found",
		},
		{
			name:    "empty raw content returns error",
			svcErr:  errors.New("fetching raw file: raw content must not be empty"),
			query:   GetRawFileQuery{InterchangeID: interchangeID},
			wantErr: "raw content",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &mockIngestionService{
				getRawFileRes: tc.svcResult,
				getRawFileErr: tc.svcErr,
			}

			handler := NewGetRawFileHandler(svc)
			result, err := handler.Handle(context.Background(), tc.query)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantContent, result.Content)
			require.Equal(t, tc.wantHash, result.ContentHash)
		})
	}
}
