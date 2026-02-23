package repos

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

const (
	messagesTable = "messages"
)

type (
	messageRow struct {
		ID                  uuid.UUID  `db:"id"`
		InterchangeID       uuid.UUID  `db:"interchange_id"`
		ProcessingVersionID uuid.UUID  `db:"processing_version_id"`
		SubscriptionID      *uuid.UUID `db:"subscription_id"`
		MessageType         string     `db:"message_type"`
		Reference           string     `db:"reference"`
		Segments            []byte     `db:"segments"`
		CreatedAt           time.Time  `db:"created_at"`
	}

	messageRepository struct {
		db DBTX
		qb sq.StatementBuilderType
	}
)

// NewMessageRepository creates a new MessageRepository backed by PostgreSQL.
func NewMessageRepository(db DBTX) ports.MessageRepository {
	return &messageRepository{
		db: db,
		qb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *messageRepository) BulkCreate(ctx context.Context, messages []*model.Message) error {
	if len(messages) == 0 {
		return nil
	}

	builder := r.qb.
		Insert(messagesTable).
		Columns(
			"id", "interchange_id", "processing_version_id",
			"subscription_id", "message_type", "reference",
			"segments", "created_at",
		)

	for _, msg := range messages {
		segJSON, err := json.Marshal(msg.Segments)
		if err != nil {
			return fmt.Errorf("marshalling message segments: %w", err)
		}

		builder = builder.Values(
			msg.ID, msg.InterchangeID, msg.ProcessingVersionID,
			msg.SubscriptionID, msg.MessageType, msg.MessageReference,
			segJSON, msg.CreatedAt,
		)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("building bulk insert messages query: %w", err)
	}

	if _, execErr := r.db.Exec(ctx, query, args...); execErr != nil {
		return fmt.Errorf("bulk inserting messages: %w", execErr)
	}

	return nil
}

func (r *messageRepository) ListByFilter(ctx context.Context, filter *model.MessageFilter) ([]*model.Message, *string, error) {
	if err := filter.Validate(); err != nil {
		return nil, nil, fmt.Errorf("validating message filter: %w", err)
	}

	builder := r.qb.
		Select(messageColumns()...).
		From(messagesTable).
		OrderBy("id ASC").
		Limit(uint64(filter.Limit + 1)) // fetch one extra to detect next page

	if filter.InterchangeID != nil {
		builder = builder.Where(sq.Eq{"interchange_id": *filter.InterchangeID})
	}

	if filter.ProcessingVersionID != nil {
		builder = builder.Where(sq.Eq{"processing_version_id": *filter.ProcessingVersionID})
	}

	if filter.SubscriptionID != nil {
		builder = builder.Where(sq.Eq{"subscription_id": *filter.SubscriptionID})
	}

	if filter.MessageType != nil {
		builder = builder.Where(sq.Eq{"message_type": *filter.MessageType})
	}

	if filter.Cursor != nil {
		cursorID, err := uuid.Parse(*filter.Cursor)
		if err != nil {
			return nil, nil, fmt.Errorf("parsing cursor: %w", err)
		}

		builder = builder.Where(sq.Gt{"id": cursorID})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, nil, fmt.Errorf("building list messages query: %w", err)
	}

	var rows []messageRow
	if scanErr := pgxscan.Select(ctx, r.db, &rows, query, args...); scanErr != nil {
		return nil, nil, fmt.Errorf("listing messages: %w", scanErr)
	}

	var nextCursor *string

	if len(rows) > filter.Limit {
		rows = rows[:filter.Limit]

		cursor := rows[filter.Limit-1].ID.String()
		nextCursor = &cursor
	}

	results := make([]*model.Message, 0, len(rows))
	for _, row := range rows {
		msg, convErr := rowToMessage(row)
		if convErr != nil {
			return nil, nil, fmt.Errorf("converting message row: %w", convErr)
		}

		results = append(results, msg)
	}

	return results, nextCursor, nil
}

func (r *messageRepository) Count(ctx context.Context, interchangeID uuid.UUID) (int64, error) {
	query, args, err := r.qb.
		Select("COUNT(*)").
		From(messagesTable).
		Where(sq.Eq{"interchange_id": interchangeID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("building count messages query: %w", err)
	}

	var count int64

	if scanErr := r.db.QueryRow(ctx, query, args...).Scan(&count); scanErr != nil {
		return 0, fmt.Errorf("counting messages: %w", scanErr)
	}

	return count, nil
}

func messageColumns() []string {
	return []string{
		"id", "interchange_id", "processing_version_id",
		"subscription_id", "message_type", "reference",
		"segments", "created_at",
	}
}

func rowToMessage(row messageRow) (*model.Message, error) {
	var segments map[string]any
	if len(row.Segments) > 0 {
		if err := json.Unmarshal(row.Segments, &segments); err != nil {
			return nil, fmt.Errorf("unmarshalling segments: %w", err)
		}
	}

	return &model.Message{
		ID:                  row.ID,
		InterchangeID:       row.InterchangeID,
		ProcessingVersionID: row.ProcessingVersionID,
		SubscriptionID:      row.SubscriptionID,
		MessageType:         row.MessageType,
		MessageReference:    row.Reference,
		Segments:            segments,
		CreatedAt:           row.CreatedAt,
	}, nil
}
