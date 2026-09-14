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

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/midtrans"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/msgbroker"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/repository"
)

var (
	ErrBookingNotFound   = errors.New("booking tidak ditemukan")
	ErrPaymentNotReady   = errors.New("pembayaran belum disiapkan")
	ErrForbidden         = errors.New("booking ini bukan milik Anda")
	ErrBookingNotPayable = errors.New("booking tidak bisa dibayar pada status ini")
	ErrInvalidSignature  = errors.New("signature Midtrans tidak valid")
)

// Caller membawa identitas pemanggil dari klaim JWT ke lapisan service,
// supaya pengecekan kepemilikan tidak bergantung pada nilai yang dikirim
// client di query string.
type Caller struct {
	UserID  string
	IsAdmin bool
}

func (c Caller) mayAccess(ownerID string) bool {
	return c.IsAdmin || (c.UserID != "" && c.UserID == ownerID)
}

type PaymentService interface {
	ProcessPayment(ctx context.Context, bookingEvent model.BookingEvent) error
	// GetSnapToken hanya membaca. Endpoint GET tidak boleh punya efek
	// samping: versi lama diam-diam membuat transaksi Midtrans baru,
	// sehingga siapa pun bisa menguras kuota hanya dengan menebak id.
	GetSnapToken(ctx context.Context, caller Caller, bookingID string) (*model.Payment, error)
	// InitiatePayment membuat transaksi Snap bila belum ada. Idempoten.
	InitiatePayment(ctx context.Context, caller Caller, bookingID string) (*model.Payment, error)
	ListPayments(ctx context.Context, status string) ([]model.Payment, error)
	HandleWebhook(ctx context.Context, notification model.MidtransNotification) error
}

type paymentService struct {
	repo              repository.PaymentRepository
	midtransClient    *midtrans.Client
	midtransServerKey string
	bookingServiceURL string
	verifier          *auth.Verifier
	httpClient        *http.Client
}

func NewPaymentService(
	repo repository.PaymentRepository,
	_ *msgbroker.Producer,
	midtransClient *midtrans.Client,
	midtransServerKey, bookingServiceURL string,
	verifier *auth.Verifier,
) PaymentService {
	return &paymentService{
		repo:              repo,
		midtransClient:    midtransClient,
		midtransServerKey: midtransServerKey,
		bookingServiceURL: strings.TrimRight(bookingServiceURL, "/"),
		verifier:          verifier,
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
		// TODO(sprint-3): ambil nama dan email asli dari user-service.
		// Sampai itu ada, struk Midtrans memakai placeholder.
		CustomerDetails: midtrans.CustomerDetails{
			FirstName: "TripNest",
			LastName:  "Customer",
			Email:     "user@tripnest.com",
			Phone:     "08123456789",
		},
		ExpiresAt: event.ExpiresAt,
		ItemName:  "TripNest booking " + event.ResourceID,
	}
	log.Printf("Membuat transaksi Snap booking_id=%s order_id=%s", event.BookingID, orderID)

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

func (s *paymentService) GetSnapToken(ctx context.Context, caller Caller, bookingID string) (*model.Payment, error) {
	if strings.TrimSpace(bookingID) == "" {
		return nil, fmt.Errorf("booking_id wajib diisi")
	}

	// Kepemilikan diperiksa lewat booking, bukan lewat payment: kalau
	// payment belum ada pun pemanggil tetap tidak boleh tahu apa pun
	// tentang booking milik orang lain.
	booking, err := s.fetchBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}
	if !caller.mayAccess(booking.UserID) {
		return nil, ErrForbidden
	}

	// Status diperiksa di sini juga, bukan hanya saat token dibuat.
	// Token yang sudah terbit tetap valid di sisi Midtrans, jadi tanpa
	// pemeriksaan ini booking yang dibatalkan beberapa detik setelah
	// dibuat masih bisa dibuka halaman pembayarannya dan dibayar.
	if err := ensurePayable(booking); err != nil {
		return nil, err
	}

	payment, err := s.repo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch payment by booking id: %w", err)
	}
	if payment == nil || payment.SnapToken == "" {
		return nil, ErrPaymentNotReady
	}
	return payment, nil
}

// ensurePayable menolak booking yang sudah kedaluwarsa, dibatalkan, atau
// sudah lunas. Uang yang masuk untuk booking seperti itu tidak bisa
// dipakai: ConfirmBooking hanya bertransisi dari PENDING_PAYMENT.
func ensurePayable(booking *bookingSnapshot) error {
	if booking.Status != bookingStatusPendingPayment {
		return fmt.Errorf("%w: status %s", ErrBookingNotPayable, booking.Status)
	}
	if !booking.ExpiresAt.IsZero() && time.Now().After(booking.ExpiresAt) {
		return fmt.Errorf("%w: masa pembayaran sudah lewat", ErrBookingNotPayable)
	}
	return nil
}

