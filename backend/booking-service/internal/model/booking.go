package model

import (
	"time"
)

type Booking struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	ResourceID  string    `json:"resource_id" db:"resource_id"`
	TotalAmount float64   `json:"total_amount" db:"total_amount"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreateBookingRequest struct {
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	TotalAmount float64 `json:"total_amount"`
}

type BookingResponse struct {
	Booking
}
