package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

const (
	// maxFileSize is the maximum allowed file size in bytes (10 MB).
	maxFileSize = 10 * 1024 * 1024

	// defaultParserVersion is used when no parser version is specified.
	defaultParserVersion = "1.0.0"
)

// Compile-time interface check.
var _ ports.IngestionService = (*IngestionService)(nil)

// IngestionService implements ports.IngestionService by orchestrating
// repositories, parser, transaction manager, and event publisher.
type (
	IngestionService struct {
		interchangeRepo  ports.InterchangeRepository
		versionRepo      ports.ProcessingVersionRepository
		messageRepo      ports.MessageRepository
		subscriptionRepo ports.SubscriptionRepository
		publisher        ports.EventPublisher
		parser           ports.FileParser
		txMgr            ports.TransactionManager
		parserVersion    string
	}
)

// NewIngestionService creates a service with all required dependencies.
func NewIngestionService(
	interchangeRepo ports.InterchangeRepository,
	versionRepo ports.ProcessingVersionRepository,
	messageRepo ports.MessageRepository,
	subscriptionRepo ports.SubscriptionRepository,
	publisher ports.EventPublisher,
	parser ports.FileParser,
	txMgr ports.TransactionManager,
	parserVersion string,
) *IngestionService {
	if parserVersion == "" {
		parserVersion = defaultParserVersion
	}

	return &IngestionService{
		interchangeRepo:  interchangeRepo,
		versionRepo:      versionRepo,
		messageRepo:      messageRepo,
		subscriptionRepo: subscriptionRepo,
		publisher:        publisher,
		parser:           parser,
		txMgr:            txMgr,
		parserVersion:    parserVersion,
	}
}

// IngestFile executes the full ingestion pipeline:
// validate -> dedup -> parse -> persist (tx) -> publish events.
func (s *IngestionService) IngestFile(ctx context.Context, fileData []byte, _ string) (*ports.IngestResult, error) {
	if err := s.validateFile(fileData); err != nil {
		return nil, err
	}

	contentHash := computeSHA256(fileData)

	// Idempotency check: return existing interchange if content already ingested.
	if existing, err := s.interchangeRepo.FindByContentHash(ctx, contentHash); err == nil && existing != nil {
		return &ports.IngestResult{
			InterchangeID: existing.ID,
			Status:        existing.Status,
		}, nil
	}

	// Parse the EDIFACT file.
	parseResult, parseErr := s.parser.Parse(fileData)
	if parseErr != nil {
		return s.handleParseFailure(ctx, fileData, contentHash, parseErr)
	}

	return s.persistAndPublish(ctx, fileData, contentHash, parseResult)
}

// GetInterchange retrieves an interchange with its active version and message count.
func (s *IngestionService) GetInterchange(ctx context.Context, id uuid.UUID) (*ports.InterchangeDetail, error) {
	interchange, err := s.interchangeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching interchange: %w", err)
	}

	detail := &ports.InterchangeDetail{
		ID:          interchange.ID,
		SenderID:    interchange.SenderID,
		ReceiverID:  interchange.ReceiverID,
		Reference:   interchange.Reference,
		PreparedAt:  interchange.PreparedAt,
		Status:      interchange.Status,
		ContentHash: interchange.ContentHash,
	}

	version, err := s.versionRepo.FindActive(ctx, id)
	if err == nil {
		detail.ActiveVersion = &ports.ProcessingVersionSummary{
			ID:            version.ID,
			VersionNumber: version.VersionNumber,
			ParserVersion: version.ParserVersion,
			FormatVersion: version.FormatVersion,
			CreatedAt:     version.CreatedAt,
		}
	}

	count, err := s.messageRepo.Count(ctx, id)
	if err == nil {
		detail.MessageCount = count
	}

	return detail, nil
}

// GetRawFile retrieves the raw content of an interchange.
func (s *IngestionService) GetRawFile(ctx context.Context, id uuid.UUID) (*ports.RawFile, error) {
	interchange, err := s.interchangeRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching raw file: %w", err)
	}

	if len(interchange.RawContent) == 0 {
		return nil, fmt.Errorf("fetching raw file: %w", model.ErrEmptyRawContent)
	}

	return &ports.RawFile{
		Content:     interchange.RawContent,
		ContentHash: interchange.ContentHash,
	}, nil
}

// ListMessages retrieves messages matching the filter criteria.
func (s *IngestionService) ListMessages(ctx context.Context, filter *model.MessageFilter) (*ports.MessageList, error) {
	if filter == nil {
		filter = &model.MessageFilter{}
	}

	if err := filter.Validate(); err != nil {
		return nil, fmt.Errorf("validating message filter: %w", err)
	}

	messages, nextCursor, err := s.messageRepo.ListByFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("listing messages: %w", err)
	}

	return &ports.MessageList{
		Messages:   messages,
		NextCursor: nextCursor,
	}, nil
}

// validateFile checks file size and emptiness.
func (s *IngestionService) validateFile(fileData []byte) error {
	if len(fileData) == 0 {
		return fmt.Errorf("validating file: %w", ErrFileEmpty)
	}

	if len(fileData) > maxFileSize {
		return fmt.Errorf("validating file: %w", ErrFileTooLarge)
	}

	return nil
}

