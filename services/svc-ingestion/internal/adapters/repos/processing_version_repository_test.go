package repos_test

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/adapters/repos"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
)

func TestProcessingVersionRepository_CreateAndFindActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewProcessingVersionRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("a1", 32)
	interchangeID := insertTestInterchange(t, pool, hash)

	version, err := model.NewProcessingVersion(interchangeID, 1, "v1.0.0", "D:96A:UN:EAN008")
	require.NoError(t, err)

	err = repo.Create(ctx, version)
	require.NoError(t, err)

	got, err := repo.FindActive(ctx, interchangeID)
	require.NoError(t, err)
	require.Equal(t, version.ID, got.ID)
	require.Equal(t, interchangeID, got.InterchangeID)
	require.Equal(t, 1, got.VersionNumber)
	require.Equal(t, "v1.0.0", got.ParserVersion)
	require.True(t, got.IsActive)
}

func TestProcessingVersionRepository_FindByInterchangeAndVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewProcessingVersionRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("b2", 32)
	interchangeID := insertTestInterchange(t, pool, hash)

	version, err := model.NewProcessingVersion(interchangeID, 1, "v1.0.0", "")
	require.NoError(t, err)

	err = repo.Create(ctx, version)
	require.NoError(t, err)

	cases := []struct {
		name          string
		interchangeID uuid.UUID
		versionNumber int
		wantErr       error
	}{
		{
			name:          "existing version",
			interchangeID: interchangeID,
			versionNumber: 1,
		},
		{
			name:          "non-existent version number",
			interchangeID: interchangeID,
			versionNumber: 99,
			wantErr:       model.ErrProcessingVersionNotFound,
		},
		{
			name:          "non-existent interchange",
			interchangeID: uuid.Must(uuid.NewV7()),
			versionNumber: 1,
			wantErr:       model.ErrProcessingVersionNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, findErr := repo.FindByInterchangeAndVersion(ctx, tc.interchangeID, tc.versionNumber)

			if tc.wantErr != nil {
				require.ErrorIs(t, findErr, tc.wantErr)
				require.Nil(t, got)

				return
			}

			require.NoError(t, findErr)
			require.Equal(t, version.ID, got.ID)
		})
	}
}

func TestProcessingVersionRepository_DeactivateAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewProcessingVersionRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("c3", 32)
	interchangeID := insertTestInterchange(t, pool, hash)

	v1, err := model.NewProcessingVersion(interchangeID, 1, "v1.0.0", "")
	require.NoError(t, err)

	err = repo.Create(ctx, v1)
	require.NoError(t, err)

	// Deactivate v1 before creating v2 (EXCLUDE constraint).
	err = repo.DeactivateAll(ctx, interchangeID)
	require.NoError(t, err)

	// Verify v1 is now inactive.
	got, err := repo.FindByInterchangeAndVersion(ctx, interchangeID, 1)
	require.NoError(t, err)
	require.False(t, got.IsActive)

	// FindActive should return not found.
	_, err = repo.FindActive(ctx, interchangeID)
	require.ErrorIs(t, err, model.ErrProcessingVersionNotFound)
}

func TestProcessingVersionRepository_Activate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewProcessingVersionRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("d4", 32)
	interchangeID := insertTestInterchange(t, pool, hash)

	// Create v1 (active).
	v1, err := model.NewProcessingVersion(interchangeID, 1, "v1.0.0", "")
	require.NoError(t, err)

	err = repo.Create(ctx, v1)
	require.NoError(t, err)

	// Deactivate v1, then create v2 (active).
	err = repo.DeactivateAll(ctx, interchangeID)
	require.NoError(t, err)

	v2, err := model.NewProcessingVersion(interchangeID, 2, "v2.0.0", "")
	require.NoError(t, err)

	err = repo.Create(ctx, v2)
	require.NoError(t, err)

	// Now activate v1 — first deactivate all, then activate v1.
	err = repo.DeactivateAll(ctx, interchangeID)
	require.NoError(t, err)

	err = repo.Activate(ctx, v1.ID)
	require.NoError(t, err)

	// v1 should be active.
	active, err := repo.FindActive(ctx, interchangeID)
	require.NoError(t, err)
	require.Equal(t, v1.ID, active.ID)
	require.True(t, active.IsActive)

	// v2 should be inactive.
	got, err := repo.FindByInterchangeAndVersion(ctx, interchangeID, 2)
	require.NoError(t, err)
	require.False(t, got.IsActive)
}

func TestProcessingVersionRepository_Activate_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewProcessingVersionRepository(pool)
	ctx := context.Background()

	err := repo.Activate(ctx, uuid.Must(uuid.NewV7()))
	require.ErrorIs(t, err, model.ErrProcessingVersionNotFound)
}

func TestProcessingVersionRepository_UniqueConstraint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewProcessingVersionRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("e5", 32)
	interchangeID := insertTestInterchange(t, pool, hash)

	v1, err := model.NewProcessingVersion(interchangeID, 1, "v1.0.0", "")
	require.NoError(t, err)

	err = repo.Create(ctx, v1)
	require.NoError(t, err)

	// Duplicate version number should fail.
	v1dup, err := model.NewProcessingVersion(interchangeID, 1, "v1.1.0", "")
	require.NoError(t, err)

	// Deactivate first to avoid EXCLUDE constraint on is_active.
	err = repo.DeactivateAll(ctx, interchangeID)
	require.NoError(t, err)

	err = repo.Create(ctx, v1dup)
	require.Error(t, err)
	require.Contains(t, err.Error(), "inserting processing version")
}
