package infrastructure

import (
	"github.com/segmentio/kafka-go"

	"github.com/architeacher/nomos-technical-challenge/services/svc-ingestion/internal/config"
)

// NewKafkaWriter creates a synchronous Kafka writer for domain events.
func NewKafkaWriter(cfg config.KafkaConfig) *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
}
