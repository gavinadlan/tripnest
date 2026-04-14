package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gavinadlan/tripnest/backend/payment-service/internal/midtrans"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/repository"
)

var ErrBookingNotFound = errors.New("booking not found")

type PaymentService interface {
	ProcessPayment(ctx context.Context, bookingEvent model.BookingEvent) error
	GetPaymentByBookingID(ctx context.Context, bookingID string) (*model.Payment, error)
	ListPayments(ctx context.Context, status string) ([]model.Payment, error)
	HandleWebhook(ctx context.Context, notification model.MidtransNotification) error
}

type paymentService struct {
	repo              repository.PaymentRepository
	midtransClient    *midtrans.Client
	midtransServerKey string
	bookingServiceURL string
	httpClient        *http.Client
}

func NewPaymentService(repo repository.PaymentRepository, _ *msgbroker.Producer, midtransClient *midtrans.Client, midtransServerKey, bookingServiceURL string) PaymentService {
	return &paymentService{
		repo:              repo,
		midtransClient:    midtransClient,
		midtransServerKey: midtransServerKey,
		bookingServiceURL: strings.TrimRight(bookingServiceURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *paymentService) ProcessPayment(ctx context.Context, event model.BookingEvent) error {
	log.Printf("Processing payment for booking %s amount %.2f", event.BookingID, event.TotalAmount)

	existing, err := s.repo.GetByBookingID(ctx, event.BookingID)
	if err != nil {
		return err
	}
	if existing != nil && existing.SnapToken != "" {
		log.Printf("Payment already initialized for booking %s, skipping", event.BookingID)
		return nil
	}

	orderID := event.BookingID
	snapResp, err := s.midtransClient.CreateSnapTransaction(ctx, midtrans.CreateSnapTransactionRequest{
		OrderID:     orderID,
		GrossAmount: event.TotalAmount,
	})
	if err != nil {
		log.Printf("Failed to create snap transaction: %v", err)
		return err
	}

	payment := &model.Payment{
		BookingID:       event.BookingID,
		Amount:          event.TotalAmount,
		Status:          "PENDING",
		TransactionID:   "",
		MidtransOrderID: orderID,
		SnapToken:       snapResp.Token,
		RedirectURL:     snapResp.RedirectURL,
	}

	if err := s.repo.UpsertInitiated(ctx, payment); err != nil {
		log.Printf("Failed to create payment record: %v", err)
		return err
	}
	log.Printf("Snap transaction created for booking %s order_id=%s", event.BookingID, orderID)

	return nil
}

func (s *paymentService) GetPaymentByBookingID(ctx context.Context, bookingID string) (*model.Payment, error) {
	payment, err := s.repo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if payment != nil && payment.SnapToken != "" {
		return payment, nil
	}

	booking, err := s.fetchBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}

	if err := s.ProcessPayment(ctx, model.BookingEvent{
		BookingID:   booking.ID,
		UserID:      booking.UserID,
		ResourceID:  booking.ResourceID,
		TotalAmount: booking.TotalAmount,
		Type:        "created",
	}); err != nil {
		return nil, err
	}

	return s.repo.GetByBookingID(ctx, bookingID)
}

func (s *paymentService) ListPayments(ctx context.Context, status string) ([]model.Payment, error) {
	return s.repo.List(ctx, status)
}

func (s *paymentService) HandleWebhook(ctx context.Context, notification model.MidtransNotification) error {
	if !s.isSignatureValid(notification) {
		return fmt.Errorf("invalid midtrans signature")
	}

	idempotencyKey := notification.OrderID + ":" + notification.TransactionStatus
	inserted, err := s.repo.MarkWebhookProcessed(ctx, idempotencyKey)
	if err != nil {
		return err
	}
	if !inserted {
		log.Printf("Duplicate webhook ignored idempotency_key=%s", idempotencyKey)
		return nil
	}

	status, topic := mapMidtransStatus(notification)
	if topic == "" {
		log.Printf("Webhook status does not emit event order_id=%s transaction_status=%s", notification.OrderID, notification.TransactionStatus)
		return nil
	}

	payment, err := s.repo.ApplyWebhookResult(ctx, notification, status)
	if err != nil {
		return err
	}

	event := model.PaymentEvent{
		PaymentID:     payment.ID,
		BookingID:     payment.BookingID,
		Amount:        payment.Amount,
		Status:        status,
		TransactionID: payment.TransactionID,
	}

	if err := s.repo.AddOutboxEvent(ctx, topic, payment.BookingID, event); err != nil {
		return err
	}
	log.Printf("Webhook processed and enqueued outbox event topic=%s booking_id=%s", topic, payment.BookingID)
	return nil
}

func (s *paymentService) isSignatureValid(notification model.MidtransNotification) bool {
	raw := notification.OrderID + notification.StatusCode + notification.GrossAmount + s.midtransServerKey
	sum := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(sum[:])
	return strings.EqualFold(expected, notification.SignatureKey)
}

func mapMidtransStatus(notification model.MidtransNotification) (string, string) {
	switch notification.TransactionStatus {
	case "capture", "settlement":
		if strings.EqualFold(notification.FraudStatus, "accept") || notification.FraudStatus == "" {
			return "SUCCESS", "payment.success"
		}
		return "FAILED", "payment.failed"
	case "deny", "cancel", "expire", "failure":
		return "FAILED", "payment.failed"
	default:
		return "PENDING", ""
	}
}

type bookingSnapshot struct {
	ID          string  `json:"id"`
	UserID      string  `json:"user_id"`
	ResourceID  string  `json:"resource_id"`
	TotalAmount float64 `json:"total_amount"`
}

func (s *paymentService) fetchBooking(ctx context.Context, bookingID string) (*bookingSnapshot, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.bookingServiceURL+"/bookings/"+bookingID, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to fetch booking %s: status %d", bookingID, resp.StatusCode)
	}

	var booking bookingSnapshot
	if err := json.NewDecoder(resp.Body).Decode(&booking); err != nil {
		return nil, err
	}
	if booking.ID == "" {
		return nil, nil
	}
	return &booking, nil
}
