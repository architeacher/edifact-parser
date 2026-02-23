package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
)

// validEDIFACTContent returns a minimal valid EDIFACT-like byte slice for testing.
func validEDIFACTContent() []byte {
	return []byte("UNA:+.? 'UNB+UNOC:3+9900000000001:500+9900000000002:500+260115:1000+REF001'UNH+MSG001+MSCONS:D:11A:UN'UNT+3+MSG001'UNZ+1+REF001'")
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])
}

func TestIngestionService_IngestFile(t *testing.T) {
	t.Parallel()

	validContent := validEDIFACTContent()
	validHash := sha256Hex(validContent)

	existingInterchange := &model.Interchange{
		ID:          uuid.Must(uuid.NewV7()),
		SenderID:    "9900000000001",
		ReceiverID:  "9900000000002",
		Reference:   "REF001",
		ContentHash: validHash,
		Status:      model.StatusCompleted,
		RawContent:  validContent,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	cases := []struct {
		name              string
		fileData          []byte
		contentType       string
		interchangeRepo   *mockInterchangeRepo
		versionRepo       *mockVersionRepo
		messageRepo       *mockMessageRepo
		subscriptionRepo  *mockSubscriptionRepo
		publisher         *mockPublisher
		parser            *mockParser
		txMgr             *mockTxManager
		wantErr           error
		wantErrContains   string
		wantStatus        model.InterchangeStatus
		wantMessageCount  int
		wantSubCount      int
		wantEventCount    int
		wantIdempotent    bool
		wantInterchangeID uuid.UUID
		wantDuplicateErr  bool
	}{
		{
			name:        "valid file ingestion",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:          "9900000000001",
					SenderQualifier:   "500",
					ReceiverID:        "9900000000002",
					ReceiverQualifier: "500",
					Reference:         "REF001",
					PreparedAt:        time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
					ContentHash:       validHash,
					Messages: []ports.ParsedMessage{
						{
							MessageType:      "MSCONS",
							MessageReference: "MSG001",
							SubscriptionID:   "DE0001234567890000000000000123456",
							Segments:         []map[string]any{{"tag": "UNH"}},
						},
					},
					Subscriptions: []string{"DE0001234567890000000000000123456"},
				},
			},
			txMgr:            &mockTxManager{},
			wantStatus:       model.StatusCompleted,
			wantMessageCount: 1,
			wantSubCount:     1,
			wantEventCount:   2,
		},
		{
			name:        "duplicate file returns existing interchange (idempotent)",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashResult: existingInterchange,
			},
			versionRepo:       &mockVersionRepo{},
			messageRepo:       &mockMessageRepo{},
			subscriptionRepo:  &mockSubscriptionRepo{},
			publisher:         &mockPublisher{},
			parser:            &mockParser{},
			txMgr:             &mockTxManager{},
			wantStatus:        model.StatusCompleted,
			wantIdempotent:    true,
			wantInterchangeID: existingInterchange.ID,
		},
		{
			name:             "oversized file rejected",
			fileData:         make([]byte, maxFileSize+1),
			contentType:      "application/edifact",
			interchangeRepo:  &mockInterchangeRepo{},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser:           &mockParser{},
			txMgr:            &mockTxManager{},
			wantErr:          ErrFileTooLarge,
		},
		{
			name:             "empty file rejected",
			fileData:         nil,
			contentType:      "application/edifact",
			interchangeRepo:  &mockInterchangeRepo{},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser:           &mockParser{},
			txMgr:            &mockTxManager{},
			wantErr:          ErrFileEmpty,
		},
		{
			name:        "parse error creates failed interchange",
			fileData:    []byte("INVALID EDIFACT CONTENT"),
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				err: errors.New("EDIFACT parse failed: missing UNB interchange header"),
			},
			txMgr:          &mockTxManager{},
			wantStatus:     model.StatusFailed,
			wantEventCount: 1,
		},
		{
			name:        "transaction begin failure returns error",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:    "9900000000001",
					ReceiverID:  "9900000000002",
					Reference:   "REF001",
					ContentHash: validHash,
				},
			},
			txMgr:           &mockTxManager{beginErr: errors.New("connection lost")},
			wantErrContains: "starting transaction",
		},
		{
			name:        "concurrent duplicate returns existing interchange with sentinel error",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr:    model.ErrInterchangeNotFound,
				createErr:               model.ErrDuplicateInterchange,
				findByContentHashResult: existingInterchange,
				findByContentHashToggle: true,
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:          "9900000000001",
					SenderQualifier:   "500",
					ReceiverID:        "9900000000002",
					ReceiverQualifier: "500",
					Reference:         "REF001",
					ContentHash:       validHash,
				},
			},
			txMgr:             &mockTxManager{},
			wantStatus:        model.StatusCompleted,
			wantIdempotent:    true,
			wantInterchangeID: existingInterchange.ID,
			wantDuplicateErr:  true,
		},
		{
			name:        "interchange create failure rolls back",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
				createErr:            errors.New("unique constraint"),
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:          "9900000000001",
					SenderQualifier:   "500",
					ReceiverID:        "9900000000002",
					ReceiverQualifier: "500",
					Reference:         "REF001",
					ContentHash:       validHash,
				},
			},
			txMgr:           &mockTxManager{},
			wantErrContains: "creating interchange",
		},
		{
			name:        "bulk message create failure rolls back",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
			},
			versionRepo: &mockVersionRepo{},
			messageRepo: &mockMessageRepo{
				bulkCreateErr: errors.New("insert error"),
			},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:    "9900000000001",
					ReceiverID:  "9900000000002",
					Reference:   "REF001",
					ContentHash: validHash,
					Messages: []ports.ParsedMessage{
						{MessageType: "MSCONS", MessageReference: "MSG001"},
					},
				},
			},
			txMgr:           &mockTxManager{},
			wantErrContains: "creating messages",
		},
		{
			name:        "kafka publish failure does not fail ingestion",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher: &mockPublisher{
				err: errors.New("kafka broker unreachable"),
			},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:    "9900000000001",
					ReceiverID:  "9900000000002",
					Reference:   "REF001",
					ContentHash: validHash,
					Messages: []ports.ParsedMessage{
						{MessageType: "MSCONS", MessageReference: "MSG001"},
					},
				},
			},
			txMgr:            &mockTxManager{},
			wantStatus:       model.StatusCompleted,
			wantMessageCount: 1,
			wantEventCount:   2,
		},
		{
			name:        "multiple messages and subscriptions",
			fileData:    validContent,
			contentType: "application/edifact",
			interchangeRepo: &mockInterchangeRepo{
				findByContentHashErr: model.ErrInterchangeNotFound,
			},
			versionRepo:      &mockVersionRepo{},
			messageRepo:      &mockMessageRepo{},
			subscriptionRepo: &mockSubscriptionRepo{},
			publisher:        &mockPublisher{},
			parser: &mockParser{
				result: &ports.ParseResult{
					SenderID:    "9900000000001",
					ReceiverID:  "9900000000002",
					Reference:   "REF001",
					ContentHash: validHash,
					Messages: []ports.ParsedMessage{
						{MessageType: "MSCONS", MessageReference: "MSG001", SubscriptionID: "SUB1"},
						{MessageType: "INVOIC", MessageReference: "MSG002", SubscriptionID: "SUB2"},
						{MessageType: "UTILMD", MessageReference: "MSG003", SubscriptionID: "SUB1"},
					},
					Subscriptions: []string{"SUB1", "SUB2"},
				},
			},
			txMgr:            &mockTxManager{},
			wantStatus:       model.StatusCompleted,
			wantMessageCount: 3,
			wantSubCount:     2,
			wantEventCount:   2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewIngestionService(
				tc.interchangeRepo,
				tc.versionRepo,
				tc.messageRepo,
				tc.subscriptionRepo,
				tc.publisher,
				tc.parser,
				tc.txMgr,
				defaultParserVersion,
			)

			result, err := svc.IngestFile(context.Background(), tc.fileData, tc.contentType)

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, result)

				return
			}

			if tc.wantErrContains != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErrContains)
				require.Nil(t, result)

				return
			}

			if tc.wantDuplicateErr {
				require.ErrorIs(t, err, model.ErrDuplicateInterchange)
				require.NotNil(t, result)
				require.Equal(t, tc.wantInterchangeID, result.InterchangeID)
				require.Equal(t, tc.wantStatus, result.Status)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantStatus, result.Status)

			if tc.wantIdempotent {
				require.Equal(t, tc.wantInterchangeID, result.InterchangeID)

				return
			}

			require.NotEqual(t, uuid.Nil, result.InterchangeID)
			require.Equal(t, tc.wantMessageCount, result.MessageCount)
			require.Equal(t, tc.wantSubCount, result.SubscriptionCount)
			require.Len(t, result.Events, tc.wantEventCount)

			if tc.wantStatus == model.StatusCompleted && tc.wantMessageCount > 0 {
				require.Equal(t, model.EventInterchangeIngested, result.Events[0].EventType)
				require.Equal(t, model.EventVersionCreated, result.Events[1].EventType)
			}

			if tc.wantStatus == model.StatusFailed {
				require.Equal(t, model.EventInterchangeFailed, result.Events[0].EventType)
			}
		})
	}
}

