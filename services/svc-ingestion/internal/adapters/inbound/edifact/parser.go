package edifact

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	// locSubscriptionPosition is the LOC qualifier for metering point identifiers.
	locSubscriptionPosition = "172"

	// segmentTagUNA is the service string advice identifier.
	segmentTagUNA = "UNA"

	// segmentTagUNB is the interchange header tag.
	segmentTagUNB = "UNB"

	// segmentTagUNH is the message header tag.
	segmentTagUNH = "UNH"

	// segmentTagUNT is the message trailer tag.
	segmentTagUNT = "UNT"

	// segmentTagUNZ is the interchange trailer tag.
	segmentTagUNZ = "UNZ"

	// segmentTagLOC is the location segment tag.
	segmentTagLOC = "LOC"

	// unbMinDataElements is the minimum number of data elements required in a UNB segment.
	unbMinDataElements = 5

	// unhMinDataElements is the minimum number of data elements required in a UNH segment.
	unhMinDataElements = 2

	// unzMinDataElements is the minimum number of data elements required in a UNZ segment.
	unzMinDataElements = 2

	// edifactTimestampFormat is the EDIFACT date+time format (YYMMDD:HHMM).
	edifactTimestampFormat = "060102:1504"
)

var (
	// ErrMissingUNB is returned when the interchange header is missing.
	ErrMissingUNB = errors.New("missing UNB interchange header")

	// ErrMissingUNZ is returned when the interchange trailer is missing.
	ErrMissingUNZ = errors.New("missing UNZ interchange trailer")

	// ErrMismatchedMessageCount is returned when UNZ message count does not match actual.
	ErrMismatchedMessageCount = errors.New("UNZ message count does not match actual messages parsed")

	// ErrMismatchedReference is returned when UNZ reference does not match UNB reference.
	ErrMismatchedReference = errors.New("UNZ interchange reference does not match UNB reference")

	// ErrMissingUNH is returned when a message header is expected but not found.
	ErrMissingUNH = errors.New("missing UNH message header")

	// ErrInvalidUNBSegment is returned when UNB segment has insufficient data elements.
	ErrInvalidUNBSegment = errors.New("UNB segment has insufficient data elements")

	// ErrInvalidUNHSegment is returned when UNH segment has insufficient data elements.
	ErrInvalidUNHSegment = errors.New("UNH segment has insufficient data elements")

	// ErrInvalidUNZSegment is returned when UNZ segment has insufficient data elements.
	ErrInvalidUNZSegment = errors.New("UNZ segment has insufficient data elements")
)

// ParseResult holds the structured output from parsing an EDIFACT interchange.
type ParseResult struct {
	// SenderID is the sender identification from UNB.
	SenderID string

	// SenderQualifier is the sender qualifier from UNB.
	SenderQualifier string

	// ReceiverID is the receiver identification from UNB.
	ReceiverID string

	// ReceiverQualifier is the receiver qualifier from UNB.
	ReceiverQualifier string

	// Reference is the interchange control reference from UNB.
	Reference string

	// PreparedAt is the preparation date+time from UNB.
	PreparedAt time.Time

	// ContentHash is the SHA-256 hex digest of the raw content.
	ContentHash string

	// Messages holds the parsed message data (one per UNH/UNT pair).
	Messages []ParsedMessage

	// Subscriptions holds unique subscription identifiers found in LOC segments.
	Subscriptions []string
}

// ParsedMessage represents a single parsed EDIFACT message within an interchange.
type ParsedMessage struct {
	// MessageType is the message type identifier (e.g., MSCONS, INVOIC, UTILMD).
	MessageType string

	// MessageVersion is the version number from UNH (e.g., D).
	MessageVersion string

	// MessageRelease is the release number from UNH (e.g., 11A).
	MessageRelease string

	// MessageReference is the message reference number from UNH.
	MessageReference string

	// SubscriptionID is the metering point identifier from LOC 172, if found.
	SubscriptionID string

	// Segments holds the raw segment data as a map for JSONB storage.
	Segments []map[string]any
}

// segment is an internal representation of a parsed EDIFACT segment.
type segment struct {
	tag            string
	dataElements   [][]string
	line           int
	column         int
}

