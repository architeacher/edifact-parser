package kafka

import (
	"errors"
)

var (
	// ErrNoBrokers is returned when no Kafka broker addresses are provided.
	ErrNoBrokers = errors.New("at least one broker address is required")

	// ErrEmptyTopic is returned when the topic name is empty.
	ErrEmptyTopic = errors.New("topic must not be empty")
)
