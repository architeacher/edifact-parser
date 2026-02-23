package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ProcessingVersion represents a single parsing pass of an interchange.
// Multiple versions exist after reprocessing.
type (
	ProcessingVersion struct {
		ID            uuid.UUID
		InterchangeID uuid.UUID
		VersionNumber int
		ParserVersion string
		FormatVersion string
		IsActive      bool
		CreatedAt     time.Time
	}
)

// NewProcessingVersion creates a validated ProcessingVersion in the active state.
func NewProcessingVersion(
	interchangeID uuid.UUID,
	versionNumber int,
	parserVersion, formatVersion string,
) (*ProcessingVersion, error) {
	if versionNumber < 1 {
		return nil, fmt.Errorf("creating processing version: %w", ErrInvalidVersionNumber)
	}

	if parserVersion == "" {
		return nil, fmt.Errorf("creating processing version: %w", ErrEmptyParserVersion)
	}

	return &ProcessingVersion{
		ID:            uuid.Must(uuid.NewV7()),
		InterchangeID: interchangeID,
		VersionNumber: versionNumber,
		ParserVersion: parserVersion,
		FormatVersion: formatVersion,
		IsActive:      true,
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// Deactivate marks this version as inactive.
func (v *ProcessingVersion) Deactivate() {
	v.IsActive = false
}

// Activate marks this version as active.
func (v *ProcessingVersion) Activate() {
	v.IsActive = true
}
