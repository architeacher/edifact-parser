package queries

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

func TestListMessagesHandler_Handle(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())
	nextCursor := "next-page-cursor"

	msg1 := &model.Message{
		ID:                  uuid.Must(uuid.NewV7()),
		InterchangeID:       interchangeID,
		ProcessingVersionID: versionID,
		MessageType:         "MSCONS",
		MessageReference:    "MSG001",
	}

	msg2 := &model.Message{
		ID:                  uuid.Must(uuid.NewV7()),
		InterchangeID:       interchangeID,
		ProcessingVersionID: versionID,
		MessageType:         "INVOIC",
		MessageReference:    "MSG002",
	}

	cases := []struct {
		name         string
		svcResult    *ports.MessageList
		svcErr       error
		query        ListMessagesQuery
		wantErr      string
		wantCount    int
		wantCursor   bool
		wantMsgTypes []string
	}{
		{
			name: "returns messages with cursor",
			svcResult: &ports.MessageList{
				Messages:   []*model.Message{msg1, msg2},
				NextCursor: &nextCursor,
			},
			query: ListMessagesQuery{
				Filter: &model.MessageFilter{
					InterchangeID: &interchangeID,
					Limit:         10,
				},
			},
			wantCount:    2,
			wantCursor:   true,
			wantMsgTypes: []string{"MSCONS", "INVOIC"},
		},
		{
			name: "returns empty list",
			svcResult: &ports.MessageList{
				Messages: []*model.Message{},
			},
			query: ListMessagesQuery{
				Filter: &model.MessageFilter{
					InterchangeID: &interchangeID,
				},
			},
			wantCount: 0,
		},
		{
			name: "nil filter delegated to service",
			svcResult: &ports.MessageList{
				Messages: []*model.Message{msg1},
			},
			query:        ListMessagesQuery{},
			wantCount:    1,
			wantMsgTypes: []string{"MSCONS"},
		},
		{
			name:   "service error propagated",
			svcErr: errors.New("listing messages: database timeout"),
			query: ListMessagesQuery{
				Filter: &model.MessageFilter{},
			},
			wantErr: "listing messages",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := &mockIngestionService{
				listMessagesRes: tc.svcResult,
				listMessagesErr: tc.svcErr,
			}

			handler := NewListMessagesHandler(svc)
			result, err := handler.Handle(context.Background(), tc.query)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.Len(t, result.Messages, tc.wantCount)

			if tc.wantCursor {
				require.NotNil(t, result.NextCursor)
			}

			for index, wantType := range tc.wantMsgTypes {
				require.Equal(t, wantType, result.Messages[index].MessageType)
			}
		})
	}
}
