package model

type BookingCreatedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	TotalAmount float64 `json:"total_amount"`
}
