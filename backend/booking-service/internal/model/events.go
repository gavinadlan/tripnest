package model

type BookingCreatedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	TotalAmount float64 `json:"total_amount"`
}

type PaymentProcessedEvent struct {
	PaymentID     string  `json:"payment_id"`
	BookingID     string  `json:"booking_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"` // SUCCESS, FAILED
	TransactionID string  `json:"transaction_id"`
}
