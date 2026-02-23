package queries

import (
	"errors"
)

var (
	// ErrNilInterchangeID is returned when an interchange ID is nil/zero.
	ErrNilInterchangeID = errors.New("interchange ID must not be nil")
)
