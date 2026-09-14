package model

import "time"

const (
	ReservationStatusReserved  = "RESERVED"
	ReservationStatusConfirmed = "CONFIRMED"
	ReservationStatusReleased  = "RELEASED"
)

type Inventory struct {
	ResourceID     string    `json:"resource_id" db:"resource_id"`
	TotalSlots     int       `json:"total_slots" db:"total_slots"`
	AvailableSlots int       `json:"available_slots" db:"available_slots"`
	ReservedSlots  int       `json:"reserved_slots"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

type InventoryReservation struct {
	ID         string    `json:"id" db:"id"`
	BookingID  string    `json:"booking_id" db:"booking_id"`
	ResourceID string    `json:"resource_id" db:"resource_id"`
	Status     string    `json:"status" db:"status"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type BookingCreatedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
	ExpiresAt   string  `json:"expires_at"`
}

// Seats mengembalikan jumlah kursi yang diminta booking. Event lama yang
// belum punya field quantity berarti satu kursi.
func (e BookingCreatedEvent) Seats() int {
	if e.Quantity < 1 {
		return 1
	}
	return e.Quantity
}

type BookingCancelledEvent struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}

type PaymentProcessedEvent struct {
	PaymentID     string  `json:"payment_id"`
	BookingID     string  `json:"booking_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"`
	TransactionID string  `json:"transaction_id"`
}

type BookingExpiredEvent struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}

// UpsertInventoryRequest hanya dipakai endpoint penyesuaian kapasitas.
// Pembuatan kapasitas baru tidak lagi lewat sini: kapasitas selalu lahir
// bersama trip-nya dalam satu transaksi.
type UpsertInventoryRequest struct {
	ResourceID string `json:"resource_id"`
	TotalSlots int    `json:"total_slots"`
}
