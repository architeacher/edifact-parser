package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Subscription represents a metering point (Zaehlpunkt) referenced in messages.
// Created on first encounter via upsert.
type (
	Subscription struct {
		ID         uuid.UUID
		Identifier string
		CreatedAt  time.Time
	}
)

// NewSubscription creates a validated Subscription.
func NewSubscription(identifier string) (*Subscription, error) {
	if identifier == "" {
		return nil, fmt.Errorf("creating subscription: %w", ErrEmptyIdentifier)
	}

	return &Subscription{
		ID:         uuid.Must(uuid.NewV7()),
		Identifier: identifier,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
