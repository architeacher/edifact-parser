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
	subscriptionsTable = "subscriptions"
)

type (
	subscriptionRow struct {
		ID         uuid.UUID `db:"id"`
		Identifier string    `db:"identifier"`
		CreatedAt  time.Time `db:"created_at"`
	}

	subscriptionRepository struct {
		db DBTX
		qb sq.StatementBuilderType
	}
)

// NewSubscriptionRepository creates a new SubscriptionRepository backed by PostgreSQL.
func NewSubscriptionRepository(db DBTX) ports.SubscriptionRepository {
	return &subscriptionRepository{
		db: db,
		qb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *subscriptionRepository) UpsertByIdentifier(ctx context.Context, identifier string) (*model.Subscription, error) {
	sub, err := model.NewSubscription(identifier)
	if err != nil {
		return nil, fmt.Errorf("creating subscription model: %w", err)
	}

	// INSERT ... ON CONFLICT DO NOTHING, then SELECT to get the existing or newly created row.
	insertSQL := `INSERT INTO subscriptions (id, identifier, created_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (identifier) DO NOTHING`

	if _, execErr := r.db.Exec(ctx, insertSQL, sub.ID, sub.Identifier, sub.CreatedAt); execErr != nil {
		return nil, fmt.Errorf("upserting subscription: %w", execErr)
	}

	// Always SELECT to return the canonical row (whether just inserted or pre-existing).
	query, args, err := r.qb.
		Select("id", "identifier", "created_at").
		From(subscriptionsTable).
		Where(sq.Eq{"identifier": identifier}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building select subscription query: %w", err)
	}

	return r.scanOne(ctx, query, args)
}

func (r *subscriptionRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error) {
	query, args, err := r.qb.
		Select("id", "identifier", "created_at").
		From(subscriptionsTable).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building find subscription by id query: %w", err)
	}

	return r.scanOne(ctx, query, args)
}

func (r *subscriptionRepository) FindAll(ctx context.Context) ([]*model.Subscription, error) {
	query, args, err := r.qb.
		Select("id", "identifier", "created_at").
		From(subscriptionsTable).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("building find all subscriptions query: %w", err)
	}

	var rows []subscriptionRow
	if scanErr := pgxscan.Select(ctx, r.db, &rows, query, args...); scanErr != nil {
		return nil, fmt.Errorf("listing subscriptions: %w", scanErr)
	}

	results := make([]*model.Subscription, 0, len(rows))
	for _, row := range rows {
		results = append(results, rowToSubscription(row))
	}

	return results, nil
}

func (r *subscriptionRepository) scanOne(ctx context.Context, query string, args []any) (*model.Subscription, error) {
	var row subscriptionRow

	if err := pgxscan.Get(ctx, r.db, &row, query, args...); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("scanning subscription: %w", model.ErrSubscriptionNotFound)
		}

		return nil, fmt.Errorf("scanning subscription: %w", err)
	}

	return rowToSubscription(row), nil
}

func rowToSubscription(row subscriptionRow) *model.Subscription {
	return &model.Subscription{
		ID:         row.ID,
		Identifier: row.Identifier,
		CreatedAt:  row.CreatedAt,
	}
}