func TestIngestionService_GetInterchange(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32)
	preparedAt := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())

	interchange := &model.Interchange{
		ID:          interchangeID,
		SenderID:    "9900000000001",
		ReceiverID:  "9900000000002",
		Reference:   "REF001",
		PreparedAt:  preparedAt,
		Status:      model.StatusCompleted,
		ContentHash: validHash,
		RawContent:  []byte("UNA:+.? '"),
	}

	version := &model.ProcessingVersion{
		ID:            versionID,
		InterchangeID: interchangeID,
		VersionNumber: 1,
		ParserVersion: "1.0.0",
		FormatVersion: "FV2310",
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
	}

	cases := []struct {
		name             string
		interchangeRepo  *mockInterchangeRepo
		versionRepo      *mockVersionRepo
		messageRepo      *mockMessageRepo
		id               uuid.UUID
		wantErr          string
		wantSenderID     string
		wantVersionNum   int
		wantMessageCount int64
	}{
		{
			name: "found with active version and messages",
			interchangeRepo: &mockInterchangeRepo{
				findByIDResult: interchange,
			},
			versionRepo: &mockVersionRepo{
				findActiveResult: version,
			},
			messageRepo: &mockMessageRepo{
				countResult: 5,
			},
			id:               interchangeID,
			wantSenderID:     "9900000000001",
			wantVersionNum:   1,
			wantMessageCount: 5,
		},
		{
			name: "not found",
			interchangeRepo: &mockInterchangeRepo{
				findByIDErr: model.ErrInterchangeNotFound,
			},
			versionRepo: &mockVersionRepo{},
			messageRepo: &mockMessageRepo{},
			id:          uuid.Must(uuid.NewV7()),
			wantErr:     "interchange not found",
		},
		{
			name: "found without active version",
			interchangeRepo: &mockInterchangeRepo{
				findByIDResult: interchange,
			},
			versionRepo: &mockVersionRepo{
				findActiveErr: model.ErrProcessingVersionNotFound,
			},
			messageRepo: &mockMessageRepo{
				countResult: 0,
			},
			id:           interchangeID,
			wantSenderID: "9900000000001",
		},
		{
			name: "message count matches",
			interchangeRepo: &mockInterchangeRepo{
				findByIDResult: interchange,
			},
			versionRepo: &mockVersionRepo{
				findActiveResult: version,
			},
			messageRepo: &mockMessageRepo{
				countResult: 12,
			},
			id:               interchangeID,
			wantSenderID:     "9900000000001",
			wantVersionNum:   1,
			wantMessageCount: 12,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewIngestionService(
				tc.interchangeRepo,
				tc.versionRepo,
				tc.messageRepo,
				&mockSubscriptionRepo{},
				&mockPublisher{},
				&mockParser{},
				&mockTxManager{},
				defaultParserVersion,
			)

			result, err := svc.GetInterchange(context.Background(), tc.id)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantSenderID, result.SenderID)
			require.Equal(t, tc.wantMessageCount, result.MessageCount)

			if tc.wantVersionNum > 0 {
				require.NotNil(t, result.ActiveVersion)
				require.Equal(t, tc.wantVersionNum, result.ActiveVersion.VersionNumber)
			}
		})
	}
}

