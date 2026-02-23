package ports

import (
	"context"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
)

// EventPublisher emits domain events to an external message broker.
type (
	EventPublisher interface {
		// Publish sends one or more domain events. Implementations should
		// batch writes where possible and return the first error encountered.
		Publish(ctx context.Context, events ...*model.DomainEvent) error

		// Close gracefully shuts down the publisher, flushing buffered messages.
		Close() error
	}
)
