package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/outbox"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	UpsertInitiated(ctx context.Context, p *model.Payment) error
	GetByBookingID(ctx context.Context, bookingID string) (*model.Payment, error)
	List(ctx context.Context, status string) ([]model.Payment, error)
	GetByMidtransOrderID(ctx context.Context, orderID string) (*model.Payment, error)
	MarkWebhookProcessed(ctx context.Context, key string) (bool, error)
	ApplyWebhookResult(ctx context.Context, notification model.MidtransNotification, status string) (*model.Payment, error)
	AddOutboxEvent(ctx context.Context, topic, key string, payload interface{}) error
	// Transact menjalankan fn dalam satu transaksi. Semua method di atas
	// yang dipanggil dengan ctx milik fn ikut di dalamnya.
	Transact(ctx context.Context, fn func(ctx context.Context) error) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) PaymentRepository {
	return &postgresRepository{db: pool}
}

func (r *postgresRepository) Transact(ctx context.Context, fn func(ctx context.Context) error) error {
	return pgtx.Transact(ctx, r.db, fn)
}

// q mengembalikan transaksi yang dibawa ctx, atau pool bila tidak ada.
func (r *postgresRepository) q(ctx context.Context) pgtx.Querier {
	return pgtx.From(ctx, r.db)
}

func (r *postgresRepository) UpsertInitiated(ctx context.Context, p *model.Payment) error {
	query := `
        INSERT INTO payments (booking_id, amount, status, transaction_id, midtrans_order_id, snap_token, redirect_url, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		ON CONFLICT (booking_id) DO UPDATE SET
			amount = EXCLUDED.amount,
			status = EXCLUDED.status,
			transaction_id = EXCLUDED.transaction_id,
			midtrans_order_id = EXCLUDED.midtrans_order_id,
			snap_token = EXCLUDED.snap_token,
			redirect_url = EXCLUDED.redirect_url,
			updated_at = EXCLUDED.updated_at
        RETURNING id
    `
	p.CreatedAt = time.Now()
	err := r.q(ctx).QueryRow(
		ctx,
		query,
		p.BookingID,
		p.Amount,
		p.Status,
		p.TransactionID,
		p.MidtransOrderID,
		p.SnapToken,
		p.RedirectURL,
		p.CreatedAt,
	).Scan(&p.ID)
	if err != nil {
		return fmt.Errorf("failed to upsert payment: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetByBookingID(ctx context.Context, bookingID string) (*model.Payment, error) {
	query := `
		SELECT
			id::text,
			booking_id::text,
			amount::double precision,
			status,
			COALESCE(transaction_id, ''),
			COALESCE(midtrans_order_id, ''),
			COALESCE(snap_token, ''),
			COALESCE(redirect_url, ''),
			COALESCE(payment_type, ''),
			COALESCE(raw_notification, '{}'::jsonb),
			created_at,
			updated_at
		FROM payments
		WHERE booking_id = $1
	`
	return r.getByQuery(ctx, query, bookingID)
}

func (r *postgresRepository) List(ctx context.Context, status string) ([]model.Payment, error) {
	rows, err := r.q(ctx).Query(ctx, `
		SELECT
			id::text,
			booking_id::text,
			amount::double precision,
			status,
			COALESCE(transaction_id, ''),
			COALESCE(midtrans_order_id, ''),
			COALESCE(snap_token, ''),
			COALESCE(redirect_url, ''),
			COALESCE(payment_type, ''),
			COALESCE(raw_notification, '{}'::jsonb),
			COALESCE(transaction_status, ''),
			COALESCE(midtrans_status_code, ''),
			COALESCE(midtrans_gross_amount, ''),
			COALESCE(midtrans_fraud_status, ''),
			COALESCE(midtrans_transaction_time, ''),
			created_at,
			updated_at
		FROM payments
		WHERE ($1 = '' OR status = $1)
		ORDER BY created_at DESC
	`, status)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}
	defer rows.Close()

	payments := make([]model.Payment, 0)
	for rows.Next() {
		var p model.Payment
		if err := rows.Scan(
			&p.ID,
			&p.BookingID,
			&p.Amount,
			&p.Status,
			&p.TransactionID,
			&p.MidtransOrderID,
			&p.SnapToken,
			&p.RedirectURL,
			&p.PaymentType,
			&p.RawNotification,
			&p.TransactionStatus,
			&p.MidtransStatusCode,
			&p.MidtransGrossAmount,
			&p.MidtransFraudStatus,
			&p.MidtransTransactionAt,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan payment row: %w", err)
		}
		payments = append(payments, p)
	}
	return payments, rows.Err()
}

func (r *postgresRepository) GetByMidtransOrderID(ctx context.Context, orderID string) (*model.Payment, error) {
	query := `
		SELECT
			id::text,
			booking_id::text,
			amount::double precision,
			status,
			COALESCE(transaction_id, ''),
			COALESCE(midtrans_order_id, ''),
			COALESCE(snap_token, ''),
			COALESCE(redirect_url, ''),
			COALESCE(payment_type, ''),
			COALESCE(raw_notification, '{}'::jsonb),
			created_at,
			updated_at
		FROM payments
		WHERE midtrans_order_id = $1
	`
	return r.getByQuery(ctx, query, orderID)
}

func (r *postgresRepository) MarkWebhookProcessed(ctx context.Context, key string) (bool, error) {
	query := `
		INSERT INTO processed_webhooks (message_key, created_at)
		VALUES ($1, NOW())
		ON CONFLICT (message_key) DO NOTHING
	`
	tag, err := r.q(ctx).Exec(ctx, query, key)
	if err != nil {
		return false, fmt.Errorf("failed to mark webhook processed: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ErrPaymentNotFound dikembalikan bila webhook merujuk order_id yang tidak
// ada di tabel payments. Pemanggil memeriksanya dengan errors.Is, bukan
// dengan mencocokkan teks error.
var ErrPaymentNotFound = errors.New("payment tidak ditemukan")

// ApplyWebhookResult tidak lagi membuka transaksinya sendiri: service
// membungkusnya bersama insert outbox dan catatan processed_webhooks
// dalam satu Transact, sehingga tidak ada lagi celah "payment terupdate
// tapi event-nya belum tertulis".
func (r *postgresRepository) ApplyWebhookResult(ctx context.Context, notification model.MidtransNotification, status string) (*model.Payment, error) {
	rawPayload, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to encode webhook payload: %w", err)
	}

	updateQuery := `
		UPDATE payments
		SET
			status = $1,
			transaction_id = $2,
			payment_type = $3,
			raw_notification = $4,
			transaction_status = $5,
			midtrans_status_code = $6,
			midtrans_gross_amount = $7,
			midtrans_fraud_status = $8,
			midtrans_transaction_time = $9,
			updated_at = NOW()
		WHERE midtrans_order_id = $10
		RETURNING
			id::text,
			booking_id::text,
			amount::double precision,
			status,
			COALESCE(transaction_id, ''),
			COALESCE(midtrans_order_id, ''),
			COALESCE(snap_token, ''),
			COALESCE(redirect_url, ''),
			COALESCE(payment_type, ''),
			COALESCE(raw_notification, '{}'::jsonb),
			created_at,
			updated_at
	`
	var p model.Payment
	if err := r.q(ctx).QueryRow(
		ctx,
		updateQuery,
		status,
		notification.TransactionID,
		notification.PaymentType,
		rawPayload,
		notification.TransactionStatus,
		notification.StatusCode,
		notification.GrossAmount,
		notification.FraudStatus,
		notification.TransactionTime,
		notification.OrderID,
	).Scan(
		&p.ID,
		&p.BookingID,
		&p.Amount,
		&p.Status,
		&p.TransactionID,
		&p.MidtransOrderID,
		&p.SnapToken,
		&p.RedirectURL,
		&p.PaymentType,
		&p.RawNotification,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaymentNotFound
		}
		return nil, fmt.Errorf("failed to update payment from webhook: %w", err)
	}

	return &p, nil
}

func (r *postgresRepository) getByQuery(ctx context.Context, query string, arg string) (*model.Payment, error) {
	var p model.Payment
	err := r.q(ctx).QueryRow(ctx, query, arg).Scan(
		&p.ID,
		&p.BookingID,
		&p.Amount,
		&p.Status,
		&p.TransactionID,
		&p.MidtransOrderID,
		&p.SnapToken,
		&p.RedirectURL,
		&p.PaymentType,
		&p.RawNotification,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch payment: %w", err)
	}
	return &p, nil
}

func (r *postgresRepository) AddOutboxEvent(ctx context.Context, topic, key string, payload interface{}) error {
	return outbox.Enqueue(ctx, r.q(ctx), topic, key, payload)
}