func TestIngestionService_GetRawFile(t *testing.T) {
	t.Parallel()

	validHash := strings.Repeat("ab", 32)
	rawContent := []byte("UNA:+.? 'UNB+UNOC:3+9900000000001:14+9900000000002:14+260115:1000+REF001'")
	interchangeID := uuid.Must(uuid.NewV7())

	cases := []struct {
		name            string
		interchangeRepo *mockInterchangeRepo
		id              uuid.UUID
		wantErr         string
		wantContent     []byte
		wantHash        string
	}{
		{
			name: "retrieved verbatim",
			interchangeRepo: &mockInterchangeRepo{
				findByIDResult: &model.Interchange{
					ID:          interchangeID,
					RawContent:  rawContent,
					ContentHash: validHash,
					SenderID:    "S1",
					ReceiverID:  "R1",
					Reference:   "REF",
					PreparedAt:  time.Now().UTC(),
					Status:      model.StatusCompleted,
				},
			},
			id:          interchangeID,
			wantContent: rawContent,
			wantHash:    validHash,
		},
		{
			name: "not found",
			interchangeRepo: &mockInterchangeRepo{
				findByIDErr: model.ErrInterchangeNotFound,
			},
			id:      uuid.Must(uuid.NewV7()),
			wantErr: "interchange not found",
		},
		{
			name: "empty raw content returns error",
			interchangeRepo: &mockInterchangeRepo{
				findByIDResult: &model.Interchange{
					ID:          interchangeID,
					RawContent:  nil,
					ContentHash: validHash,
					SenderID:    "S1",
					ReceiverID:  "R1",
					Reference:   "REF",
					PreparedAt:  time.Now().UTC(),
					Status:      model.StatusCompleted,
				},
			},
			id:      interchangeID,
			wantErr: "raw content",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewIngestionService(
				tc.interchangeRepo,
				&mockVersionRepo{},
				&mockMessageRepo{},
				&mockSubscriptionRepo{},
				&mockPublisher{},
				&mockParser{},
				&mockTxManager{},
				defaultParserVersion,
			)

			result, err := svc.GetRawFile(context.Background(), tc.id)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantContent, result.Content)
			require.Equal(t, tc.wantHash, result.ContentHash)
		})
	}
}

