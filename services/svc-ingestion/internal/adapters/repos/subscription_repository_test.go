package repos_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/repos"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
)

func TestSubscriptionRepository_UpsertByIdentifier(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewSubscriptionRepository(pool)
	ctx := context.Background()

	identifier := "DE0000801043700000000000011145670"

	// First upsert creates.
	first, err := repo.UpsertByIdentifier(ctx, identifier)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, first.ID)
	require.Equal(t, identifier, first.Identifier)

	// Second upsert returns same record (idempotent).
	second, err := repo.UpsertByIdentifier(ctx, identifier)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.Identifier, second.Identifier)
}

func TestSubscriptionRepository_UpsertByIdentifier_EmptyIdentifier(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewSubscriptionRepository(pool)
	ctx := context.Background()

	_, err := repo.UpsertByIdentifier(ctx, "")
	require.Error(t, err)
	require.ErrorIs(t, err, model.ErrEmptyIdentifier)
}

func TestSubscriptionRepository_FindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewSubscriptionRepository(pool)
	ctx := context.Background()

	identifier := "DE0000801043700000000000099887766"

	created, err := repo.UpsertByIdentifier(ctx, identifier)
	require.NoError(t, err)

	cases := []struct {
		name    string
		id      uuid.UUID
		wantErr error
	}{
		{
			name: "existing subscription",
			id:   created.ID,
		},
		{
			name:    "non-existent ID",
			id:      uuid.Must(uuid.NewV7()),
			wantErr: model.ErrSubscriptionNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, findErr := repo.FindByID(ctx, tc.id)

			if tc.wantErr != nil {
				require.ErrorIs(t, findErr, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, findErr)
			require.Equal(t, created.ID, got.ID)
			require.Equal(t, identifier, got.Identifier)
		})
	}
}

func TestSubscriptionRepository_FindAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewSubscriptionRepository(pool)
	ctx := context.Background()

	// Empty initially.
	all, err := repo.FindAll(ctx)
	require.NoError(t, err)
	require.Empty(t, all)

	// Insert two subscriptions.
	_, err = repo.UpsertByIdentifier(ctx, "DE0000000000000000000000000000001")
	require.NoError(t, err)

	_, err = repo.UpsertByIdentifier(ctx, "DE0000000000000000000000000000002")
	require.NoError(t, err)

	all, err = repo.FindAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
}

func TestSubscriptionRepository_UpsertMultipleDistinct(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewSubscriptionRepository(pool)
	ctx := context.Background()

	sub1, err := repo.UpsertByIdentifier(ctx, "DE1111111111111111111111111111111")
	require.NoError(t, err)

	sub2, err := repo.UpsertByIdentifier(ctx, "DE2222222222222222222222222222222")
	require.NoError(t, err)

	require.NotEqual(t, sub1.ID, sub2.ID)
}
