package queries

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

// GetInterchangeQuery carries the parameters for retrieving an interchange.
type (
	GetInterchangeQuery struct {
		InterchangeID uuid.UUID
	}
)

// ProcessingVersionSummary is a lightweight view of the active processing version.
type (
	ProcessingVersionSummary struct {
		ID            uuid.UUID
		VersionNumber int
		ParserVersion string
		FormatVersion string
		CreatedAt     time.Time
	}
)

// GetInterchangeResult holds the denormalized view of an interchange.
type (
	GetInterchangeResult struct {
		ID            uuid.UUID
		SenderID      string
		ReceiverID    string
		Reference     string
		PreparedAt    time.Time
		Status        model.InterchangeStatus
		ContentHash   string
		ActiveVersion *ProcessingVersionSummary
		MessageCount  int64
	}
)

// NewGetInterchangeQuery validates and creates a GetInterchangeQuery.
func NewGetInterchangeQuery(interchangeID uuid.UUID) (*GetInterchangeQuery, error) {
	if interchangeID == uuid.Nil {
		return nil, fmt.Errorf("creating get interchange query: %w", ErrNilInterchangeID)
	}

	return &GetInterchangeQuery{
		InterchangeID: interchangeID,
	}, nil
}

// GetInterchangeHandler delegates interchange retrieval to the IngestionService.
type (
	GetInterchangeHandler struct {
		svc ports.IngestionService
	}
)

// NewGetInterchangeHandler creates a handler with the ingestion service dependency.
func NewGetInterchangeHandler(svc ports.IngestionService) *GetInterchangeHandler {
	return &GetInterchangeHandler{
		svc: svc,
	}
}

// Handle retrieves an interchange with its active version and message count.
func (h *GetInterchangeHandler) Handle(ctx context.Context, query GetInterchangeQuery) (GetInterchangeResult, error) {
	detail, err := h.svc.GetInterchange(ctx, query.InterchangeID)
	if err != nil {
		return GetInterchangeResult{}, err
	}

	result := GetInterchangeResult{
		ID:           detail.ID,
		SenderID:     detail.SenderID,
		ReceiverID:   detail.ReceiverID,
		Reference:    detail.Reference,
		PreparedAt:   detail.PreparedAt,
		Status:       detail.Status,
		ContentHash:  detail.ContentHash,
		MessageCount: detail.MessageCount,
	}

	if detail.ActiveVersion != nil {
		result.ActiveVersion = &ProcessingVersionSummary{
			ID:            detail.ActiveVersion.ID,
			VersionNumber: detail.ActiveVersion.VersionNumber,
			ParserVersion: detail.ActiveVersion.ParserVersion,
			FormatVersion: detail.ActiveVersion.FormatVersion,
			CreatedAt:     detail.ActiveVersion.CreatedAt,
		}
	}

	return result, nil
}