func TestIngestionService_ListMessages(t *testing.T) {
	t.Parallel()

	interchangeID := uuid.Must(uuid.NewV7())
	versionID := uuid.Must(uuid.NewV7())
	nextCursor := "next-page-cursor"

	msg1 := &model.Message{
		ID:                  uuid.Must(uuid.NewV7()),
		InterchangeID:       interchangeID,
		ProcessingVersionID: versionID,
		MessageType:         "MSCONS",
		MessageReference:    "MSG001",
	}

	msg2 := &model.Message{
		ID:                  uuid.Must(uuid.NewV7()),
		InterchangeID:       interchangeID,
		ProcessingVersionID: versionID,
		MessageType:         "INVOIC",
		MessageReference:    "MSG002",
	}

	cases := []struct {
		name         string
		messageRepo  *mockMessageRepo
		filter       *model.MessageFilter
		wantErr      string
		wantCount    int
		wantCursor   bool
		wantMsgTypes []string
	}{
		{
			name: "returns messages with cursor",
			messageRepo: &mockMessageRepo{
				listMessages:   []*model.Message{msg1, msg2},
				listNextCursor: &nextCursor,
			},
			filter: &model.MessageFilter{
				InterchangeID: &interchangeID,
				Limit:         10,
			},
			wantCount:    2,
			wantCursor:   true,
			wantMsgTypes: []string{"MSCONS", "INVOIC"},
		},
		{
			name: "returns empty list",
			messageRepo: &mockMessageRepo{
				listMessages: []*model.Message{},
			},
			filter: &model.MessageFilter{
				InterchangeID: &interchangeID,
			},
			wantCount: 0,
		},
		{
			name: "nil filter defaults to empty filter",
			messageRepo: &mockMessageRepo{
				listMessages: []*model.Message{msg1},
			},
			filter:       nil,
			wantCount:    1,
			wantMsgTypes: []string{"MSCONS"},
		},
		{
			name: "repository error propagated",
			messageRepo: &mockMessageRepo{
				listErr: errors.New("database timeout"),
			},
			filter:  &model.MessageFilter{},
			wantErr: "listing messages",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			svc := NewIngestionService(
				&mockInterchangeRepo{},
				&mockVersionRepo{},
				tc.messageRepo,
				&mockSubscriptionRepo{},
				&mockPublisher{},
				&mockParser{},
				&mockTxManager{},
				defaultParserVersion,
			)

			result, err := svc.ListMessages(context.Background(), tc.filter)

			if tc.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.wantErr)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			require.Len(t, result.Messages, tc.wantCount)

			if tc.wantCursor {
				require.NotNil(t, result.NextCursor)
			}

			for index, wantType := range tc.wantMsgTypes {
				require.Equal(t, wantType, result.Messages[index].MessageType)
			}
		})
	}
}

// --- Mock implementations ---

// mockInterchangeRepo implements ports.InterchangeRepository for tests.
type (
	mockInterchangeRepo struct {
		findByContentHashResult *model.Interchange
		findByContentHashErr    error
		findByContentHashToggle bool // when true, first call returns err, second returns result
		findByContentHashCalls  int
		findByIDResult          *model.Interchange
		findByIDErr             error
		createErr               error
		created                 *model.Interchange
	}
)

