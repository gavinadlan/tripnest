package model

import (
	"encoding/json"
	"time"
)

type Payment struct {
	ID                    string          `json:"id" db:"id"`
	BookingID             string          `json:"booking_id" db:"booking_id"`
	Amount                float64         `json:"amount" db:"amount"`
	Status                string          `json:"status" db:"status"`
	TransactionID         string          `json:"transaction_id" db:"transaction_id"`
	MidtransOrderID       string          `json:"midtrans_order_id" db:"midtrans_order_id"`
	SnapToken             string          `json:"snap_token" db:"snap_token"`
	RedirectURL           string          `json:"redirect_url" db:"redirect_url"`
	PaymentType           string          `json:"payment_type" db:"payment_type"`
	RawNotification       json.RawMessage `json:"raw_notification" db:"raw_notification"`
	TransactionStatus     string          `json:"transaction_status" db:"transaction_status"`
	MidtransStatusCode    string          `json:"midtrans_status_code" db:"midtrans_status_code"`
	MidtransGrossAmount   string          `json:"midtrans_gross_amount" db:"midtrans_gross_amount"`
	MidtransFraudStatus   string          `json:"midtrans_fraud_status" db:"midtrans_fraud_status"`
	MidtransTransactionAt string          `json:"midtrans_transaction_time" db:"midtrans_transaction_time"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}

type BookingEvent struct {
	BookingID   string  `json:"booking_id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	TotalAmount float64 `json:"total_amount"`
	Type        string  `json:"type"` // created, confirmed, failed
}

type PaymentEvent struct {
	PaymentID     string  `json:"payment_id"`
	BookingID     string  `json:"booking_id"`
	Amount        float64 `json:"amount"`
	Status        string  `json:"status"` // SUCCESS, FAILED
	TransactionID string  `json:"transaction_id"`
}

type MidtransNotification struct {
	OrderID           string `json:"order_id"`
	StatusCode        string `json:"status_code"`
	GrossAmount       string `json:"gross_amount"`
	SignatureKey      string `json:"signature_key"`
	TransactionStatus string `json:"transaction_status"`
	FraudStatus       string `json:"fraud_status"`
	PaymentType       string `json:"payment_type"`
	TransactionID     string `json:"transaction_id"`
	TransactionTime   string `json:"transaction_time"`
}
