package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/common/outbox"
	"github.com/gavinadlan/tripnest/backend/common/pgtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepository interface {
	// CreateWithEvent menyimpan booking dan event outbox-nya dalam SATU
	// transaksi. Sebelumnya insert outbox dijalankan di goroutine terpisah
	// setelah commit, sehingga booking bisa tersimpan tanpa event -- itu
	// membatalkan seluruh gunanya pola outbox.
	CreateWithEvent(ctx context.Context, booking *model.Booking, topic string, payload interface{}) error
	GetByID(ctx context.Context, id string) (*model.Booking, error)
	List(ctx context.Context, filter ListFilter) ([]model.Booking, error)
	UpdateStatusIfCurrent(ctx context.Context, id, currentStatus, nextStatus string) (bool, error)
	ExpirePendingBookings(ctx context.Context, now time.Time) ([]string, error)
	AddOutboxEvent(ctx context.Context, topic, key string, payload interface{}) error
	// CancelStuckSeatBookings membatalkan booking yang tertinggal di
	// PENDING_SEAT melewati batas waktu penahanan kursi.
	CancelStuckSeatBookings(ctx context.Context, cutoff time.Time) ([]string, error)
	// Transact menjalankan fn dalam satu transaksi. Semua method di atas
	// yang dipanggil dengan ctx milik fn ikut di dalamnya.
	Transact(ctx context.Context, fn func(ctx context.Context) error) error
}

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) BookingRepository {
	return &postgresRepository{db: pool}
}

func (r *postgresRepository) Transact(ctx context.Context, fn func(ctx context.Context) error) error {
	return pgtx.Transact(ctx, r.db, fn)
}

// q mengembalikan transaksi yang dibawa ctx, atau pool bila tidak ada.
func (r *postgresRepository) q(ctx context.Context) pgtx.Querier {
	return pgtx.From(ctx, r.db)
}

func (r *postgresRepository) CreateWithEvent(ctx context.Context, b *model.Booking, topic string, payload interface{}) error {
	return r.Transact(ctx, func(ctx context.Context) error {
		return r.createWithEvent(ctx, b, topic, payload)
	})
}

func (r *postgresRepository) createWithEvent(ctx context.Context, b *model.Booking, topic string, payload interface{}) error {
	tx := r.q(ctx)

	insertBooking := `
		INSERT INTO bookings (user_id, resource_id, quantity, total_amount, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at, expires_at`

	// Status datang dari pemanggil, tidak lagi dipaku di sini. Sejak ada
	// PENDING_SEAT, keadaan awal booking adalah keputusan service --
	// menimpanya dengan PENDING_PAYMENT di lapisan ini justru melewati
	// seluruh penjagaan kursi.
	if err := tx.QueryRow(ctx, insertBooking,
		b.UserID,
		b.ResourceID,
		b.Quantity,
		b.TotalAmount,
		b.Status,
		b.ExpiresAt,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt, &b.ExpiresAt); err != nil {
		return fmt.Errorf("failed to create booking: %w", err)
	}

	// Payload event butuh id booking, yang baru diketahui setelah insert
	// di atas. Karena keduanya berada dalam transaksi yang sama, event
	// hanya ada bila booking-nya benar-benar tersimpan, dan sebaliknya.
	if setter, ok := payload.(interface{ SetBookingID(string) }); ok {
		setter.SetBookingID(b.ID)
	}

	return outbox.Enqueue(ctx, tx, topic, b.ID, payload)
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*model.Booking, error) {
	query := `
		SELECT id, user_id, resource_id, quantity, total_amount, status, expires_at, created_at, updated_at
		FROM bookings
		WHERE id = $1
	`
	var booking model.Booking
	err := r.q(ctx).QueryRow(ctx, query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.ResourceID,
		&booking.Quantity,
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

// ListFilter membatasi hasil List. UserID yang tidak kosong membatasi
// hasil hanya pada booking milik user tersebut -- handler mengisinya dari
// JWT untuk pemanggil non-admin, sehingga endpoint yang sama aman dipakai
// user biasa maupun admin.
type ListFilter struct {
	UserID      string
	Status      string
	CreatedDate string
	Limit       int
	Offset      int
}

func (r *postgresRepository) List(ctx context.Context, filter ListFilter) ([]model.Booking, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	query := `
		SELECT id, user_id, resource_id, quantity, total_amount, status, expires_at, created_at, updated_at
		FROM bookings
		WHERE ($1 = '' OR user_id = $1::uuid)
		  AND ($2 = '' OR status = $2)
		  AND ($3 = '' OR DATE(created_at) = $3::date)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.q(ctx).Query(ctx, query, filter.UserID, filter.Status, filter.CreatedDate, filter.Limit, filter.Offset)
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
			&booking.Quantity,
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
	tag, err := r.q(ctx).Exec(ctx, query, nextStatus, id, currentStatus)
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
	rows, err := r.q(ctx).Query(ctx, query, model.BookingStatusExpired, model.BookingStatusPendingPayment, now)
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

// CancelStuckSeatBookings memakai created_at, bukan expires_at: batas
// waktu menunggu kursi diukur dalam detik, sementara expires_at adalah
// tenggat pembayaran yang panjangnya belasan menit.
func (r *postgresRepository) CancelStuckSeatBookings(ctx context.Context, cutoff time.Time) ([]string, error) {
	query := `
		UPDATE bookings
		SET status = $1, updated_at = NOW()
		WHERE status = $2 AND created_at <= $3
		RETURNING id
	`
	rows, err := r.q(ctx).Query(ctx, query, model.BookingStatusCancelled, model.BookingStatusPendingSeat, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stuckIDs := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		stuckIDs = append(stuckIDs, id)
	}
	return stuckIDs, rows.Err()
}

func (r *postgresRepository) AddOutboxEvent(ctx context.Context, topic, key string, payload interface{}) error {
	return outbox.Enqueue(ctx, r.q(ctx), topic, key, payload)
}
