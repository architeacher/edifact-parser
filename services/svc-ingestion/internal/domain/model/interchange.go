package model

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	sha256HexLen = 64
)

// Interchange represents a submitted EDIFACT file — the atomic unit of ingestion.
type (
	Interchange struct {
		ID                uuid.UUID
		SenderID          string
		SenderQualifier   string
		ReceiverID        string
		ReceiverQualifier string
		PreparedAt        time.Time
		Reference         string
		RawContent        []byte
		ContentHash       string
		Status            InterchangeStatus
		ErrorDetail       *string
		CreatedAt         time.Time
		UpdatedAt         time.Time
	}
)

// NewInterchange creates a validated Interchange in the completed state.
func NewInterchange(
	sender, senderQualifier, receiver, receiverQualifier string,
	preparedAt time.Time,
	reference string,
	rawContent []byte,
	contentHash string,
) (*Interchange, error) {
	if sender == "" {
		return nil, fmt.Errorf("creating interchange: %w", ErrEmptySender)
	}

	if receiver == "" {
		return nil, fmt.Errorf("creating interchange: %w", ErrEmptyReceiver)
	}

	if reference == "" {
		return nil, fmt.Errorf("creating interchange: %w", ErrEmptyReference)
	}

	if len(rawContent) == 0 {
		return nil, fmt.Errorf("creating interchange: %w", ErrEmptyRawContent)
	}

	if err := validateContentHash(contentHash); err != nil {
		return nil, fmt.Errorf("creating interchange: %w", err)
	}

	now := time.Now().UTC()

	return &Interchange{
		ID:                uuid.Must(uuid.NewV7()),
		SenderID:          sender,
		SenderQualifier:   senderQualifier,
		ReceiverID:        receiver,
		ReceiverQualifier: receiverQualifier,
		PreparedAt:        preparedAt,
		Reference:         reference,
		RawContent:        rawContent,
		ContentHash:       contentHash,
		Status:            StatusCompleted,
		ErrorDetail:       nil,
		CreatedAt:         now,
		UpdatedAt:         now,
	}, nil
}

// MarkFailed transitions the interchange to the failed state with the given reason.
func (i *Interchange) MarkFailed(reason string) error {
	if reason == "" {
		return fmt.Errorf("marking interchange failed: %w", ErrEmptyReason)
	}

	i.Status = StatusFailed
	i.ErrorDetail = &reason
	i.UpdatedAt = time.Now().UTC()

	return nil
}

func validateContentHash(hash string) error {
	if len(hash) != sha256HexLen {
		return fmt.Errorf("%w", ErrEmptyContentHash)
	}

	if _, err := hex.DecodeString(hash); err != nil {
		return fmt.Errorf("%w", ErrEmptyContentHash)
	}

	return nil
}