// Parser converts a lexer token stream into an EDIFACT Interchange domain model.
type Parser struct {
	lexer        *Lexer
	currentToken Token
	errors       []string
	rawData      []byte
}

// NewParser creates a new Parser for the given EDIFACT data.
// It detects UNA service string advice to determine delimiters,
// falling back to ISO 9735 defaults.
func NewParser(data []byte) (*Parser, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("creating parser: data must not be empty")
	}

	delimiters := DefaultDelimiters()

	// Check for UNA prefix and parse custom delimiters.
	trimmed := trimLeadingWhitespace(data)
	if len(trimmed) >= unaLength && string(trimmed[:len(unaPrefix)]) == segmentTagUNA {
		parsed, err := ParseUNA(trimmed[:unaLength])
		if err != nil {
			return nil, fmt.Errorf("parsing UNA delimiters: %w", err)
		}

		delimiters = parsed
	}

	lexer := NewLexer(trimmed, delimiters)

	parser := &Parser{
		lexer:   lexer,
		rawData: data,
	}

	parser.advance()

	return parser, nil
}

// Parse processes the EDIFACT token stream and returns the structured parse result.
// It collects all errors encountered during parsing rather than failing on the first one.
func (p *Parser) Parse() (*ParseResult, error) {
	result := &ParseResult{
		ContentHash: computeContentHash(p.rawData),
	}

	// Skip UNA segment tag if present (delimiters already parsed in constructor).
	if p.currentToken.Type == TokenSegmentTag && string(p.currentToken.Value) == segmentTagUNA {
		p.skipSegment()
	}

	// Parse UNB (required).
	if !p.isSegmentTag(segmentTagUNB) {
		p.addError(p.currentToken.Line, p.currentToken.Column, ErrMissingUNB.Error())

		return nil, p.combinedError()
	}

	if err := p.parseUNB(result); err != nil {
		return nil, err
	}

	// Parse messages (UNH/UNT pairs).
	subscriptionSet := make(map[string]struct{})

	for p.isSegmentTag(segmentTagUNH) {
		msg, err := p.parseMessage()
		if err != nil {
			continue
		}

		result.Messages = append(result.Messages, *msg)

		if msg.SubscriptionID != "" {
			if _, exists := subscriptionSet[msg.SubscriptionID]; !exists {
				subscriptionSet[msg.SubscriptionID] = struct{}{}
				result.Subscriptions = append(result.Subscriptions, msg.SubscriptionID)
			}
		}
	}

	// Parse UNZ (required).
	if !p.isSegmentTag(segmentTagUNZ) {
		p.addError(p.currentToken.Line, p.currentToken.Column, ErrMissingUNZ.Error())

		return nil, p.combinedError()
	}

	if err := p.parseUNZ(result); err != nil {
		return nil, err
	}

	if len(p.errors) > 0 {
		return nil, p.combinedError()
	}

	return result, nil
}

// parseUNB extracts interchange header data from the UNB segment.
func (p *Parser) parseUNB(result *ParseResult) error {
	seg := p.readSegment()

	if len(seg.dataElements) < unbMinDataElements {
		p.addError(seg.line, seg.column,
			fmt.Sprintf("%s: expected at least %d, got %d", ErrInvalidUNBSegment, unbMinDataElements, len(seg.dataElements)))

		return p.combinedError()
	}

	// Data element 1: syntax identifier (e.g., UNOC:3) - skip for now.
	// Data element 2: sender (ID:qualifier).
	if len(seg.dataElements[1]) > 0 {
		result.SenderID = seg.dataElements[1][0]
	}

	if len(seg.dataElements[1]) > 1 {
		result.SenderQualifier = seg.dataElements[1][1]
	}

	// Data element 3: receiver (ID:qualifier).
	if len(seg.dataElements[2]) > 0 {
		result.ReceiverID = seg.dataElements[2][0]
	}

	if len(seg.dataElements[2]) > 1 {
		result.ReceiverQualifier = seg.dataElements[2][1]
	}

	// Data element 4: date+time (YYMMDD:HHMM).
	if len(seg.dataElements[3]) >= 2 {
		dateStr := seg.dataElements[3][0] + ":" + seg.dataElements[3][1]

		prepared, err := time.Parse(edifactTimestampFormat, dateStr)
		if err != nil {
			p.addError(seg.line, seg.column,
				fmt.Sprintf("invalid UNB date/time %q: %v", dateStr, err))
		} else {
			result.PreparedAt = prepared
		}
	}

	// Data element 5: interchange control reference.
	if len(seg.dataElements[4]) > 0 {
		result.Reference = seg.dataElements[4][0]
	}

	// Validate required fields.
	if result.SenderID == "" {
		p.addError(seg.line, seg.column, "UNB sender identification is empty")
	}

	if result.ReceiverID == "" {
		p.addError(seg.line, seg.column, "UNB receiver identification is empty")
	}

	if result.Reference == "" {
		p.addError(seg.line, seg.column, "UNB interchange control reference is empty")
	}

	if len(p.errors) > 0 {
		return p.combinedError()
	}

	return nil
}

