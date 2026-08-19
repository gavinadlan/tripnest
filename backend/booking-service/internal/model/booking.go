package model

import (
	"time"
)

const (
	BookingStatusPendingPayment = "PENDING_PAYMENT"
	BookingStatusConfirmed      = "CONFIRMED"
	BookingStatusCancelled      = "CANCELLED"
	BookingStatusExpired        = "EXPIRED"
)

// MaxQuantity membatasi jumlah kursi per booking.
const MaxQuantity = 10

type Booking struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	ResourceID  string    `json:"resource_id" db:"resource_id"`
	Quantity    int       `json:"quantity" db:"quantity"`
	TotalAmount float64   `json:"total_amount" db:"total_amount"`
	Status      string    `json:"status" db:"status"`
	ExpiresAt   time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateBookingRequest sengaja tidak lagi punya field user_id dan
// total_amount. Identitas diambil dari JWT dan harga dari katalog;
// keduanya tidak boleh ditentukan client.
type CreateBookingRequest struct {
	ResourceID string `json:"resource_id"`
	Quantity   int    `json:"quantity"`
}
