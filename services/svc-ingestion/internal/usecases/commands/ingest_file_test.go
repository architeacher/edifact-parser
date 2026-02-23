package commands

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

func TestIngestFileHandler_Handle(t *testing.T) {
	t.Parallel()

	completedID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name              string
		cmd               IngestFileCommand
		svcResult         *ports.IngestResult
		svcErr            error
		wantErr           bool
		wantErrContains   string
		wantStatus        model.InterchangeStatus
		wantInterchangeID uuid.UUID
		wantMessageCount  int
		wantSubCount      int
		wantEventCount    int
	}{
		{
			name: "successful ingestion delegates to service",
			cmd: IngestFileCommand{
				FileData:    []byte("valid-edifact"),
				ContentType: "application/edifact",
			},
			svcResult: &ports.IngestResult{
				InterchangeID:     completedID,
				Status:            model.StatusCompleted,
				MessageCount:      3,
				SubscriptionCount: 2,
				Events:            make([]*model.DomainEvent, 2),
			},
			wantStatus:        model.StatusCompleted,
			wantInterchangeID: completedID,
			wantMessageCount:  3,
			wantSubCount:      2,
			wantEventCount:    2,
		},
		{
			name: "service error propagated",
			cmd: IngestFileCommand{
				FileData:    nil,
				ContentType: "application/edifact",
			},
			svcErr:          errors.New("validating file: file must not be empty"),
			wantErr:         true,
			wantErrContains: "file must not be empty",
		},
		{
			name: "failed status mapped correctly",
			cmd: IngestFileCommand{
				FileData:    []byte("bad-data"),
				ContentType: "application/edifact",
			},
			svcResult: &ports.IngestResult{
				InterchangeID: uuid.Must(uuid.NewV7()),
				Status:        model.StatusFailed,
				Events:        make([]*model.DomainEvent, 1),
			},
			wantStatus:     model.StatusFailed,
			wantEventCount: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &mockIngestionService{
				ingestResult: tc.svcResult,
				ingestErr:    tc.svcErr,
			}

			handler := NewIngestFileHandler(svc)
			result, err := handler.Handle(context.Background(), tc.cmd)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantErrContains != "" {
					require.ErrorContains(t, err, tc.wantErrContains)
				}
				require.Nil(t, result)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantStatus, result.Status)

			if tc.wantInterchangeID != uuid.Nil {
				require.Equal(t, tc.wantInterchangeID, result.InterchangeID)
			}

			require.Equal(t, tc.wantMessageCount, result.MessageCount)
			require.Equal(t, tc.wantSubCount, result.SubscriptionCount)
			require.Len(t, result.Events, tc.wantEventCount)
		})
	}
}

// --- Mock implementation ---

// mockIngestionService implements ports.IngestionService for handler tests.
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