// parseMessage parses a single UNH/UNT message pair and its body segments.
func (p *Parser) parseMessage() (*ParsedMessage, error) {
	msg := &ParsedMessage{}

	// Parse UNH header.
	unhSeg := p.readSegment()

	if len(unhSeg.dataElements) < unhMinDataElements {
		p.addError(unhSeg.line, unhSeg.column,
			fmt.Sprintf("%s: expected at least %d, got %d", ErrInvalidUNHSegment, unhMinDataElements, len(unhSeg.dataElements)))

		return nil, p.combinedError()
	}

	// Data element 1: message reference number.
	if len(unhSeg.dataElements[0]) > 0 {
		msg.MessageReference = unhSeg.dataElements[0][0]
	}

	// Data element 2: message identifier (type:version:release:agency:association).
	if len(unhSeg.dataElements[1]) > 0 {
		msg.MessageType = unhSeg.dataElements[1][0]
	}

	if len(unhSeg.dataElements[1]) > 1 {
		msg.MessageVersion = unhSeg.dataElements[1][1]
	}

	if len(unhSeg.dataElements[1]) > 2 {
		msg.MessageRelease = unhSeg.dataElements[1][2]
	}

	if msg.MessageReference == "" {
		p.addError(unhSeg.line, unhSeg.column, "UNH message reference is empty")
	}

	if msg.MessageType == "" {
		p.addError(unhSeg.line, unhSeg.column, "UNH message type is empty")
	}

	// Add UNH as first segment.
	msg.Segments = append(msg.Segments, segmentToMap(segmentTagUNH, unhSeg))

	// Parse body segments until UNT.
	for p.currentToken.Type != TokenEOF {
		if p.isSegmentTag(segmentTagUNT) {
			break
		}

		bodySeg := p.readSegment()

		// Check for LOC 172 subscription identifier.
		if bodySeg.tag == segmentTagLOC && len(bodySeg.dataElements) > 0 {
			if len(bodySeg.dataElements[0]) > 0 && bodySeg.dataElements[0][0] == locSubscriptionPosition {
				if len(bodySeg.dataElements) > 1 && len(bodySeg.dataElements[1]) > 0 {
					msg.SubscriptionID = bodySeg.dataElements[1][0]
				}
			}
		}

		msg.Segments = append(msg.Segments, segmentToMap(bodySeg.tag, bodySeg))
	}

	// Parse UNT trailer.
	if p.isSegmentTag(segmentTagUNT) {
		untSeg := p.readSegment()
		msg.Segments = append(msg.Segments, segmentToMap(segmentTagUNT, untSeg))
	} else {
		p.addError(p.currentToken.Line, p.currentToken.Column,
			fmt.Sprintf("expected UNT trailer for message %q", msg.MessageReference))
	}

	return msg, nil
}

