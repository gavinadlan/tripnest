package model

type BookingCreatedEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	Quantity    int     `json:"quantity"`
	TotalAmount float64 `json:"total_amount"`
	ExpiresAt   string  `json:"expires_at"`
}

// SetBookingID dipanggil repository di dalam transaksi, setelah id booking
// dibangkitkan Postgres tapi sebelum event ditulis ke outbox.
func (e *BookingCreatedEvent) SetBookingID(id string) { e.BookingID = id }

type PaymentProcessedEvent struct {
	PaymentID     string  `json:"payment_id"`
	BookingID     string  `json:"booking_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"` // SUCCESS, FAILED
	TransactionID string  `json:"transaction_id"`
}

type BookingExpiredEvent struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}
