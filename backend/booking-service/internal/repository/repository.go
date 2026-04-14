package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository interface {
	Create(ctx context.Context, booking *model.Booking) error
	GetByID(ctx context.Context, id string) (*model.Booking, error)
	GetByUserID(ctx context.Context, userID string) ([]model.Booking, error)
	List(ctx context.Context, status, createdDate string) ([]model.Booking, error)
	UpdateStatusIfCurrent(ctx context.Context, id, currentStatus, nextStatus string) (bool, error)
	ExpirePendingBookings(ctx context.Context, now time.Time) ([]string, error)
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

func NewPostgresRepository(connString string) (BookingRepository, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %v", err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	return &postgresRepository{db: pool}, nil
}

func (r *postgresRepository) Close() {
	r.db.Close()
}

func (r *postgresRepository) Create(ctx context.Context, b *model.Booking) error {
	query := `
		INSERT INTO bookings (user_id, resource_id, total_amount, status, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at, expires_at`

	err := r.db.QueryRow(ctx, query,
		b.UserID,
		b.ResourceID,
		b.TotalAmount,
		model.BookingStatusPendingPayment,
		b.ExpiresAt,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt, &b.ExpiresAt)

	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}
	b.Status = model.BookingStatusPendingPayment
	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	query := `
		SELECT id, user_id, resource_id, total_amount, status, expires_at, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`
	var booking model.Booking
	err := r.db.QueryRow(ctx, query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.ResourceID,
		&booking.TotalAmount,
		&booking.Status,
		&booking.ExpiresAt,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &booking, nil
}

func (r *postgresRepository) GetByUserID(ctx context.Context, userID string) ([]model.Booking, error) {
	query := `
		SELECT id, user_id, resource_id, total_amount, status, expires_at, created_at, updated_at
		FROM bookings
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		var booking model.Booking
		if err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.ResourceID,
			&booking.TotalAmount,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func (r *postgresRepository) List(ctx context.Context, status, createdDate string) ([]model.Booking, error) {
	query := `
		SELECT id, user_id, resource_id, total_amount, status, expires_at, created_at, updated_at
		FROM bookings
		WHERE ($1 = '' OR status = $1)
		  AND ($2 = '' OR DATE(created_at) = $2::date)
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, status, createdDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bookings := make([]model.Booking, 0)
	for rows.Next() {
		var booking model.Booking
		if err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.ResourceID,
			&booking.TotalAmount,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		); err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

func (r *postgresRepository) UpdateStatusIfCurrent(ctx context.Context, id, currentStatus, nextStatus string) (bool, error) {
	query := `
		UPDATE bookings
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND status = $3
	`
	tag, err := r.db.Exec(ctx, query, nextStatus, id, currentStatus)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *postgresRepository) ExpirePendingBookings(ctx context.Context, now time.Time) ([]string, error) {
	query := `
		UPDATE bookings
		SET status = $1, updated_at = NOW()
		WHERE status = $2 AND expires_at <= $3
		RETURNING id
	`
	rows, err := r.db.Query(ctx, query, model.BookingStatusExpired, model.BookingStatusPendingPayment, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	expiredIDs := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		expiredIDs = append(expiredIDs, id)
	}
	return expiredIDs, rows.Err()
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
