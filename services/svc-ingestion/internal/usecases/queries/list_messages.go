package queries

import (
	"context"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

// ListMessagesQuery carries the parameters for listing messages with filtering and pagination.
type (
	ListMessagesQuery struct {
		Filter *model.MessageFilter
	}
)

// ListMessagesResult holds the paginated list of messages.
type (
	ListMessagesResult struct {
		Messages   []*model.Message
		NextCursor *string
	}
)

// ListMessagesHandler delegates message listing to the IngestionService.
type (
	ListMessagesHandler struct {
		svc ports.IngestionService
	}
)

// NewListMessagesHandler creates a handler with the ingestion service dependency.
func NewListMessagesHandler(svc ports.IngestionService) *ListMessagesHandler {
	return &ListMessagesHandler{
		svc: svc,
	}
}

// Handle retrieves messages matching the filter criteria.
func (h *ListMessagesHandler) Handle(ctx context.Context, query ListMessagesQuery) (ListMessagesResult, error) {
	msgList, err := h.svc.ListMessages(ctx, query.Filter)
	if err != nil {
		return ListMessagesResult{}, err
	}

	return ListMessagesResult{
		Messages:   msgList.Messages,
		NextCursor: msgList.NextCursor,
	}, nil
}
