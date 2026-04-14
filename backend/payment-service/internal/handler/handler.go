package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/service"
)

type Handler struct {
	svc service.PaymentService
}

func NewHandler(svc service.PaymentService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("GET /payments", h.ListPayments)
	mux.HandleFunc("GET /payments/snap-token", h.GetSnapToken)

	// Support current and legacy webhook paths so Midtrans / old test data keep working.
	mux.HandleFunc("POST /payments/webhook", h.MidtransWebhook)
	mux.HandleFunc("POST /webhooks/midtrans", h.MidtransWebhook)
	mux.HandleFunc("POST /api/payments/webhook/midtrans", h.MidtransWebhook)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handler) GetSnapToken(w http.ResponseWriter, r *http.Request) {
	bookingID := strings.TrimSpace(r.URL.Query().Get("booking_id"))
	if bookingID == "" {
		http.Error(w, "booking_id is required", http.StatusBadRequest)
		return
	}

	payment, err := h.svc.GetPaymentByBookingID(r.Context(), bookingID)
	if err != nil {
		log.Printf("GetSnapToken failed for booking_id=%s: %+v", bookingID, err)
		if errors.Is(err, service.ErrBookingNotFound) {
			http.Error(w, "booking not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if payment == nil || payment.SnapToken == "" {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"token":      payment.SnapToken,
		"snap_token": payment.SnapToken,
	})
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	payments, err := h.svc.ListPayments(r.Context(), status)
	if err != nil {
		http.Error(w, "failed to list payments", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": payments})
}

func (h *Handler) MidtransWebhook(w http.ResponseWriter, r *http.Request) {
	var notification model.MidtransNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		http.Error(w, "invalid webhook payload", http.StatusBadRequest)
		return
	}

	if notification.OrderID == "" ||
		notification.StatusCode == "" ||
		notification.GrossAmount == "" ||
		notification.TransactionStatus == "" {
		http.Error(w, "missing required webhook fields", http.StatusBadRequest)
		return
	}

	log.Printf(
		"Received Midtrans webhook order_id=%s transaction_status=%s status_code=%s",
		notification.OrderID,
		notification.TransactionStatus,
		notification.StatusCode,
	)

	err := h.svc.HandleWebhook(r.Context(), notification)
	if err != nil {
		if errors.Is(err, service.ErrBookingNotFound) {
			log.Printf("Booking not found for webhook order_id=%s: %+v", notification.OrderID, err)
			http.Error(w, "booking not found", http.StatusNotFound)
			return
		}

		if isDuplicateWebhookErr(err) {
			log.Printf("Duplicate/conflicting webhook ignored order_id=%s: %+v", notification.OrderID, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":  "ok",
				"message": "already processed",
			})
			return
		}

		if isInvalidSignatureErr(err) {
			log.Printf("Rejected Midtrans webhook due to invalid signature order_id=%s", notification.OrderID)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		log.Printf("Failed to process Midtrans webhook order_id=%s: %+v", notification.OrderID, err)
		http.Error(w, "failed to process webhook", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func isInvalidSignatureErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "invalid midtrans signature")
}

func isDuplicateWebhookErr(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "conflict") ||
		strings.Contains(msg, "duplicate") ||
		strings.Contains(msg, "already processed") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "409")
}