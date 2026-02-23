package queries

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

func TestNewGetInterchangeQuery(t *testing.T) {
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

			got, err := NewGetInterchangeQuery(tc.id)

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

func TestGetInterchangeHandler_Handle(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	cases := []struct {
		name             string
		svcResult        *ports.InterchangeDetail
		svcErr           error
		query            GetInterchangeQuery
		wantErr          string
		wantSenderID     string
		wantVersionNum   int
		wantMessageCount int64
	}{
		{
			name: "found with active version and messages",
			svcResult: &ports.InterchangeDetail{
				ID:          interchangeID,
				SenderID:    "9900000000001",
				ReceiverID:  "9900000000002",
				Reference:   "REF001",
				PreparedAt:  preparedAt,
				Status:      model.StatusCompleted,
				ContentHash: "abc123",
				ActiveVersion: &ports.ProcessingVersionSummary{
					ID:            versionID,
					VersionNumber: 1,
					ParserVersion: "1.0.0",
					FormatVersion: "FV2310",
					CreatedAt:     time.Now().UTC(),
				},
				MessageCount: 5,
			},
			query:            GetInterchangeQuery{InterchangeID: interchangeID},
			wantSenderID:     "9900000000001",
			wantVersionNum:   1,
			wantMessageCount: 5,
		},
		{
			name:    "not found",
			svcErr:  errors.New("fetching interchange: interchange not found"),
			query:   GetInterchangeQuery{InterchangeID: uuid.Must(uuid.NewV7())},
			wantErr: "interchange not found",
		},
		{
			name: "found without active version",
			svcResult: &ports.InterchangeDetail{
				ID:          interchangeID,
				SenderID:    "9900000000001",
				ReceiverID:  "9900000000002",
				Reference:   "REF001",
				PreparedAt:  preparedAt,
				Status:      model.StatusCompleted,
				ContentHash: "abc123",
			},
			query:        GetInterchangeQuery{InterchangeID: interchangeID},
			wantSenderID: "9900000000001",
		},
		{
			name: "message count matches",
			svcResult: &ports.InterchangeDetail{
				ID:          interchangeID,
				SenderID:    "9900000000001",
				ReceiverID:  "9900000000002",
				Reference:   "REF001",
				PreparedAt:  preparedAt,
				Status:      model.StatusCompleted,
				ContentHash: "abc123",
				ActiveVersion: &ports.ProcessingVersionSummary{
					ID:            versionID,
					VersionNumber: 1,
					ParserVersion: "1.0.0",
					FormatVersion: "FV2310",
					CreatedAt:     time.Now().UTC(),
				},
				MessageCount: 12,
			},
			query:            GetInterchangeQuery{InterchangeID: interchangeID},
			wantSenderID:     "9900000000001",
			wantVersionNum:   1,
			wantMessageCount: 12,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &mockIngestionService{
				getInterchangeRes: tc.svcResult,
				getInterchangeErr: tc.svcErr,
			}

			handler := NewGetInterchangeHandler(svc)
			result, err := handler.Handle(context.Background(), tc.query)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.wantSenderID, result.SenderID)
			require.Equal(t, tc.wantMessageCount, result.MessageCount)

			if tc.wantVersionNum > 0 {
				require.NotNil(t, result.ActiveVersion)
				require.Equal(t, tc.wantVersionNum, result.ActiveVersion.VersionNumber)
			}
		})
	}
}

// --- Mock implementations for ports interfaces ---

// errSentinel is a reusable test error.
var errSentinel = errors.New("test error")

// mockIngestionService implements ports.IngestionService for query handler tests.
type (
	mockIngestionService struct {
		ingestResult      *ports.IngestResult
		ingestErr         error
		getInterchangeRes *ports.InterchangeDetail
		getInterchangeErr error
		getRawFileRes     *ports.RawFile
		getRawFileErr     error
		listMessagesRes   *ports.MessageList
		listMessagesErr   error
	}
)

func (m *mockIngestionService) IngestFile(_ context.Context, _ []byte, _ string) (*ports.IngestResult, error) {
	return m.ingestResult, m.ingestErr
}

func (m *mockIngestionService) GetInterchange(_ context.Context, _ uuid.UUID) (*ports.InterchangeDetail, error) {
	return m.getInterchangeRes, m.getInterchangeErr
}

func (m *mockIngestionService) GetRawFile(_ context.Context, _ uuid.UUID) (*ports.RawFile, error) {
	return m.getRawFileRes, m.getRawFileErr
}

func (m *mockIngestionService) ListMessages(_ context.Context, _ *model.MessageFilter) (*ports.MessageList, error) {
	return m.listMessagesRes, m.listMessagesErr
}

// mockHealthChecker implements ports.HealthChecker for tests.
type (
	mockHealthChecker struct {
		err error
	}
)

func (m *mockHealthChecker) Ping(_ context.Context) error {
	return m.err
}
