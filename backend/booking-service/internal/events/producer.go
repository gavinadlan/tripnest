package events

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	Publish(ctx context.Context, topic string, key string, payload interface{}) error
	Close()
}

type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string) *KafkaProducer {
	log.Printf("Connecting to Kafka brokers: %v", brokers)
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, topic string, key string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: data,
	})
}

func (p *KafkaProducer) Close() {
	if err := p.writer.Close(); err != nil {
		log.Printf("Failed to close kafka writer: %v", err)
	}
}
