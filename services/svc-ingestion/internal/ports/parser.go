package ports

import (
	"time"
)

// ParseResult holds the structured output from parsing an EDIFACT interchange.
type (
	ParseResult struct {
		SenderID          string
		SenderQualifier   string
		ReceiverID        string
		ReceiverQualifier string
		Reference         string
		PreparedAt        time.Time
		ContentHash       string
		Messages          []ParsedMessage
		Subscriptions     []string
	}
)

// ParsedMessage represents a single parsed EDIFACT message within an interchange.
type (
	ParsedMessage struct {
		MessageType      string
		MessageVersion   string
		MessageRelease   string
		MessageReference string
		SubscriptionID   string
		Segments         []map[string]any
	}
)

// FileParser converts raw EDIFACT bytes into a structured parse result.
type (
	FileParser interface {
		// Parse processes raw EDIFACT file content and returns the structured result.
		Parse(data []byte) (*ParseResult, error)
	}
)
