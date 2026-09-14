package events

import (
	"context"
	"encoding/json"
	"log"
	"time"

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
			// WriteMessages sinkron menunggu batch terisi atau BatchTimeout
			// lewat. Default 1 detik berarti worker outbox yang mengirim
			// satu per satu butuh ~1 detik per event -- batch 50 memakan
			// hampir semenit.
			BatchTimeout: 10 * time.Millisecond,
			// kafka.Writer{} default ke RequireNone: WriteMessages kembali
			// sukses begitu bytes masuk socket, tanpa ack broker. Outbox
			// yang menandai PUBLISHED atas dasar itu bohong -- broker yang
			// mati sesaat kemudian tidak pernah punya pesannya.
			RequiredAcks: kafka.RequireAll,
			// Topik yang belum punya consumer belum tentu sudah ada di
			// broker. payment.orphaned adalah contohnya: belum ada yang
			// mendengarkan, dan tanpa ini publikasinya gagal terus dengan
			// "Unknown Topic Or Partition" -- persis untuk peristiwa yang
			// paling tidak boleh hilang.
			AllowAutoTopicCreation: true,
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
