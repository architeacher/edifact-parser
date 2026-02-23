package edifact

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

const (
	// bufferSize is the size of the buffered reader (64 KB) for streaming large EDIFACT files.
	bufferSize = 64 * 1024

	// eof is the sentinel value indicating end-of-input.
	eof byte = 0
)

// Lexer converts an EDIFACT byte stream into a sequence of typed tokens.
// It uses a buffered reader for memory-efficient streaming and tracks
// position (line/column) for diagnostic messages.
type Lexer struct {
	reader     *bufio.Reader
	delimiters *Delimiters
	current    byte
	peek       byte
	line       int
	column     int
	atEOF      bool
	// afterSegmentEnd tracks whether we just emitted a SegmentEnd token,
	// meaning the next alphanumeric sequence should be a segment tag.
	afterSegmentEnd bool
	// atSegmentStart tracks whether we are at the start of a segment
	// (before any data element separator has been seen).
	atSegmentStart bool
}

// NewLexer creates a new Lexer that tokenizes the provided EDIFACT data
// using the given delimiter set.
func NewLexer(data []byte, delimiters *Delimiters) *Lexer {
	lexer := &Lexer{
		reader:          bufio.NewReaderSize(bytes.NewReader(data), bufferSize),
		delimiters:      delimiters,
		line:            1,
		column:          1,
		afterSegmentEnd: true,
		atSegmentStart:  true,
	}

	lexer.current = lexer.readByte()
	lexer.peek = lexer.readByte()

	return lexer
}

// NextToken returns the next token from the EDIFACT stream.
// It returns TokenEOF when the input is exhausted and TokenError
// if an invalid sequence is encountered.
func (l *Lexer) NextToken() Token {
	if l.afterSegmentEnd {
		l.skipInterSegmentWhitespace()
	}

	if l.current == eof {
		return Token{Type: TokenEOF, Line: l.line, Column: l.column}
	}

	// After a segment end (or at the start), the next alphanumeric
	// sequence is a segment tag.
	if l.afterSegmentEnd && isAlphanumeric(l.current) {
		return l.lexSegmentTag()
	}

	// Segment terminator.
	if l.current == l.delimiters.SegmentTerminator {
		return l.lexSegmentEnd()
	}

	// Data element separator.
	if l.current == l.delimiters.DataElementSeparator {
		l.advance()
		l.atSegmentStart = false

		return l.lexValue(TokenDataElement)
	}

	// Component separator.
	if l.current == l.delimiters.ComponentSeparator {
		l.advance()

		return l.lexValue(TokenComponent)
	}

	// If we're at a segment start and see alphanumeric, it's a tag.
	if l.atSegmentStart && isAlphanumeric(l.current) {
		return l.lexSegmentTag()
	}

	return Token{
		Type:   TokenError,
		Value:  []byte(fmt.Sprintf("unexpected character %q", l.current)),
		Line:   l.line,
		Column: l.column,
	}
}

// lexSegmentTag reads a segment tag consisting of consecutive alphanumeric bytes.
func (l *Lexer) lexSegmentTag() Token {
	line, col := l.line, l.column

	var buf []byte

	for l.current != eof && isAlphanumeric(l.current) {
		buf = append(buf, l.current)
		l.advance()
	}

	l.afterSegmentEnd = false
	l.atSegmentStart = true

	return Token{
		Type:   TokenSegmentTag,
		Value:  buf,
		Line:   line,
		Column: col,
	}
}

// lexValue reads a value (data element or component) until the next
// delimiter or end of input. It handles the release character for escaping.
func (l *Lexer) lexValue(tokenType TokenType) Token {
	line, col := l.line, l.column

	var buf []byte

	for l.current != eof {
		// Release character: the next byte is literal.
		if l.delimiters.IsReleaseChar(l.current) {
			next := l.peek
			if next == eof {
				return Token{
					Type:   TokenError,
					Value:  []byte("release character at end of input"),
					Line:   l.line,
					Column: l.column,
				}
			}

			l.advance() // skip release character
			buf = append(buf, l.current)
			l.advance() // skip escaped character

			continue
		}

		// Stop at any structural delimiter.
		if l.isDelimiter(l.current) {
			break
		}

		buf = append(buf, l.current)
		l.advance()
	}

	if buf == nil {
		buf = []byte{}
	}

	return Token{
		Type:   tokenType,
		Value:  buf,
		Line:   line,
		Column: col,
	}
}

// lexSegmentEnd emits a SegmentEnd token and advances past the terminator.
func (l *Lexer) lexSegmentEnd() Token {
	line, col := l.line, l.column
	terminator := l.current

	l.advance()
	l.afterSegmentEnd = true
	l.atSegmentStart = true

	return Token{
		Type:   TokenSegmentEnd,
		Value:  []byte{terminator},
		Line:   line,
		Column: col,
	}
}

// skipInterSegmentWhitespace consumes whitespace characters (spaces, tabs,
// newlines, carriage returns) that appear between segments.
func (l *Lexer) skipInterSegmentWhitespace() {
	for l.current != eof && isWhitespace(l.current) {
		l.advance()
	}
}

// advance moves the scanner forward by one byte, updating position tracking.
func (l *Lexer) advance() {
	if l.current == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}

	l.current = l.peek
	l.peek = l.readByte()
}

// readByte reads a single byte from the buffered reader.
// Returns eof (0) when the input is exhausted.
func (l *Lexer) readByte() byte {
	if l.atEOF {
		return eof
	}

	b, err := l.reader.ReadByte()
	if err != nil {
		if err == io.EOF {
			l.atEOF = true
		}

		return eof
	}

	return b
}

// isDelimiter reports whether b is any structural EDIFACT delimiter.
func (l *Lexer) isDelimiter(b byte) bool {
	return b == l.delimiters.ComponentSeparator ||
		b == l.delimiters.DataElementSeparator ||
		b == l.delimiters.SegmentTerminator
}

// isAlphanumeric reports whether b is an ASCII letter or digit.
func isAlphanumeric(b byte) bool {
	return (b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z') ||
		(b >= '0' && b <= '9')
}

// isWhitespace reports whether b is a whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
