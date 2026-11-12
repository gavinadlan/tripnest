// Package inbox adalah loop consumer Kafka yang idempoten dan at-least-once.
//
// Urutannya untuk setiap pesan:
//
//  1. FetchMessage -- offset BELUM di-commit.
//  2. Satu transaksi Postgres: jalankan handler, lalu INSERT ke
//     processed_messages. Kalau insert-nya bentrok, pesan ini sudah
//     pernah diproses: transaksi di-rollback dan efek handler ikut batal.
//  3. Commit transaksi.
//  4. CommitMessages -- baru sekarang offset Kafka maju.
//
// Kalau proses mati di antara langkah mana pun, pesan datang lagi setelah
// restart: kalau transaksinya belum commit, handler jalan ulang; kalau
// sudah, processed_messages menolaknya. Versi lama menandai
// processed_messages lebih dulu dan ReadMessage meng-commit offset sebelum
// handler jalan, jadi handler yang gagal tidak pernah dicoba lagi --
// at-most-once yang menyamar sebagai at-least-once.
package inbox

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/segmentio/kafka-go"
)

// Reader adalah irisan kafka.Reader yang dipakai Consume.
type Reader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

// Handler menerima pesan di dalam transaksi yang dibawa ctx. Setiap query
// repository yang memakai pgtx.From(ctx, ...) otomatis ikut di dalamnya.
type Handler func(ctx context.Context, msg kafka.Message) error

// ErrDuplicate menandai pesan yang sudah pernah diproses consumer group ini.
var ErrDuplicate = errors.New("pesan sudah pernah diproses")

// PermanentError membungkus kegagalan yang tidak akan sembuh dengan
// diulang -- payload yang tidak bisa di-decode, misalnya. Pesan seperti
// ini dilewati (dan dicatat keras) supaya tidak menyumbat partisinya.
type PermanentError struct{ Err error }

func (e *PermanentError) Error() string { return "kegagalan permanen: " + e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// Permanent menandai err sebagai tidak layak diulang.
func Permanent(err error) error { return &PermanentError{Err: err} }

const (
	initialBackoff = time.Second
	maxBackoff     = 30 * time.Second
)

// Consume membaca pesan sampai ctx selesai. Pesan yang handler-nya gagal
// karena sebab sementara (database atau layanan lain sedang tidak bisa
// dihubungi) diulang di tempat dengan backoff eksponensial; consumer tidak
// pindah ke pesan berikutnya sebelum yang ini beres. Itu yang membuat
// "matikan Kafka atau Postgres sebentar" tidak menghilangkan satu pesan pun.
func Consume(ctx context.Context, reader Reader, pool *pgxpool.Pool, group string, handle Handler) {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[inbox %s] gagal membaca pesan: %v", group, err)
			if !sleep(ctx, initialBackoff) {
				return
			}
			continue
		}

		if !process(ctx, pool, group, msg, handle) {
			return
		}

		// Kalau commit offset gagal, pesan ini datang lagi setelah
		// rebalance atau restart dan processed_messages yang menolaknya.
		if err := reader.CommitMessages(ctx, msg); err != nil && ctx.Err() == nil {
			log.Printf("[inbox %s] gagal commit offset topic=%s key=%s: %v", group, msg.Topic, string(msg.Key), err)
		}
	}
}

// process mengulang satu pesan sampai berhasil, terdeteksi duplikat, atau
// gagal permanen. Mengembalikan false hanya bila ctx selesai.
func process(ctx context.Context, pool *pgxpool.Pool, group string, msg kafka.Message, handle Handler) bool {
	key := MessageKey(msg)
	backoff := initialBackoff

	for attempt := 1; ; attempt++ {
		err := pgtx.Transact(ctx, pool, func(txCtx context.Context) error {
			if err := handle(txCtx, msg); err != nil {
				return err
			}
			inserted, err := MarkProcessed(txCtx, pgtx.From(txCtx, pool), group, msg.Topic, key)
			if err != nil {
				return err
			}
			if !inserted {
				return ErrDuplicate
			}
			return nil
		})

		var permanent *PermanentError
		switch {
		case err == nil:
			return true
		case errors.Is(err, ErrDuplicate):
			log.Printf("[inbox %s] duplikat diabaikan topic=%s key=%s", group, msg.Topic, key)
			return true
		case errors.As(err, &permanent):
			log.Printf("[inbox %s] PESAN DILEWATI topic=%s partition=%d offset=%d: %v", group, msg.Topic, msg.Partition, msg.Offset, err)
			return true
		case ctx.Err() != nil:
			return false
		}

		log.Printf("[inbox %s] percobaan ke-%d gagal topic=%s key=%s, ulang dalam %s: %v", group, attempt, msg.Topic, key, backoff, err)
		if !sleep(ctx, backoff) {
			return false
		}
		if backoff *= 2; backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// MarkProcessed mencatat (group, topic, key) dan mengembalikan false bila
// kombinasi itu sudah ada. Dipanggil di dalam transaksi handler; boleh
// juga dipakai langsung oleh alur non-Kafka yang butuh dedup serupa.
func MarkProcessed(ctx context.Context, q pgtx.Querier, group, topic, key string) (bool, error) {
	tag, err := q.Exec(ctx, `
		INSERT INTO processed_messages (consumer_group, topic, message_key, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (consumer_group, topic, message_key) DO NOTHING
	`, group, topic, key)
	if err != nil {
		return false, fmt.Errorf("gagal mencatat processed_messages: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// MessageKey mengembalikan kunci dedup pesan. Pesan tanpa key (seharusnya
// tidak ada, semua producer kita mengisi key dengan booking_id) memakai
// posisi di partisi supaya tidak semua pesan kosong dianggap satu.
func MessageKey(msg kafka.Message) string {
	if len(msg.Key) > 0 {
		return string(msg.Key)
	}
	return fmt.Sprintf("%s/%d/%d", msg.Topic, msg.Partition, msg.Offset)
}

func sleep(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
