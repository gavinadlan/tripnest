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
	TotalAmount float64 `json:"total_amount"`
	ExpiresAt   string  `json:"expires_at"`
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

type InventoryStatusEvent struct {
	BookingID  string `json:"booking_id"`
	ResourceID string `json:"resource_id"`
	Status     string `json:"status"`
}

type UpsertInventoryRequest struct {
	ResourceID string `json:"resource_id"`
	TotalSlots int    `json:"total_slots"`
}
