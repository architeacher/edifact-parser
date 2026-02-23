package edifact

import (
	"errors"
	"fmt"
)

const (
	// unaPrefix is the service string advice identifier.
	unaPrefix = "UNA"

	// unaLength is the total length of a UNA segment: 3-byte prefix + 6 delimiter characters.
	unaLength = 9

	// delimiterCount is the number of delimiter characters in a UNA segment.
	delimiterCount = 6
)

var (
	// ErrInvalidUNALength is returned when UNA data is not exactly 9 bytes.
	ErrInvalidUNALength = errors.New("UNA segment must be exactly 9 bytes")

	// ErrInvalidUNAPrefix is returned when data does not start with "UNA".
	ErrInvalidUNAPrefix = errors.New("UNA segment must start with 'UNA'")
)

// Delimiters holds the six EDIFACT delimiter characters defined by the
// UNA service string advice or the ISO 9735 defaults.
type Delimiters struct {
	// ComponentSeparator separates component data elements within a composite (default ':').
	ComponentSeparator byte

	// DataElementSeparator separates data elements within a segment (default '+').
	DataElementSeparator byte

	// DecimalNotation indicates the decimal mark character (default '.').
	DecimalNotation byte

	// ReleaseCharacter escapes the next character (default '?').
	ReleaseCharacter byte

	// RepetitionSeparator separates repeating data elements (default '*').
	RepetitionSeparator byte

	// SegmentTerminator marks the end of a segment (default '\'').
	SegmentTerminator byte
}

// DefaultDelimiters returns the ISO 9735 default delimiter set.
func DefaultDelimiters() *Delimiters {
	return &Delimiters{
		ComponentSeparator:   ':',
		DataElementSeparator: '+',
		DecimalNotation:      '.',
		ReleaseCharacter:     '?',
		RepetitionSeparator:  '*',
		SegmentTerminator:    '\'',
	}
}

// ParseUNA extracts delimiters from a UNA service string advice segment.
// The expected format is: "UNA" followed by exactly 6 delimiter bytes.
func ParseUNA(data []byte) (*Delimiters, error) {
	if len(data) < unaLength {
		return nil, fmt.Errorf("%w: got %d bytes", ErrInvalidUNALength, len(data))
	}

	if string(data[:len(unaPrefix)]) != unaPrefix {
		return nil, fmt.Errorf("%w: got %q", ErrInvalidUNAPrefix, string(data[:len(unaPrefix)]))
	}

	return &Delimiters{
		ComponentSeparator:   data[3],
		DataElementSeparator: data[4],
		DecimalNotation:      data[5],
		ReleaseCharacter:     data[6],
		RepetitionSeparator:  data[7],
		SegmentTerminator:    data[8],
	}, nil
}

// IsReleaseChar reports whether b matches the release (escape) character.
func (d *Delimiters) IsReleaseChar(b byte) bool {
	return b == d.ReleaseCharacter
}
