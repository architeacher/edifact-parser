package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// EventInterchangeIngested is emitted on successful initial ingestion.
	EventInterchangeIngested EventType = "interchange.ingested"

	// EventInterchangeFailed is emitted when ingestion fails.
	EventInterchangeFailed EventType = "interchange.failed"

	// EventVersionCreated is emitted when a new processing version is committed.
	EventVersionCreated EventType = "version.created"

	// EventVersionActivated is emitted when a version is set as active.
	EventVersionActivated EventType = "version.activated"

	// EventVersionRolledBack is emitted when a rollback operation completes.
	EventVersionRolledBack EventType = "version.rolled_back"
)

// EventType represents the canonical type of a domain event.
type (
	EventType string
)

// DomainEvent is an immutable audit record of a significant system occurrence.
// Published directly to Kafka after DB commit; the domain_events table is audit-only.
type (
	DomainEvent struct {
		ID            uuid.UUID
		InterchangeID *uuid.UUID
		EventType     EventType
		Payload       map[string]any
		CreatedAt     time.Time
	}
)

// NewDomainEvent creates a validated DomainEvent.
func NewDomainEvent(
	interchangeID *uuid.UUID,
	eventType EventType,
	payload map[string]any,
) (*DomainEvent, error) {
	if eventType == "" {
		return nil, fmt.Errorf("creating domain event: %w", ErrEventTypeEmpty)
	}

	if payload == nil {
		payload = map[string]any{}
	}

	id := uuid.Must(uuid.NewV7())
	now := time.Now().UTC()

	payload["event_id"] = id.String()
	payload["event_type"] = string(eventType)
	payload["occurred_at"] = now.Format(time.RFC3339Nano)

	if interchangeID != nil {
		payload["interchange_id"] = interchangeID.String()
	}

	return &DomainEvent{
		ID:            id,
		InterchangeID: interchangeID,
		EventType:     eventType,
		Payload:       payload,
		CreatedAt:     now,
	}, nil
}

// NewIngestedEvent creates an interchange.ingested domain event.
func NewIngestedEvent(interchangeID uuid.UUID, messageCount int, sender, receiver string) (*DomainEvent, error) {
	return NewDomainEvent(
		&interchangeID,
		EventInterchangeIngested,
		map[string]any{
			"message_count": messageCount,
			"sender":        sender,
			"receiver":      receiver,
		},
	)
}

// NewFailedEvent creates an interchange.failed domain event.
func NewFailedEvent(interchangeID uuid.UUID, errMsg string) (*DomainEvent, error) {
	return NewDomainEvent(
		&interchangeID,
		EventInterchangeFailed,
		map[string]any{
			"error": errMsg,
		},
	)
}

// NewVersionCreatedEvent creates a version.created domain event.
func NewVersionCreatedEvent(interchangeID, versionID uuid.UUID, versionNumber int, parserVersion string) (*DomainEvent, error) {
	return NewDomainEvent(
		&interchangeID,
		EventVersionCreated,
		map[string]any{
			"version_id":     versionID.String(),
			"version_number": versionNumber,
			"parser_version": parserVersion,
		},
	)
}

// NewVersionActivatedEvent creates a version.activated domain event.
func NewVersionActivatedEvent(interchangeID, versionID uuid.UUID, versionNumber int) (*DomainEvent, error) {
	return NewDomainEvent(
		&interchangeID,
		EventVersionActivated,
		map[string]any{
			"version_id":     versionID.String(),
			"version_number": versionNumber,
		},
	)
}

// NewVersionRolledBackEvent creates a version.rolled_back domain event.
func NewVersionRolledBackEvent(interchangeID uuid.UUID, fromVersion, toVersion int) (*DomainEvent, error) {
	return NewDomainEvent(
		&interchangeID,
		EventVersionRolledBack,
		map[string]any{
			"from_version": fromVersion,
			"to_version":   toVersion,
		},
	)
}