// parseUNZ extracts and validates the interchange trailer.
func (p *Parser) parseUNZ(result *ParseResult) error {
	seg := p.readSegment()

	if len(seg.dataElements) < unzMinDataElements {
		p.addError(seg.line, seg.column,
			fmt.Sprintf("%s: expected at least %d, got %d", ErrInvalidUNZSegment, unzMinDataElements, len(seg.dataElements)))

		return p.combinedError()
	}

	// Data element 1: interchange control count.
	if len(seg.dataElements[0]) > 0 {
		countStr := seg.dataElements[0][0]

		var count int
		if _, err := fmt.Sscanf(countStr, "%d", &count); err == nil {
			if count != len(result.Messages) {
				p.addError(seg.line, seg.column,
					fmt.Sprintf("%s: UNZ declares %d, parsed %d",
						ErrMismatchedMessageCount, count, len(result.Messages)))
			}
		}
	}

	// Data element 2: interchange control reference.
	if len(seg.dataElements[1]) > 0 {
		unzRef := seg.dataElements[1][0]
		if unzRef != result.Reference {
			p.addError(seg.line, seg.column,
				fmt.Sprintf("%s: UNB=%q, UNZ=%q",
					ErrMismatchedReference, result.Reference, unzRef))
		}
	}

	if len(p.errors) > 0 {
		return p.combinedError()
	}

	return nil
}

// readSegment reads a complete segment (tag + data elements) from the token stream.
func (p *Parser) readSegment() segment {
	seg := segment{
		line:   p.currentToken.Line,
		column: p.currentToken.Column,
	}

	// Read segment tag.
	if p.currentToken.Type == TokenSegmentTag {
		seg.tag = string(p.currentToken.Value)
		p.advance()
	}

	// Read data elements and their components.
	var currentElement []string

	for p.currentToken.Type != TokenEOF {
		switch p.currentToken.Type {
		case TokenDataElement:
			if currentElement != nil {
				seg.dataElements = append(seg.dataElements, currentElement)
			}

			currentElement = []string{string(p.currentToken.Value)}
			p.advance()

		case TokenComponent:
			if currentElement == nil {
				currentElement = []string{}
			}

			currentElement = append(currentElement, string(p.currentToken.Value))
			p.advance()

		case TokenSegmentEnd:
			if currentElement != nil {
				seg.dataElements = append(seg.dataElements, currentElement)
			}

			p.advance()

			return seg

		case TokenSegmentTag:
			// Next segment started without terminator - unusual but handle gracefully.
			if currentElement != nil {
				seg.dataElements = append(seg.dataElements, currentElement)
			}

			return seg

		default:
			p.addError(p.currentToken.Line, p.currentToken.Column,
				fmt.Sprintf("unexpected token %s in segment %s", p.currentToken.Type, seg.tag))
			p.advance()
		}
	}

	// EOF reached mid-segment.
	if currentElement != nil {
		seg.dataElements = append(seg.dataElements, currentElement)
	}

	return seg
}

// skipSegment skips all tokens in the current segment until the next segment or EOF.
func (p *Parser) skipSegment() {
	for p.currentToken.Type != TokenEOF {
		if p.currentToken.Type == TokenSegmentEnd {
			p.advance()

			return
		}

		p.advance()
	}
}

// advance reads the next token from the lexer.
func (p *Parser) advance() {
	p.currentToken = p.lexer.NextToken()
}

// isSegmentTag checks whether the current token is a segment tag with the given name.
func (p *Parser) isSegmentTag(tag string) bool {
	return p.currentToken.Type == TokenSegmentTag && string(p.currentToken.Value) == tag
}

// addError records a parse error with position information.
func (p *Parser) addError(line, column int, msg string) {
	p.errors = append(p.errors, fmt.Sprintf("parser error at line %d, column %d: %s", line, column, msg))
}

// combinedError returns all collected errors as a single error.
func (p *Parser) combinedError() error {
	if len(p.errors) == 0 {
		return nil
	}

	return fmt.Errorf("EDIFACT parse failed:\n%s", strings.Join(p.errors, "\n"))
}

// segmentToMap converts a segment into a map suitable for JSONB storage.
func segmentToMap(tag string, seg segment) map[string]any {
	result := map[string]any{
		"tag": tag,
	}

	for index, element := range seg.dataElements {
		key := fmt.Sprintf("de%d", index+1)

		if len(element) == 1 {
			result[key] = element[0]
		} else {
			result[key] = element
		}
	}

	return result
}

// computeContentHash returns the SHA-256 hex digest of data.
func computeContentHash(data []byte) string {
	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])
}

// trimLeadingWhitespace trims leading whitespace bytes.
func trimLeadingWhitespace(data []byte) []byte {
	index := 0

	for index < len(data) && isWhitespace(data[index]) {
		index++
	}

	return data[index:]
}
