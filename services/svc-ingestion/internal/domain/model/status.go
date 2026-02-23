package model

const (
	// StatusCompleted indicates successful parsing and persistence.
	StatusCompleted InterchangeStatus = "completed"

	// StatusFailed indicates a parse or persistence failure.
	StatusFailed InterchangeStatus = "failed"
)

// InterchangeStatus represents the processing state of an interchange.
// Only two terminal states exist — no transient "processing" state is persisted.
type (
	InterchangeStatus string
)

var validStatuses = map[InterchangeStatus]struct{}{
	StatusCompleted: {},
	StatusFailed:    {},
}

// IsValid reports whether s is a recognized interchange status.
func (s InterchangeStatus) IsValid() bool {
	_, ok := validStatuses[s]

	return ok
}

// String returns the string representation of the status.
func (s InterchangeStatus) String() string {
	return string(s)
}
