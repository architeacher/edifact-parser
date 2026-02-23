package edifact

import (
	"fmt"
)

// TokenType classifies a lexical token produced by the EDIFACT scanner.
type TokenType int

const (
	// TokenUNA represents the UNA service string advice segment.
	TokenUNA TokenType = iota

	// TokenSegmentTag represents a segment identifier (e.g., UNB, UNH, BGM).
	TokenSegmentTag

	// TokenDataElement represents a data element separated by '+'.
	TokenDataElement

	// TokenComponent represents a component data element separated by ':'.
	TokenComponent

	// TokenSegmentEnd represents a segment terminator (').
	TokenSegmentEnd

	// TokenEOF signals the end of input.
	TokenEOF

	// TokenError signals a lexer error.
	TokenError
)

var tokenTypeNames = map[TokenType]string{
	TokenUNA:         "UNA",
	TokenSegmentTag:  "SegmentTag",
	TokenDataElement: "DataElement",
	TokenComponent:   "Component",
	TokenSegmentEnd:  "SegmentEnd",
	TokenEOF:         "EOF",
	TokenError:       "Error",
}

// String returns the human-readable name of the token type.
func (t TokenType) String() string {
	if name, ok := tokenTypeNames[t]; ok {
		return name
	}

	return fmt.Sprintf("Unknown(%d)", int(t))
}

// Token represents a single lexical unit produced by the EDIFACT scanner.
type Token struct {
	// Value holds the raw bytes of the token content.
	Value []byte

	// Type classifies the token.
	Type TokenType

	// Line is the 1-based line number where the token starts.
	Line int

	// Column is the 1-based column number where the token starts.
	Column int
}

// String returns a human-readable representation of the token for debugging.
func (t Token) String() string {
	return fmt.Sprintf("{Type: %s, Value: %q, Line: %d, Column: %d}", t.Type, t.Value, t.Line, t.Column)
}
