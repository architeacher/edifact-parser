package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
)

// InterchangeRepository defines the persistence contract for interchanges.
type (
	InterchangeRepository interface {
		Create(ctx context.Context, interchange *model.Interchange) error
		FindByContentHash(ctx context.Context, hash string) (*model.Interchange, error)
		FindByID(ctx context.Context, id uuid.UUID) (*model.Interchange, error)
		UpdateStatus(ctx context.Context, id uuid.UUID, status model.InterchangeStatus, detail *string) error
	}
)

// ProcessingVersionRepository defines the persistence contract for processing versions.
type (
	ProcessingVersionRepository interface {
		Create(ctx context.Context, version *model.ProcessingVersion) error
		FindActive(ctx context.Context, interchangeID uuid.UUID) (*model.ProcessingVersion, error)
		FindByInterchangeAndVersion(ctx context.Context, interchangeID uuid.UUID, versionNumber int) (*model.ProcessingVersion, error)
		DeactivateAll(ctx context.Context, interchangeID uuid.UUID) error
		Activate(ctx context.Context, versionID uuid.UUID) error
	}
)

// MessageRepository defines the persistence contract for messages.
type (
	MessageRepository interface {
		BulkCreate(ctx context.Context, messages []*model.Message) error
		ListByFilter(ctx context.Context, filter *model.MessageFilter) ([]*model.Message, *string, error)
		Count(ctx context.Context, interchangeID uuid.UUID) (int64, error)
	}
)

// SubscriptionRepository defines the persistence contract for subscriptions.
type (
	SubscriptionRepository interface {
		UpsertByIdentifier(ctx context.Context, identifier string) (*model.Subscription, error)
		FindByID(ctx context.Context, id uuid.UUID) (*model.Subscription, error)
		FindAll(ctx context.Context) ([]*model.Subscription, error)
	}
)

// HealthChecker verifies connectivity to an external dependency.
type (
	HealthChecker interface {
		Ping(ctx context.Context) error
	}
)
