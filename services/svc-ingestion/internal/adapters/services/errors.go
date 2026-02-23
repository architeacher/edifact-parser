package services

import (
	"errors"
)

var (
	// ErrFileTooLarge is returned when the submitted file exceeds the size limit.
	ErrFileTooLarge = errors.New("file exceeds maximum allowed size")

	// ErrFileEmpty is returned when the submitted file is nil or zero-length.
	ErrFileEmpty = errors.New("file must not be empty")
)
