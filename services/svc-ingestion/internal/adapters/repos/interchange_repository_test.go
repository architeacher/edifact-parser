package repos_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/repos"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
)

func TestInterchangeRepository_CreateAndFindByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewInterchangeRepository(pool)
	ctx := context.Background()

	validHash := strings.Repeat("ab", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	interchange, err := model.NewInterchange(
		"9900000000001", "14",
		"9900000000002", "14",
		preparedAt, "REF001",
		[]byte("UNA:+.? 'UNB+..."),
		validHash,
	)
	require.NoError(t, err)

	err = repo.Create(ctx, interchange)
	require.NoError(t, err)

	got, err := repo.FindByID(ctx, interchange.ID)
	require.NoError(t, err)
	require.Equal(t, interchange.ID, got.ID)
	require.Equal(t, interchange.SenderID, got.SenderID)
	require.Equal(t, interchange.SenderQualifier, got.SenderQualifier)
	require.Equal(t, interchange.ReceiverID, got.ReceiverID)
	require.Equal(t, interchange.ReceiverQualifier, got.ReceiverQualifier)
	require.Equal(t, interchange.Reference, got.Reference)
	require.Equal(t, interchange.RawContent, got.RawContent)
	require.Equal(t, interchange.ContentHash, got.ContentHash)
	require.Equal(t, model.StatusCompleted, got.Status)
	require.Nil(t, got.ErrorDetail)
}

func TestInterchangeRepository_FindByContentHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewInterchangeRepository(pool)
	ctx := context.Background()

	validHash := strings.Repeat("cd", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	interchange, err := model.NewInterchange(
		"SENDER01", "14", "RECEIVER01", "14",
		preparedAt, "REF002",
		[]byte("UNA:+.? 'UNB+test"),
		validHash,
	)
	require.NoError(t, err)

	err = repo.Create(ctx, interchange)
	require.NoError(t, err)

	cases := []struct {
		name    string
		hash    string
		wantNil bool
		wantErr error
	}{
		{
			name: "existing hash returns interchange",
			hash: validHash,
		},
		{
			name:    "non-existent hash returns not found",
			hash:    strings.Repeat("ff", 32),
			wantNil: true,
			wantErr: model.ErrInterchangeNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, findErr := repo.FindByContentHash(ctx, tc.hash)

			if tc.wantErr != nil {
				require.ErrorIs(t, findErr, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, findErr)
			require.NotNil(t, got)
			require.Equal(t, interchange.ID, got.ID)
		})
	}
}

func TestInterchangeRepository_DuplicateContentHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewInterchangeRepository(pool)
	ctx := context.Background()

	validHash := strings.Repeat("de", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	first, err := model.NewInterchange(
		"SENDER01", "14", "RECEIVER01", "14",
		preparedAt, "REF003",
		[]byte("content-one"),
		validHash,
	)
	require.NoError(t, err)

	err = repo.Create(ctx, first)
	require.NoError(t, err)

	second, err := model.NewInterchange(
		"SENDER02", "14", "RECEIVER02", "14",
		preparedAt, "REF004",
		[]byte("content-two"),
		validHash,
	)
	require.NoError(t, err)

	err = repo.Create(ctx, second)
	require.ErrorIs(t, err, model.ErrDuplicateInterchange)
}

func TestInterchangeRepository_UpdateStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewInterchangeRepository(pool)
	ctx := context.Background()

	validHash := strings.Repeat("ef", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	interchange, err := model.NewInterchange(
		"SENDER01", "14", "RECEIVER01", "14",
		preparedAt, "REF005",
		[]byte("UNA:+.? 'UNB+update-test"),
		validHash,
	)
	require.NoError(t, err)

	err = repo.Create(ctx, interchange)
	require.NoError(t, err)

	cases := []struct {
		name    string
		id      uuid.UUID
		status  model.InterchangeStatus
		detail  *string
		wantErr error
	}{
		{
			name:   "update to failed with detail",
			id:     interchange.ID,
			status: model.StatusFailed,
			detail: ptr("parse error at segment 42"),
		},
		{
			name:   "update back to completed",
			id:     interchange.ID,
			status: model.StatusCompleted,
			detail: nil,
		},
		{
			name:    "non-existent ID returns not found",
			id:      uuid.Must(uuid.NewV7()),
			status:  model.StatusFailed,
			detail:  ptr("some error"),
			wantErr: model.ErrInterchangeNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			updateErr := repo.UpdateStatus(ctx, tc.id, tc.status, tc.detail)

			if tc.wantErr != nil {
				require.ErrorIs(t, updateErr, tc.wantErr)

				return
			}

			require.NoError(t, updateErr)

			got, findErr := repo.FindByID(ctx, tc.id)
			require.NoError(t, findErr)
			require.Equal(t, tc.status, got.Status)

			if tc.detail != nil {
				require.NotNil(t, got.ErrorDetail)
				require.Equal(t, *tc.detail, *got.ErrorDetail)
			} else {
				require.Nil(t, got.ErrorDetail)
			}
		})
	}
}

func TestInterchangeRepository_FindByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewInterchangeRepository(pool)
	ctx := context.Background()

	got, err := repo.FindByID(ctx, uuid.Must(uuid.NewV7()))
	require.ErrorIs(t, err, model.ErrInterchangeNotFound)
	require.Nil(t, got)
}
