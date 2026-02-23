package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/domain/model"
	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/ports"
	kafkago "github.com/segmentio/kafka-go"
)

// Compile-time interface compliance check.
var _ ports.EventPublisher = (*Publisher)(nil)

// Publisher writes domain events to a Kafka topic as JSON messages.
// It implements ports.EventPublisher.
type (
	Publisher struct {
		writer *kafkago.Writer
	}
)

// NewPublisher creates a Publisher backed by a synchronous kafka.Writer.
// Brokers must contain at least one address. Topic is the destination topic name.
func NewPublisher(brokers []string, topic string) (*Publisher, error) {
	if len(brokers) == 0 {
		return nil, fmt.Errorf("creating kafka publisher: %w", ErrNoBrokers)
	}

	if topic == "" {
		return nil, fmt.Errorf("creating kafka publisher: %w", ErrEmptyTopic)
	}

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(brokers...),
		Topic:        topic,
		RequiredAcks: kafkago.RequireAll,
		Compression:  kafkago.Gzip,
	}

	return &Publisher{writer: writer}, nil
}

// NewPublisherFromWriter creates a Publisher from an existing kafka.Writer.
// This is useful for reusing an existing writer across multiple publishers.
func NewPublisherFromWriter(writer *kafkago.Writer) *Publisher {
	return &Publisher{writer: writer}
}

// Publish serializes each DomainEvent as JSON and writes them to Kafka in a
// single batch call. The message key is the interchange ID (empty if nil).
func (p *Publisher) Publish(ctx context.Context, events ...*model.DomainEvent) error {
	if len(events) == 0 {
		return nil
	}

	messages := make([]kafkago.Message, 0, len(events))

	for _, event := range events {
		value, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("marshalling event %s: %w", event.ID, err)
		}

		var key []byte
		if event.InterchangeID != nil {
			key = []byte(event.InterchangeID.String())
		}

		messages = append(messages, kafkago.Message{
			Key:   key,
			Value: value,
		})
	}

	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		return fmt.Errorf("writing events to kafka: %w", err)
	}

	return nil
}

// Close flushes pending writes and releases the underlying Kafka connection.
func (p *Publisher) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("closing kafka publisher: %w", err)
	}

	return nil
}
