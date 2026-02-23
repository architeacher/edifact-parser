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

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

const (
	processingVersionsTable = "processing_versions"
)

type (
	processingVersionRow struct {
		ID            uuid.UUID `db:"id"`
		InterchangeID uuid.UUID `db:"interchange_id"`
		VersionNumber int       `db:"version_number"`
		ParserVersion string    `db:"parser_version"`
		IsActive      bool      `db:"is_active"`
		CreatedAt     time.Time `db:"created_at"`
	}

	processingVersionRepository struct {
		db DBTX
		qb sq.StatementBuilderType
	}
)

// NewProcessingVersionRepository creates a new ProcessingVersionRepository backed by PostgreSQL.
func NewProcessingVersionRepository(db DBTX) ports.ProcessingVersionRepository {
	return &processingVersionRepository{
		db: db,
		qb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *processingVersionRepository) Create(ctx context.Context, version *model.ProcessingVersion) error {
	query, args, err := r.qb.
		Insert(processingVersionsTable).
		Columns("id", "interchange_id", "version_number", "parser_version", "is_active", "created_at").
		Values(
			version.ID, version.InterchangeID, version.VersionNumber,
			version.ParserVersion, version.IsActive, version.CreatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("building insert processing version query: %w", err)
	}

	if _, execErr := r.db.Exec(ctx, query, args...); execErr != nil {
		return fmt.Errorf("inserting processing version: %w", execErr)
	}

	return nil
}

func (r *processingVersionRepository) FindActive(ctx context.Context, interchangeID uuid.UUID) (*model.ProcessingVersion, error) {
	query, args, err := r.qb.
		Select(processingVersionColumns()...).
		From(processingVersionsTable).
		Where(sq.Eq{"interchange_id": interchangeID, "is_active": true}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building find active version query: %w", err)
	}

	return r.scanOne(ctx, query, args)
}

func (r *processingVersionRepository) FindByInterchangeAndVersion(ctx context.Context, interchangeID uuid.UUID, versionNumber int) (*model.ProcessingVersion, error) {
	query, args, err := r.qb.
		Select(processingVersionColumns()...).
		From(processingVersionsTable).
		Where(sq.Eq{"interchange_id": interchangeID, "version_number": versionNumber}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building find by interchange and version query: %w", err)
	}

	return r.scanOne(ctx, query, args)
}

func (r *processingVersionRepository) DeactivateAll(ctx context.Context, interchangeID uuid.UUID) error {
	query, args, err := r.qb.
		Update(processingVersionsTable).
		Set("is_active", false).
		Where(sq.Eq{"interchange_id": interchangeID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("building deactivate all query: %w", err)
	}

	if _, execErr := r.db.Exec(ctx, query, args...); execErr != nil {
		return fmt.Errorf("deactivating all processing versions: %w", execErr)
	}

	return nil
}

func (r *processingVersionRepository) Activate(ctx context.Context, versionID uuid.UUID) error {
	// Atomic: first deactivate siblings, then activate the target.
	// The EXCLUDE constraint ensures at most one active version per interchange.
	// We get the interchange_id first, then deactivate all, then activate target.
	deactivateSQL := `UPDATE processing_versions SET is_active = FALSE
		WHERE interchange_id = (SELECT interchange_id FROM processing_versions WHERE id = $1)`

	if _, err := r.db.Exec(ctx, deactivateSQL, versionID); err != nil {
		return fmt.Errorf("deactivating sibling versions: %w", err)
	}

	activateSQL := `UPDATE processing_versions SET is_active = TRUE WHERE id = $1`

	tag, err := r.db.Exec(ctx, activateSQL, versionID)
	if err != nil {
		return fmt.Errorf("activating processing version: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("activating processing version: %w", model.ErrProcessingVersionNotFound)
	}

	return nil
}

func (r *processingVersionRepository) scanOne(ctx context.Context, query string, args []any) (*model.ProcessingVersion, error) {
	var row processingVersionRow

	if err := pgxscan.Get(ctx, r.db, &row, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("scanning processing version: %w", model.ErrProcessingVersionNotFound)
		}

		return nil, fmt.Errorf("scanning processing version: %w", err)
	}

	return rowToProcessingVersion(row), nil
}

func processingVersionColumns() []string {
	return []string{
		"id", "interchange_id", "version_number",
		"parser_version", "is_active", "created_at",
	}
}

func rowToProcessingVersion(row processingVersionRow) *model.ProcessingVersion {
	return &model.ProcessingVersion{
		ID:            row.ID,
		InterchangeID: row.InterchangeID,
		VersionNumber: row.VersionNumber,
		ParserVersion: row.ParserVersion,
		IsActive:      row.IsActive,
		CreatedAt:     row.CreatedAt,
	}
}
