package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository interface {
	Create(ctx context.Context, p *model.Payment) error
	GetByBookingID(ctx context.Context, bookingID string) (*model.Payment, error)
	Close()
}

type postgresRepository struct {
	db *pgxpool.Pool
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

func (r *postgresRepository) Create(ctx context.Context, p *model.Payment) error {
	query := `
        INSERT INTO payments (booking_id, amount, status, transaction_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $5)
        RETURNING id
    `
	p.CreatedAt = time.Now()
	err := r.db.QueryRow(ctx, query, p.BookingID, p.Amount, p.Status, p.TransactionID, p.CreatedAt).Scan(&p.ID)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetByBookingID(ctx context.Context, bookingID string) (*model.Payment, error) {
	// Implement check for idempotency (if needed later)
	return nil, nil // Placeholder
}
