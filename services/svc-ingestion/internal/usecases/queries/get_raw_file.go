package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

// GetRawFileQuery carries the parameters for retrieving raw file content.
type (
	GetRawFileQuery struct {
		InterchangeID uuid.UUID
	}
)

// GetRawFileResult holds the raw content and its hash.
type (
	GetRawFileResult struct {
		Content     []byte
		ContentHash string
	}
)

// NewGetRawFileQuery validates and creates a GetRawFileQuery.
func NewGetRawFileQuery(interchangeID uuid.UUID) (*GetRawFileQuery, error) {
	if interchangeID == uuid.Nil {
		return nil, fmt.Errorf("creating get raw file query: %w", ErrNilInterchangeID)
	}

	return &GetRawFileQuery{
		InterchangeID: interchangeID,
	}, nil
}

// GetRawFileHandler delegates raw file retrieval to the IngestionService.
type (
	GetRawFileHandler struct {
		svc ports.IngestionService
	}
)

// NewGetRawFileHandler creates a handler with the ingestion service dependency.
func NewGetRawFileHandler(svc ports.IngestionService) *GetRawFileHandler {
	return &GetRawFileHandler{
		svc: svc,
	}
}

// Handle retrieves the raw content of an interchange.
func (h *GetRawFileHandler) Handle(ctx context.Context, query GetRawFileQuery) (GetRawFileResult, error) {
	rawFile, err := h.svc.GetRawFile(ctx, query.InterchangeID)
	if err != nil {
		return GetRawFileResult{}, err
	}

	return GetRawFileResult{
		Content:     rawFile.Content,
		ContentHash: rawFile.ContentHash,
	}, nil
}
