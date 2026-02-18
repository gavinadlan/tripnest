package repository

import (
	"context"
	"fmt"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository interface {
	Create(ctx context.Context, booking *model.Booking) error
	GetByID(ctx context.Context, id string) (*model.Booking, error)
	GetByUserID(ctx context.Context, userID string) ([]model.Booking, error)
	UpdateStatus(ctx context.Context, id, status string) error
	Close()
}

type postgresRepository struct {
	db *pgxpool.Pool
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
		INSERT INTO bookings (user_id, resource_id, total_amount, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		b.UserID,
		b.ResourceID,
		b.TotalAmount,
		"PENDING", // Default status
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}
	b.Status = "PENDING"
	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	// Implementation needed
	return nil, nil
}

func (r *postgresRepository) GetByUserID(ctx context.Context, userID string) ([]model.Booking, error) {
	// Implementation needed
	return nil, nil
}

func (r *postgresRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE bookings SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}
