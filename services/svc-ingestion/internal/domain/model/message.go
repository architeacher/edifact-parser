package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Message represents an individual EDIFACT message extracted during a
// specific processing version.
type (
	Message struct {
		ID                  uuid.UUID
		InterchangeID       uuid.UUID
		ProcessingVersionID uuid.UUID
		SubscriptionID      *uuid.UUID
		MessageType         string
		MessageReference    string
		Segments            map[string]any
		CreatedAt           time.Time
	}
)

// NewMessage creates a validated Message. The subscriptionID is nullable
// (LOC 172 not always present).
func NewMessage(
	interchangeID, versionID uuid.UUID,
	subscriptionID *uuid.UUID,
	msgType, msgRef string,
	segments map[string]any,
) (*Message, error) {
	if msgType == "" {
		return nil, fmt.Errorf("creating message: %w", ErrEmptyMessageType)
	}

	if msgRef == "" {
		return nil, fmt.Errorf("creating message: %w", ErrEmptyMessageReference)
	}

	return &Message{
		ID:                  uuid.Must(uuid.NewV7()),
		InterchangeID:       interchangeID,
		ProcessingVersionID: versionID,
		SubscriptionID:      subscriptionID,
		MessageType:         msgType,
		MessageReference:    msgRef,
		Segments:            segments,
		CreatedAt:           time.Now().UTC(),
	}, nil
}
