package msgbroker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
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
		},
	}
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, payload interface{}) error {
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

func (p *Producer) Close() {
	p.writer.Close()
}
