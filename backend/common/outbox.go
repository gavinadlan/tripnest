// Package outbox adalah worker transactional-outbox yang dipakai
// booking-, payment-, dan catalog-service. Ketiganya memakai tabel
// outbox_events dengan skema yang sama, jadi worker-nya cukup satu.
//
// Setiap putaran:
//
//  1. Buka transaksi, ambil batch baris PENDING yang sudah waktunya
//     dengan FOR UPDATE SKIP LOCKED. Replika lain yang berjalan bersamaan
//     melewati baris yang sedang kita pegang -- satu event hanya pernah
//     dipegang satu worker.
//  2. Publish satu per satu ke Kafka. Berhasil -> PUBLISHED. Gagal ->
//     retry_count naik dan next_attempt_at mundur secara eksponensial;
//     setelah MaxRetry, baris jadi FAILED dan disalin ke
//     outbox_dead_letters supaya bisa diperiksa dan diputar ulang.
//  3. Commit.
//
// Kalau worker mati di tengah batch, transaksinya rollback dan barisnya
// kembali PENDING. Event yang sempat terkirim akan terkirim lagi; itu
// kontrak at-least-once yang ditanggung consumer lewat processed_messages.
package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/env"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Publisher dipenuhi producer Kafka tiap service.
type Publisher interface {
	Publish(ctx context.Context, topic, key string, payload interface{}) error
}

