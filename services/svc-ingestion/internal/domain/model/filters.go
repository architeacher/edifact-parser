package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	// DefaultPageLimit is the default number of items per page.
	DefaultPageLimit = 20

	// MaxPageLimit is the maximum allowed items per page.
	MaxPageLimit = 100
)

// ReprocessFilter defines criteria for bulk reprocessing requests.
// At least one field should be set; omitting all fields matches the entire corpus.
type (
	ReprocessFilter struct {
		InterchangeID *uuid.UUID
		SenderID      *string
		MessageType   *string
		FromDate      *time.Time
		ToDate        *time.Time
	}
)

// MessageFilter defines criteria for querying messages.
type (
	MessageFilter struct {
		SubscriptionID      *uuid.UUID
		MessageType         *string
		InterchangeID       *uuid.UUID
		ProcessingVersionID *uuid.UUID
		Limit               int
		Cursor              *string
	}
)

// IsEmpty reports whether no filter criteria are set.
func (f ReprocessFilter) IsEmpty() bool {
	return f.InterchangeID == nil &&
		f.SenderID == nil &&
		f.MessageType == nil &&
		f.FromDate == nil &&
		f.ToDate == nil
}

// Validate normalises and validates the filter values.
func (f *MessageFilter) Validate() error {
	if f.Limit < 1 {
		f.Limit = DefaultPageLimit
	}

	if f.Limit > MaxPageLimit {
		f.Limit = MaxPageLimit
	}

	return nil
}
