package service

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
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
	if strings.TrimSpace(event.BookingID) == "" {
		return fmt.Errorf("invalid booking_id")
	}

	existing, err := s.repo.GetByBookingID(ctx, event.BookingID)
	if err != nil {
		return err
	}
	log.Printf(
		"Existing payment check booking_id=%s exists=%t has_snap_token=%t",
		event.BookingID,
		existing != nil,
		existing != nil && existing.SnapToken != "",
	)
	if existing != nil && existing.SnapToken != "" {
		log.Printf("Payment already initialized for booking %s, skipping", event.BookingID)
		return nil
	}

	orderID := buildOrderID(event.BookingID)
	requestPayload := midtrans.CreateSnapTransactionRequest{
		OrderID:     orderID,
		GrossAmount: int64(math.Round(event.TotalAmount)),
		CustomerDetails: midtrans.CustomerDetails{
			FirstName: "TripNest",
			LastName:  "Customer",
			Email:     "user@tripnest.com",
			Phone:     "08123456789",
		},
	}
	log.Printf("MIDTRANS PAYLOAD: %+v", requestPayload)

	snapResp, err := s.midtransClient.CreateSnapTransaction(ctx, requestPayload)
	if err != nil {
		log.Printf("MIDTRANS ERROR: %+v", err)
		return fmt.Errorf("midtrans create transaction failed: %w", err)
	}
	if strings.TrimSpace(snapResp.Token) == "" {
		return fmt.Errorf("midtrans returned empty snap token")
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
		log.Printf("DB ERROR failed to create payment record booking_id=%s: %+v", event.BookingID, err)
		return fmt.Errorf("failed to save payment: %w", err)
	}
	log.Printf("Snap transaction created for booking %s order_id=%s", event.BookingID, orderID)

	return nil
}

func (s *paymentService) GetPaymentByBookingID(ctx context.Context, bookingID string) (*model.Payment, error) {
	if strings.TrimSpace(bookingID) == "" {
		return nil, fmt.Errorf("booking_id is required")
	}

	payment, err := s.repo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment by booking id: %w", err)
	}
	log.Printf(
		"GetPaymentByBookingID existing payment booking_id=%s exists=%t has_snap_token=%t",
		bookingID,
		payment != nil,
		payment != nil && payment.SnapToken != "",
	)
	if payment != nil && payment.SnapToken != "" {
		log.Printf("Returning existing snap token for booking_id=%s", bookingID)
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

	payment, err = s.repo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created payment by booking id: %w", err)
	}
	return payment, nil
}

func (s *paymentService) ListPayments(ctx context.Context, status string) ([]model.Payment, error) {
	return s.repo.List(ctx, status)
}

func (s *paymentService) HandleWebhook(ctx context.Context, notification model.MidtransNotification) error {
	if !s.isSignatureValid(notification) {
		return fmt.Errorf("invalid midtrans signature")
	}

	// Ignore notifications that are clearly not from our TripNest flow.
	// This avoids 500s from old/test notifications in Midtrans history.
	if !strings.HasPrefix(notification.OrderID, "ORD-") {
		log.Printf("Ignoring unsupported order_id format: %s", notification.OrderID)
		return nil
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
		log.Printf(
			"Webhook status does not emit event order_id=%s transaction_status=%s",
			notification.OrderID,
			notification.TransactionStatus,
		)
		return nil
	}

	log.Printf(
		"Applying webhook result order_id=%s transaction_status=%s mapped_status=%s",
		notification.OrderID,
		notification.TransactionStatus,
		status,
	)

	payment, err := s.repo.ApplyWebhookResult(ctx, notification, status)
	if err != nil {
		if isIgnorableWebhookError(err) {
			log.Printf("Ignoring webhook processing error for order_id=%s: %+v", notification.OrderID, err)
			return nil
		}
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
		if isIgnorableWebhookError(err) {
			log.Printf("Ignoring outbox duplicate/conflict for booking_id=%s: %+v", payment.BookingID, err)
			return nil
		}
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
	bookingURL := s.bookingServiceURL + "/bookings/" + bookingID
	log.Printf("Fetching booking booking_id=%s url=%s", bookingID, bookingURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bookingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build booking request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("booking service unreachable at %s: %w", s.bookingServiceURL, err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed reading booking response: %w", readErr)
	}
	log.Printf("Booking fetch response booking_id=%s status=%d body=%s", bookingID, resp.StatusCode, string(bodyBytes))

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to fetch booking %s: status %d body=%s", bookingID, resp.StatusCode, string(bodyBytes))
	}

	var booking bookingSnapshot
	if err := json.Unmarshal(bodyBytes, &booking); err != nil {
		return nil, fmt.Errorf("failed to decode booking payload: %w", err)
	}
	if booking.ID == "" {
		return nil, nil
	}

	return &booking, nil
}

func buildOrderID(bookingID string) string {
	shortID := bookingID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("ORD-%s-%d", shortID, time.Now().Unix())
}

func isIgnorableWebhookError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "conflict") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "already processed") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "404")
}