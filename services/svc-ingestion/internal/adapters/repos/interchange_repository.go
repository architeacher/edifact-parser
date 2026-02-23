package repos

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

const (
	interchangesTable = "interchanges"
)

type (
	interchangeRow struct {
		ID                uuid.UUID              `db:"id"`
		SenderID          string                 `db:"sender_id"`
		SenderQualifier   string                 `db:"sender_qualifier"`
		ReceiverID        string                 `db:"receiver_id"`
		ReceiverQualifier string                 `db:"receiver_qualifier"`
		PreparedAt        time.Time              `db:"prepared_at"`
		Reference         string                 `db:"reference"`
		RawContent        []byte                 `db:"raw_content"`
		ContentHash       string                 `db:"content_hash"`
		Status            model.InterchangeStatus `db:"status"`
		ErrorDetail       *string                `db:"error_detail"`
		CreatedAt         time.Time              `db:"created_at"`
		UpdatedAt         time.Time              `db:"updated_at"`
	}

	interchangeRepository struct {
		db DBTX
		qb sq.StatementBuilderType
	}
)

// NewInterchangeRepository creates a new InterchangeRepository backed by PostgreSQL.
func NewInterchangeRepository(db DBTX) ports.InterchangeRepository {
	return &interchangeRepository{
		db: db,
		qb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *interchangeRepository) Create(ctx context.Context, interchange *model.Interchange) error {
	query, args, err := r.qb.
		Insert(interchangesTable).
		Columns(
			"id", "sender_id", "sender_qualifier",
			"receiver_id", "receiver_qualifier", "prepared_at",
			"reference", "raw_content", "content_hash",
			"status", "error_detail", "created_at", "updated_at",
		).
		Values(
			interchange.ID, interchange.SenderID, interchange.SenderQualifier,
			interchange.ReceiverID, interchange.ReceiverQualifier, interchange.PreparedAt,
			interchange.Reference, interchange.RawContent, interchange.ContentHash,
			interchange.Status, interchange.ErrorDetail, interchange.CreatedAt, interchange.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("building insert interchange query: %w", err)
	}

	if _, execErr := r.db.Exec(ctx, query, args...); execErr != nil {
		var pgErr *pgconn.PgError
		if errors.As(execErr, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("inserting interchange: %w", model.ErrDuplicateInterchange)
		}

		return fmt.Errorf("inserting interchange: %w", execErr)
	}

	return nil
}

func (r *interchangeRepository) FindByContentHash(ctx context.Context, hash string) (*model.Interchange, error) {
	query, args, err := r.qb.
		Select(interchangeColumns()...).
		From(interchangesTable).
		Where(sq.Eq{"content_hash": hash}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building find-by-hash query: %w", err)
	}

	return r.scanOne(ctx, query, args)
}

func (r *interchangeRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Interchange, error) {
	query, args, err := r.qb.
		Select(interchangeColumns()...).
		From(interchangesTable).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building find-by-id query: %w", err)
	}

	return r.scanOne(ctx, query, args)
}

func (r *interchangeRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.InterchangeStatus, detail *string) error {
	query, args, err := r.qb.
		Update(interchangesTable).
		Set("status", status).
		Set("error_detail", detail).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("building update status query: %w", err)
	}

	tag, execErr := r.db.Exec(ctx, query, args...)
	if execErr != nil {
		return fmt.Errorf("updating interchange status: %w", execErr)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("updating interchange status: %w", model.ErrInterchangeNotFound)
	}

	return nil
}

func (r *interchangeRepository) scanOne(ctx context.Context, query string, args []any) (*model.Interchange, error) {
	var row interchangeRow

	if err := pgxscan.Get(ctx, r.db, &row, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("scanning interchange: %w", model.ErrInterchangeNotFound)
		}

		return nil, fmt.Errorf("scanning interchange: %w", err)
	}

	return rowToInterchange(row), nil
}

func interchangeColumns() []string {
	return []string{
		"id", "sender_id", "sender_qualifier",
		"receiver_id", "receiver_qualifier", "prepared_at",
		"reference", "raw_content", "content_hash",
		"status", "error_detail", "created_at", "updated_at",
	}
}

func rowToInterchange(row interchangeRow) *model.Interchange {
	return &model.Interchange{
		ID:                row.ID,
		SenderID:          row.SenderID,
		SenderQualifier:   row.SenderQualifier,
		ReceiverID:        row.ReceiverID,
		ReceiverQualifier: row.ReceiverQualifier,
		PreparedAt:        row.PreparedAt,
		Reference:         row.Reference,
		RawContent:        row.RawContent,
		ContentHash:       row.ContentHash,
		Status:            row.Status,
		ErrorDetail:       row.ErrorDetail,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