type Config struct {
	Interval    time.Duration
	BatchSize   int
	MaxRetry    int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// ConfigFromEnv membaca OUTBOX_* dari environment; nilai yang tidak diisi
// memakai default yang masuk akal untuk lokal maupun demo.
func ConfigFromEnv() Config {
	cfg := Config{
		Interval:    time.Duration(env.GetInt("OUTBOX_INTERVAL_SECONDS", 2)) * time.Second,
		BatchSize:   env.GetInt("OUTBOX_BATCH_SIZE", 50),
		MaxRetry:    env.GetInt("OUTBOX_MAX_RETRY", 8),
		BaseBackoff: time.Duration(env.GetInt("OUTBOX_BASE_BACKOFF_SECONDS", 2)) * time.Second,
		MaxBackoff:  time.Duration(env.GetInt("OUTBOX_MAX_BACKOFF_SECONDS", 300)) * time.Second,
	}
	if cfg.Interval < time.Second {
		cfg.Interval = time.Second
	}
	if cfg.BatchSize < 1 {
		cfg.BatchSize = 50
	}
	if cfg.MaxRetry < 1 {
		cfg.MaxRetry = 1
	}
	if cfg.BaseBackoff < time.Second {
		cfg.BaseBackoff = time.Second
	}
	if cfg.MaxBackoff < cfg.BaseBackoff {
		cfg.MaxBackoff = cfg.BaseBackoff
	}
	return cfg
}

type Event struct {
	ID         string
	Topic      string
	Key        string
	Payload    json.RawMessage
	RetryCount int
}

type Worker struct {
	pool *pgxpool.Pool
	pub  Publisher
	cfg  Config
}

func NewWorker(pool *pgxpool.Pool, pub Publisher, cfg Config) *Worker {
	return &Worker{pool: pool, pub: pub, cfg: cfg}
}

// Enqueue menulis satu event ke outbox lewat q -- transaksi yang sama
// dengan perubahan datanya bila q adalah pgtx.From(ctx) di dalam Transact.
func Enqueue(ctx context.Context, q pgtx.Querier, topic, key string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal meng-encode payload %s: %w", topic, err)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO outbox_events (topic, message_key, payload, status, retry_count, next_attempt_at, created_at, updated_at)
		VALUES ($1, $2, $3, 'PENDING', 0, NOW(), NOW(), NOW())
	`, topic, key, data); err != nil {
		return fmt.Errorf("gagal menulis outbox %s: %w", topic, err)
	}
	return nil
}

// Run memutar worker sampai ctx selesai.
func (w *Worker) Run(ctx context.Context) {
	log.Printf("[outbox] worker aktif interval=%s batch=%d max_retry=%d backoff=%s..%s",
		w.cfg.Interval, w.cfg.BatchSize, w.cfg.MaxRetry, w.cfg.BaseBackoff, w.cfg.MaxBackoff)
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.RunOnce(ctx); err != nil && ctx.Err() == nil {
				log.Printf("[outbox] putaran gagal: %v", err)
			}
		}
	}
}

// RunOnce memproses satu batch. Dipisah dari Run supaya bisa dipanggil
// dari test tanpa menunggu ticker.
func (w *Worker) RunOnce(ctx context.Context) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("gagal membuka transaksi: %w", err)
	}
	defer tx.Rollback(ctx)

	events, err := claim(ctx, tx, w.cfg.BatchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return tx.Commit(ctx)
	}

	for _, evt := range events {
		if err := w.publish(ctx, evt); err != nil {
			if err := w.fail(ctx, tx, evt, err); err != nil {
				return err
			}
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE outbox_events
			SET status = 'PUBLISHED', published_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, evt.ID); err != nil {
			return fmt.Errorf("gagal menandai PUBLISHED %s: %w", evt.ID, err)
		}
		log.Printf("Published outbox event topic=%s key=%s", evt.Topic, evt.Key)
	}

	return tx.Commit(ctx)
}

// claim mengambil dan mengunci batch yang akan diproses putaran ini.
func claim(ctx context.Context, tx pgx.Tx, limit int) ([]Event, error) {
	rows, err := tx.Query(ctx, `
		SELECT id, topic, message_key, payload, retry_count
		FROM outbox_events
		WHERE status = 'PENDING' AND next_attempt_at <= NOW()
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil outbox: %w", err)
	}
	defer rows.Close()

	events := make([]Event, 0, limit)
	for rows.Next() {
		var evt Event
		if err := rows.Scan(&evt.ID, &evt.Topic, &evt.Key, &evt.Payload, &evt.RetryCount); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

// errPermanent menandai event yang tidak akan pernah bisa terbit -- key
// kosong atau payload rusak. Tidak ada gunanya diulang delapan kali.
var errPermanent = errors.New("event tidak valid")

func (w *Worker) publish(ctx context.Context, evt Event) error {
	if strings.TrimSpace(evt.Key) == "" {
		return fmt.Errorf("%w: message_key kosong", errPermanent)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return fmt.Errorf("%w: payload bukan objek JSON: %v", errPermanent, err)
	}
	return w.pub.Publish(ctx, evt.Topic, evt.Key, payload)
}

// fail mencatat kegagalan dan memutuskan: coba lagi nanti, atau menyerah.
func (w *Worker) fail(ctx context.Context, tx pgx.Tx, evt Event, cause error) error {
	retry := evt.RetryCount + 1
	giveUp := errors.Is(cause, errPermanent) || retry >= w.cfg.MaxRetry

	if !giveUp {
		// clock_timestamp(), bukan NOW(): NOW() adalah waktu mulai
		// transaksi, sedangkan publish yang gagal karena timeout sudah
		// memakan beberapa detik sejak itu. Dengan NOW(), jadwal ulangnya
		// bisa sudah lewat saat di-commit dan backoff-nya tidak terasa.
		delay := w.backoff(retry)
		if _, err := tx.Exec(ctx, `
			UPDATE outbox_events
			SET retry_count = $2, last_error = $3, next_attempt_at = clock_timestamp() + make_interval(secs => $4), updated_at = NOW()
			WHERE id = $1
		`, evt.ID, retry, cause.Error(), delay.Seconds()); err != nil {
			return fmt.Errorf("gagal mencatat retry %s: %w", evt.ID, err)
		}
		log.Printf("[outbox] gagal publish topic=%s key=%s (percobaan %d/%d), ulang dalam %s: %v",
			evt.Topic, evt.Key, retry, w.cfg.MaxRetry, delay, cause)
		return nil
	}

	if _, err := tx.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'FAILED', retry_count = $2, last_error = $3, updated_at = NOW()
		WHERE id = $1
	`, evt.ID, retry, cause.Error()); err != nil {
		return fmt.Errorf("gagal menandai FAILED %s: %w", evt.ID, err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_dead_letters (id, topic, message_key, payload, retry_count, last_error, created_at, failed_at)
		SELECT id, topic, message_key, payload, retry_count, last_error, created_at, NOW()
		FROM outbox_events WHERE id = $1
		ON CONFLICT (id) DO NOTHING
	`, evt.ID); err != nil {
		return fmt.Errorf("gagal menulis dead letter %s: %w", evt.ID, err)
	}
	log.Printf("[outbox] DEAD LETTER topic=%s key=%s id=%s setelah %d percobaan: %v",
		evt.Topic, evt.Key, evt.ID, retry, cause)
	return nil
}

// backoff: base * 2^(retry-1), dibatasi MaxBackoff. Dengan default 2s dan
// 8 percobaan: 2, 4, 8, 16, 32, 64, 128 detik -- sekitar 4 menit total
// sebelum menyerah, cukup untuk broker yang restart tanpa membanjiri log.
func (w *Worker) backoff(retry int) time.Duration {
	d := w.cfg.BaseBackoff
	for i := 1; i < retry && d < w.cfg.MaxBackoff; i++ {
		d *= 2
	}
	if d > w.cfg.MaxBackoff {
		d = w.cfg.MaxBackoff
	}
	return d
}
