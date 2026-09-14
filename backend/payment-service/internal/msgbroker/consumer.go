package msgbroker

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic string, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  groupID,
			MinBytes: 1,    // 1B for low-volume event responsiveness
			MaxBytes: 10e6, // 10MB
			MaxWait:  1 * time.Second,
			// Anggota grup yang mati mendadak (SIGKILL, OOM) baru
			// dikeluarkan broker setelah SessionTimeout; sampai saat itu
			// partisinya tidak dibagikan ke siapa pun. Default 30 detik
			// terlalu lama untuk pesan yang menahan pembayaran.
			SessionTimeout:    10 * time.Second,
			HeartbeatInterval: 3 * time.Second,
		}),
	}
}

// FetchMessage mengambil pesan TANPA meng-commit offset-nya. Offset baru
// maju lewat CommitMessages, setelah pesan selesai diproses -- ReadMessage
// meng-commit sebelum handler jalan, sehingga proses yang mati di tengah
// kehilangan pesannya.
func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.FetchMessage(ctx)
}

func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.reader.CommitMessages(ctx, msgs...)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