// handleParseFailure creates a failed interchange record and returns a result with status=failed.
func (s *IngestionService) handleParseFailure(
	ctx context.Context,
	rawContent []byte,
	contentHash string,
	parseErr error,
) (*ports.IngestResult, error) {
	interchange := &model.Interchange{
		ID:          uuid.Must(uuid.NewV7()),
		SenderID:    "UNKNOWN",
		ReceiverID:  "UNKNOWN",
		Reference:   "UNKNOWN",
		RawContent:  rawContent,
		ContentHash: contentHash,
		Status:      model.StatusFailed,
		ErrorDetail: ptr(parseErr.Error()),
	}

	// Best effort: persist the failed interchange for auditability.
	_ = s.interchangeRepo.Create(ctx, interchange)

	failedEvent, _ := model.NewFailedEvent(interchange.ID, parseErr.Error())

	var events []*model.DomainEvent
	if failedEvent != nil {
		events = append(events, failedEvent)
		_ = s.publisher.Publish(ctx, failedEvent)
	}

	return &ports.IngestResult{
		InterchangeID: interchange.ID,
		Status:        model.StatusFailed,
		Events:        events,
	}, nil
}

// persistAndPublish runs the entire persistence pipeline within a transaction.
func (s *IngestionService) persistAndPublish(
	ctx context.Context,
	rawContent []byte,
	contentHash string,
	parseResult *ports.ParseResult,
) (*ports.IngestResult, error) {
	tx, err := s.txMgr.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is no-op after commit

	// Create interchange.
	interchange, err := model.NewInterchange(
		parseResult.SenderID,
		parseResult.SenderQualifier,
		parseResult.ReceiverID,
		parseResult.ReceiverQualifier,
		parseResult.PreparedAt,
		parseResult.Reference,
		rawContent,
		contentHash,
	)
	if err != nil {
		return nil, fmt.Errorf("creating interchange: %w", err)
	}

	if err := s.interchangeRepo.Create(ctx, interchange); err != nil {
		if errors.Is(err, model.ErrDuplicateInterchange) {
			existing, findErr := s.interchangeRepo.FindByContentHash(ctx, contentHash)
			if findErr != nil {
				return nil, fmt.Errorf("recovering duplicate interchange: %w", findErr)
			}

			return &ports.IngestResult{
				InterchangeID: existing.ID,
				Status:        existing.Status,
			}, model.ErrDuplicateInterchange
		}

		return nil, fmt.Errorf("creating interchange: %w", err)
	}

	// Create a processing version.
	version, err := model.NewProcessingVersion(
		interchange.ID,
		1,
		s.parserVersion,
		"",
	)
	if err != nil {
		return nil, fmt.Errorf("creating processing version: %w", err)
	}

	if err := s.versionRepo.Create(ctx, version); err != nil {
		return nil, fmt.Errorf("creating processing version: %w", err)
	}

	// Upsert subscriptions first to get their IDs.
	subscriptionIDs := make(map[string]uuid.UUID)
	subscriptionCount := 0

	for _, subID := range parseResult.Subscriptions {
		sub, err := s.subscriptionRepo.UpsertByIdentifier(ctx, subID)
		if err != nil {
			return nil, fmt.Errorf("upserting subscription %q: %w", subID, err)
		}

		subscriptionIDs[subID] = sub.ID
		subscriptionCount++
	}

	// Bulk create messages with correct subscription IDs.
	messages, err := s.buildMessages(interchange.ID, version.ID, parseResult.Messages, subscriptionIDs)
	if err != nil {
		return nil, fmt.Errorf("building messages: %w", err)
	}

	if len(messages) > 0 {
		if err := s.messageRepo.BulkCreate(ctx, messages); err != nil {
			return nil, fmt.Errorf("creating messages: %w", err)
		}
	}

	// Build domain events.
	ingestedEvent, _ := model.NewIngestedEvent(
		interchange.ID,
		len(messages),
		interchange.SenderID,
		interchange.ReceiverID,
	)

	versionEvent, _ := model.NewVersionCreatedEvent(
		interchange.ID,
		version.ID,
		version.VersionNumber,
		version.ParserVersion,
	)

	events := []*model.DomainEvent{ingestedEvent, versionEvent}

	// Commit the transaction.
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	// Publish events post-commit. Failure is logged but does not fail the ingest.
	_ = s.publisher.Publish(ctx, events...)

	return &ports.IngestResult{
		InterchangeID:     interchange.ID,
		Status:            model.StatusCompleted,
		MessageCount:      len(messages),
		SubscriptionCount: subscriptionCount,
		Events:            events,
	}, nil
}

// buildMessages converts parsed messages into domain Message entities using the provided subscription IDs.
func (s *IngestionService) buildMessages(
	interchangeID, versionID uuid.UUID,
	parsed []ports.ParsedMessage,
	subscriptionIDs map[string]uuid.UUID,
) ([]*model.Message, error) {
	messages := make([]*model.Message, 0, len(parsed))

	for _, pm := range parsed {
		var subscriptionID *uuid.UUID
		if pm.SubscriptionID != "" {
			if id, ok := subscriptionIDs[pm.SubscriptionID]; ok {
				subscriptionID = &id
			}
		}

		// Flatten segment slices into a single map for JSONB storage.
		segments := make(map[string]any, len(pm.Segments))
		for index, seg := range pm.Segments {
			key := fmt.Sprintf("segment_%d", index)
			segments[key] = seg
		}

		msg, err := model.NewMessage(
			interchangeID,
			versionID,
			subscriptionID,
			pm.MessageType,
			pm.MessageReference,
			segments,
		)
		if err != nil {
			return nil, fmt.Errorf("building message %q: %w", pm.MessageReference, err)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// computeSHA256 returns the hex-encoded SHA-256 digest of data.
func computeSHA256(data []byte) string {
	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
