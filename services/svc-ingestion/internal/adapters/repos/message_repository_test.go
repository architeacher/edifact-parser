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

func TestMessageRepository_BulkCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewMessageRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("f1", 32)
	interchangeID := insertTestInterchange(t, pool, hash)
	versionID := insertTestProcessingVersion(t, pool, interchangeID, 1, true)

	messages := make([]*model.Message, 3)
	for index := range messages {
		msg, err := model.NewMessage(
			interchangeID, versionID,
			nil,
			"MSCONS", "REF"+strings.Repeat("0", index),
			map[string]any{"tag": "BGM", "index": index},
		)
		require.NoError(t, err)

		messages[index] = msg
	}

	err := repo.BulkCreate(ctx, messages)
	require.NoError(t, err)

	count, err := repo.Count(ctx, interchangeID)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}

func TestMessageRepository_BulkCreate_EmptySlice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewMessageRepository(pool)
	ctx := context.Background()

	err := repo.BulkCreate(ctx, nil)
	require.NoError(t, err)

	err = repo.BulkCreate(ctx, []*model.Message{})
	require.NoError(t, err)
}

func TestMessageRepository_ListByFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewMessageRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("f2", 32)
	interchangeID := insertTestInterchange(t, pool, hash)
	versionID := insertTestProcessingVersion(t, pool, interchangeID, 1, true)
	subID := insertTestSubscription(t, pool, "DE0000801043700000000000011145670")

	// Insert mixed message types.
	mscons, err := model.NewMessage(interchangeID, versionID, &subID, "MSCONS", "REF-M1", map[string]any{"type": "MSCONS"})
	require.NoError(t, err)

	invoic, err := model.NewMessage(interchangeID, versionID, nil, "INVOIC", "REF-I1", map[string]any{"type": "INVOIC"})
	require.NoError(t, err)

	utilmd, err := model.NewMessage(interchangeID, versionID, &subID, "UTILMD", "REF-U1", map[string]any{"type": "UTILMD"})
	require.NoError(t, err)

	err = repo.BulkCreate(ctx, []*model.Message{mscons, invoic, utilmd})
	require.NoError(t, err)

	cases := []struct {
		name      string
		filter    model.MessageFilter
		wantCount int
	}{
		{
			name:      "all messages for interchange",
			filter:    model.MessageFilter{InterchangeID: &interchangeID, Limit: 10},
			wantCount: 3,
		},
		{
			name:      "filter by message type MSCONS",
			filter:    model.MessageFilter{InterchangeID: &interchangeID, MessageType: ptr("MSCONS"), Limit: 10},
			wantCount: 1,
		},
		{
			name:      "filter by subscription",
			filter:    model.MessageFilter{SubscriptionID: &subID, Limit: 10},
			wantCount: 2,
		},
		{
			name:      "filter by processing version",
			filter:    model.MessageFilter{ProcessingVersionID: &versionID, Limit: 10},
			wantCount: 3,
		},
		{
			name:      "non-existent interchange returns empty",
			filter:    model.MessageFilter{InterchangeID: ptr(uuid.Must(uuid.NewV7())), Limit: 10},
			wantCount: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			results, _, filterErr := repo.ListByFilter(ctx, &tc.filter)
			require.NoError(t, filterErr)
			require.Len(t, results, tc.wantCount)
		})
	}
}

func TestMessageRepository_ListByFilter_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewMessageRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("f3", 32)
	interchangeID := insertTestInterchange(t, pool, hash)
	versionID := insertTestProcessingVersion(t, pool, interchangeID, 1, true)

	// Insert 5 messages.
	messages := make([]*model.Message, 5)
	for index := range messages {
		msg, err := model.NewMessage(
			interchangeID, versionID, nil,
			"MSCONS", "REF-P"+strings.Repeat("0", index+1),
			map[string]any{"index": index},
		)
		require.NoError(t, err)

		messages[index] = msg
	}

	err := repo.BulkCreate(ctx, messages)
	require.NoError(t, err)

	// First page: limit 2.
	filter := &model.MessageFilter{InterchangeID: &interchangeID, Limit: 2}
	page1, cursor1, err := repo.ListByFilter(ctx, filter)
	require.NoError(t, err)
	require.Len(t, page1, 2)
	require.NotNil(t, cursor1)

	// Second page using cursor.
	filter2 := &model.MessageFilter{InterchangeID: &interchangeID, Limit: 2, Cursor: cursor1}
	page2, cursor2, err := repo.ListByFilter(ctx, filter2)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	require.NotNil(t, cursor2)

	// Third page: only 1 remaining.
	filter3 := &model.MessageFilter{InterchangeID: &interchangeID, Limit: 2, Cursor: cursor2}
	page3, cursor3, err := repo.ListByFilter(ctx, filter3)
	require.NoError(t, err)
	require.Len(t, page3, 1)
	require.Nil(t, cursor3)

	// No overlap between pages.
	allIDs := make(map[uuid.UUID]struct{})
	for _, msg := range page1 {
		allIDs[msg.ID] = struct{}{}
	}
	for _, msg := range page2 {
		allIDs[msg.ID] = struct{}{}
	}
	for _, msg := range page3 {
		allIDs[msg.ID] = struct{}{}
	}
	require.Len(t, allIDs, 5)
}

func TestMessageRepository_Count(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewMessageRepository(pool)
	ctx := context.Background()

	// Count for non-existent interchange should be 0.
	count, err := repo.Count(ctx, uuid.Must(uuid.NewV7()))
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestMessageRepository_BulkCreate_WithSegments(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Parallel()

	pool := newTestPool(t)
	repo := repos.NewMessageRepository(pool)
	ctx := context.Background()

	hash := strings.Repeat("f4", 32)
	interchangeID := insertTestInterchange(t, pool, hash)
	versionID := insertTestProcessingVersion(t, pool, interchangeID, 1, true)

	segments := map[string]any{
		"unb": map[string]any{
			"sender":   map[string]any{"id": "9900000000001", "qualifier": "14"},
			"receiver": map[string]any{"id": "9900000000002", "qualifier": "14"},
		},
		"unh": map[string]any{
			"reference":    "1",
			"message_type": "MSCONS",
		},
		"loc_172": "DE0000801043700000000000011145670",
	}

	msg, err := model.NewMessage(interchangeID, versionID, nil, "MSCONS", "REF-SEG", segments)
	require.NoError(t, err)

	err = repo.BulkCreate(ctx, []*model.Message{msg})
	require.NoError(t, err)

	filter := &model.MessageFilter{InterchangeID: &interchangeID, Limit: 10}
	results, _, err := repo.ListByFilter(ctx, filter)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "DE0000801043700000000000011145670", results[0].Segments["loc_172"])
}
