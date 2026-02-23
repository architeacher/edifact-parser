package commands

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

// IngestFileCommand carries the parameters for ingesting an EDIFACT file.
type (
	IngestFileCommand struct {
		FileData    []byte
		ContentType string
	}
)

// IngestFileResult holds the outcome of an ingestion operation.
type (
	IngestFileResult struct {
		InterchangeID     uuid.UUID
		Status            model.InterchangeStatus
		MessageCount      int
		SubscriptionCount int
		Events            []*model.DomainEvent
	}
)

// IngestFileHandler delegates file ingestion to the IngestionService.
type (
	IngestFileHandler struct {
		svc ports.IngestionService
	}
)

// NewIngestFileHandler creates a handler with the ingestion service dependency.
func NewIngestFileHandler(svc ports.IngestionService) *IngestFileHandler {
	return &IngestFileHandler{
		svc: svc,
	}
}

// Handle executes the ingestion pipeline by delegating to the service layer.
func (h *IngestFileHandler) Handle(ctx context.Context, cmd IngestFileCommand) (*IngestFileResult, error) {
	result, err := h.svc.IngestFile(ctx, cmd.FileData, cmd.ContentType)
	if err != nil {
		// For duplicate interchanges, propagate both the result and the sentinel
		// error so the HTTP handler can distinguish 200 (duplicate) from 201 (new).
		if errors.Is(err, model.ErrDuplicateInterchange) && result != nil {
			return &IngestFileResult{
				InterchangeID: result.InterchangeID,
				Status:        result.Status,
			}, err
		}

		return nil, err
	}

	return &IngestFileResult{
		InterchangeID:     result.InterchangeID,
		Status:            result.Status,
		MessageCount:      result.MessageCount,
		SubscriptionCount: result.SubscriptionCount,
		Events:            result.Events,
	}, nil
}