func (m *mockInterchangeRepo) Create(_ context.Context, i *model.Interchange) error {
	m.created = i

	return m.createErr
}

func (m *mockInterchangeRepo) FindByContentHash(_ context.Context, _ string) (*model.Interchange, error) {
	m.findByContentHashCalls++

	// Toggle mode: first call returns error (initial dedup check), subsequent calls return result (recovery).
	if m.findByContentHashToggle && m.findByContentHashCalls == 1 {
		return nil, m.findByContentHashErr
	}

	if m.findByContentHashToggle {
		return m.findByContentHashResult, nil
	}

	return m.findByContentHashResult, m.findByContentHashErr
}

func (m *mockInterchangeRepo) FindByID(_ context.Context, _ uuid.UUID) (*model.Interchange, error) {
	return m.findByIDResult, m.findByIDErr
}

func (m *mockInterchangeRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ model.InterchangeStatus, _ *string) error {
	return nil
}

// mockVersionRepo implements ports.ProcessingVersionRepository for tests.
type (
	mockVersionRepo struct {
		createErr        error
		findActiveResult *model.ProcessingVersion
		findActiveErr    error
	}
)

func (m *mockVersionRepo) Create(_ context.Context, _ *model.ProcessingVersion) error {
	return m.createErr
}

func (m *mockVersionRepo) FindActive(_ context.Context, _ uuid.UUID) (*model.ProcessingVersion, error) {
	return m.findActiveResult, m.findActiveErr
}

func (m *mockVersionRepo) FindByInterchangeAndVersion(_ context.Context, _ uuid.UUID, _ int) (*model.ProcessingVersion, error) {
	return nil, nil
}

func (m *mockVersionRepo) DeactivateAll(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockVersionRepo) Activate(_ context.Context, _ uuid.UUID) error {
	return nil
}

// mockMessageRepo implements ports.MessageRepository for tests.
type (
	mockMessageRepo struct {
		bulkCreateErr  error
		countResult    int64
		countErr       error
		listMessages   []*model.Message
		listNextCursor *string
		listErr        error
	}
)

func (m *mockMessageRepo) BulkCreate(_ context.Context, _ []*model.Message) error {
	return m.bulkCreateErr
}

func (m *mockMessageRepo) ListByFilter(_ context.Context, _ *model.MessageFilter) ([]*model.Message, *string, error) {
	return m.listMessages, m.listNextCursor, m.listErr
}

func (m *mockMessageRepo) Count(_ context.Context, _ uuid.UUID) (int64, error) {
	return m.countResult, m.countErr
}

// mockSubscriptionRepo implements ports.SubscriptionRepository for tests.
type (
	mockSubscriptionRepo struct {
		upsertErr    error
		upsertResult *model.Subscription
	}
)

func (m *mockSubscriptionRepo) UpsertByIdentifier(_ context.Context, identifier string) (*model.Subscription, error) {
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}

	if m.upsertResult != nil {
		return m.upsertResult, nil
	}

	return &model.Subscription{
		ID:         uuid.Must(uuid.NewV7()),
		Identifier: identifier,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

func (m *mockSubscriptionRepo) FindByID(_ context.Context, _ uuid.UUID) (*model.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepo) FindAll(_ context.Context) ([]*model.Subscription, error) {
	return nil, nil
}

// mockPublisher implements ports.EventPublisher for tests.
type (
	mockPublisher struct {
		err       error
		published []*model.DomainEvent
	}
)

func (m *mockPublisher) Publish(_ context.Context, events ...*model.DomainEvent) error {
	m.published = append(m.published, events...)

	return m.err
}

func (m *mockPublisher) Close() error {
	return nil
}

// mockParser implements ports.FileParser for tests.
type (
	mockParser struct {
		result *ports.ParseResult
		err    error
	}
)

func (m *mockParser) Parse(_ []byte) (*ports.ParseResult, error) {
	return m.result, m.err
}

// mockTxManager implements ports.TransactionManager for tests.
type (
	mockTxManager struct {
		beginErr error
	}
)

func (m *mockTxManager) Begin(_ context.Context) (ports.Transaction, error) {
	if m.beginErr != nil {
		return nil, m.beginErr
	}

	return &mockTransaction{}, nil
}

// mockTransaction implements ports.Transaction for tests.
type (
	mockTransaction struct {
		committed  bool
		rolledBack bool
	}
)

func (m *mockTransaction) Commit(_ context.Context) error {
	m.committed = true

	return nil
}

func (m *mockTransaction) Rollback(_ context.Context) error {
	m.rolledBack = true

	return nil
}
