package ports

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
)

// IngestResult holds the outcome of a file ingestion operation.
type (
	IngestResult struct {
		InterchangeID     uuid.UUID
		Status            model.InterchangeStatus
		MessageCount      int
		SubscriptionCount int
		Events            []*model.DomainEvent
	}
)

// InterchangeDetail is a denormalized view of an interchange with its active version and message count.
type (
	InterchangeDetail struct {
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

// ProcessingVersionSummary is a lightweight view of a processing version.
type (
	ProcessingVersionSummary struct {
		ID            uuid.UUID
		VersionNumber int
		ParserVersion string
		FormatVersion string
		CreatedAt     time.Time
	}
)

// RawFile holds the raw content of an interchange and its hash.
type (
	RawFile struct {
		Content     []byte
		ContentHash string
	}
)

// MessageList holds a paginated list of messages.
type (
	MessageList struct {
		Messages   []*model.Message
		NextCursor *string
	}
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

// IngestionService defines the domain operations for the ingestion-bounded context.
//
//counterfeiter:generate . IngestionService
type (
	IngestionService interface {
		// IngestFile validates, parses, persists, and publishes an EDIFACT file.
		IngestFile(ctx context.Context, fileData []byte, contentType string) (*IngestResult, error)

		// GetInterchange retrieves an interchange with its active version and message count.
		GetInterchange(ctx context.Context, id uuid.UUID) (*InterchangeDetail, error)

		// GetRawFile retrieves the raw content of an interchange.
		GetRawFile(ctx context.Context, id uuid.UUID) (*RawFile, error)

		// ListMessages retrieves messages matching the filter criteria.
		ListMessages(ctx context.Context, filter *model.MessageFilter) (*MessageList, error)
	}
)
