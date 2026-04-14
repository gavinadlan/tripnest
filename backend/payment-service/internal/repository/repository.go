package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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
	FetchPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, id string) error
	MarkOutboxEventFailed(ctx context.Context, id string, errMsg string) error
	MarkMessageProcessed(ctx context.Context, consumerGroup, topic, messageKey string) (bool, error)
	Close()
}

type postgresRepository struct {
	db *pgxpool.Pool
}

type OutboxEvent struct {
	ID      string
	Topic   string
	Key     string
	Payload json.RawMessage
}

func NewPostgresRepository(connString string) (PaymentRepository, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}
	return &postgresRepository{db: pool}, nil
}

func (r *postgresRepository) Close() {
	r.db.Close()
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
	err := r.db.QueryRow(
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
	rows, err := r.db.Query(ctx, `
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
	tag, err := r.db.Exec(ctx, query, key)
	if err != nil {
		return false, fmt.Errorf("failed to mark webhook processed: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *postgresRepository) ApplyWebhookResult(ctx context.Context, notification model.MidtransNotification, status string) (*model.Payment, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}
	defer tx.Rollback(ctx)

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
	if err := tx.QueryRow(
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
		return nil, fmt.Errorf("failed to update payment from webhook: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit tx: %w", err)
	}

	return &p, nil
}

func (r *postgresRepository) getByQuery(ctx context.Context, query string, arg string) (*model.Payment, error) {
	var p model.Payment
	err := r.db.QueryRow(ctx, query, arg).Scan(
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
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO outbox_events (topic, message_key, payload, status, retry_count, created_at, updated_at)
		VALUES ($1, $2, $3, 'PENDING', 0, NOW(), NOW())
	`, topic, key, data)
	return err
}

func (r *postgresRepository) FetchPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, topic, message_key, payload
		FROM outbox_events
		WHERE status = 'PENDING'
		ORDER BY created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]OutboxEvent, 0)
	for rows.Next() {
		var evt OutboxEvent
		if err := rows.Scan(&evt.ID, &evt.Topic, &evt.Key, &evt.Payload); err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, rows.Err()
}

func (r *postgresRepository) MarkOutboxEventPublished(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE outbox_events
		SET status = 'PUBLISHED', published_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id)
	return err
}

func (r *postgresRepository) MarkOutboxEventFailed(ctx context.Context, id string, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE outbox_events
		SET retry_count = retry_count + 1, last_error = $2, updated_at = NOW()
		WHERE id = $1
	`, id, errMsg)
	return err
}

func (r *postgresRepository) MarkMessageProcessed(ctx context.Context, consumerGroup, topic, messageKey string) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		INSERT INTO processed_messages (consumer_group, topic, message_key, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (consumer_group, topic, message_key) DO NOTHING
	`, consumerGroup, topic, messageKey)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}