func (s *paymentService) InitiatePayment(ctx context.Context, caller Caller, bookingID string) (*model.Payment, error) {
	if strings.TrimSpace(bookingID) == "" {
		return nil, fmt.Errorf("booking_id wajib diisi")
	}

	booking, err := s.fetchBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	if booking == nil {
		return nil, ErrBookingNotFound
	}
	if !caller.mayAccess(booking.UserID) {
		return nil, ErrForbidden
	}

	if err := ensurePayable(booking); err != nil {
		return nil, err
	}

	if err := s.ProcessPayment(ctx, model.BookingEvent{
		BookingID:   booking.ID,
		UserID:      booking.UserID,
		ResourceID:  booking.ResourceID,
		TotalAmount: booking.TotalAmount,
		ExpiresAt:   booking.ExpiresAt,
		Type:        "created",
	}); err != nil {
		return nil, err
	}

	payment, err := s.repo.GetByBookingID(ctx, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created payment by booking id: %w", err)
	}
	if payment == nil || payment.SnapToken == "" {
		return nil, ErrPaymentNotReady
	}
	return payment, nil
}

func (s *paymentService) ListPayments(ctx context.Context, status string) ([]model.Payment, error) {
	return s.repo.List(ctx, status)
}

// errDuplicateWebhook menandai notifikasi yang sudah pernah diproses.
// Hanya dipakai di dalam HandleWebhook untuk memicu rollback.
var errDuplicateWebhook = errors.New("webhook duplikat")

func (s *paymentService) HandleWebhook(ctx context.Context, notification model.MidtransNotification) error {
	if !s.isSignatureValid(notification) {
		return ErrInvalidSignature
	}

	// Ignore notifications that are clearly not from our TripNest flow.
	// This avoids 500s from old/test notifications in Midtrans history.
	if !strings.HasPrefix(notification.OrderID, "ORD-") {
		log.Printf("Ignoring unsupported order_id format: %s", notification.OrderID)
		return nil
	}

	// Update payment, event outbox, dan catatan idempotensi ditulis dalam
	// SATU transaksi. Catatan ditulis paling akhir: kalau ternyata sudah
	// ada, seluruh transaksi di-rollback dan tidak ada efek samping yang
	// tertinggal. Versi lama menandai lebih dulu -- webhook yang gagal di
	// tengah tidak pernah bisa diulang Midtrans karena sudah dianggap
	// selesai.
	idempotencyKey := notification.OrderID + ":" + notification.TransactionStatus
	err := s.repo.Transact(ctx, func(ctx context.Context) error {
		if err := s.applyWebhook(ctx, notification); err != nil {
			return err
		}
		inserted, err := s.repo.MarkWebhookProcessed(ctx, idempotencyKey)
		if err != nil {
			return err
		}
		if !inserted {
			return errDuplicateWebhook
		}
		return nil
	})
	if errors.Is(err, errDuplicateWebhook) {
		log.Printf("Duplicate webhook ignored idempotency_key=%s", idempotencyKey)
		return nil
	}
	return err
}

// applyWebhook menjalankan efek bisnis satu notifikasi. Dipanggil di
// dalam transaksi HandleWebhook.
func (s *paymentService) applyWebhook(ctx context.Context, notification model.MidtransNotification) error {
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
		if errors.Is(err, repository.ErrPaymentNotFound) {
			// order_id berawalan ORD- tapi tidak ada di tabel payments.
			// Ini anomali nyata, bukan sesuatu yang boleh ditelan diam-diam:
			// versi lama membalas 200 dan pembayarannya hilang tanpa jejak.
			log.Printf("ANOMALI: webhook untuk order_id=%s tidak punya baris payment", notification.OrderID)
			return fmt.Errorf("payment untuk order_id %s tidak ditemukan: %w", notification.OrderID, err)
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

const bookingStatusPendingPayment = "PENDING_PAYMENT"

type bookingSnapshot struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	ResourceID  string    `json:"resource_id"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (s *paymentService) fetchBooking(ctx context.Context, bookingID string) (*bookingSnapshot, error) {
	// Endpoint booking sekarang butuh autentikasi, jadi panggilan
	// antar-service memakai token berumur pendek dengan role "service".
	bookingURL := s.bookingServiceURL + "/internal/bookings/" + bookingID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, bookingURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build booking request: %w", err)
	}

	token, err := s.verifier.ServiceToken("payment-service")
	if err != nil {
		return nil, fmt.Errorf("failed to mint service token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("booking service unreachable at %s: %w", s.bookingServiceURL, err)
	}
	defer resp.Body.Close()

	bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return nil, fmt.Errorf("failed reading booking response: %w", readErr)
	}
	// Jangan mencatat body booking utuh: isinya data pesanan pengguna.
	log.Printf("Booking fetch booking_id=%s status=%d", bookingID, resp.StatusCode)

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("failed to fetch booking %s: status %d", bookingID, resp.StatusCode)
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
