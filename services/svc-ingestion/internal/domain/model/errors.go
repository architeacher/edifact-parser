package model

import (
	"errors"
)

var (
	// ErrDuplicateInterchange is returned when an interchange with the same content hash already exists.
	ErrDuplicateInterchange = errors.New("interchange with this content hash already exists")

	// ErrEmptyContentHash is returned when the content hash is missing or invalid.
	ErrEmptyContentHash = errors.New("content hash must be a valid SHA-256 hex string (64 characters)")

	// ErrEmptyIdentifier is returned when a subscription identifier is empty.
	ErrEmptyIdentifier = errors.New("identifier must not be empty")

	// ErrEmptyMessageReference is returned when a message reference is empty.
	ErrEmptyMessageReference = errors.New("message reference must not be empty")

	// ErrEmptyMessageType is returned when a message type is empty.
	ErrEmptyMessageType = errors.New("message type must not be empty")

	// ErrEmptyParserVersion is returned when the parser version string is empty.
	ErrEmptyParserVersion = errors.New("parser version must not be empty")

	// ErrEmptyRawContent is returned when raw file content is nil or zero-length.
	ErrEmptyRawContent = errors.New("raw content must not be empty")

	// ErrEmptyReason is returned when a failure reason is empty.
	ErrEmptyReason = errors.New("reason must not be empty")

	// ErrEmptyReceiver is returned when the receiver identifier is empty.
	ErrEmptyReceiver = errors.New("receiver must not be empty")

	// ErrEmptyReference is returned when the interchange reference is empty.
	ErrEmptyReference = errors.New("reference must not be empty")

	// ErrEmptySender is returned when the sender identifier is empty.
	ErrEmptySender = errors.New("sender must not be empty")

	// ErrEventTypeEmpty is returned when a domain event type is empty.
	ErrEventTypeEmpty = errors.New("event type must not be empty")

	// ErrInterchangeNotFound is returned when an interchange cannot be located by ID or hash.
	ErrInterchangeNotFound = errors.New("interchange not found")

	// ErrInvalidVersionNumber is returned when a version number is not positive.
	ErrInvalidVersionNumber = errors.New("version number must be positive")

	// ErrMessageNotFound is returned when a message cannot be located.
	ErrMessageNotFound = errors.New("message not found")

	// ErrProcessingVersionNotFound is returned when a processing version cannot be located.
	ErrProcessingVersionNotFound = errors.New("processing version not found")

	// ErrSubscriptionNotFound is returned when a subscription cannot be located.
	ErrSubscriptionNotFound = errors.New("subscription not found")

	// ErrVersionConflict is returned when optimistic concurrency detects a version collision.
	ErrVersionConflict = errors.New("version conflict: another process has incremented the version number")
)
