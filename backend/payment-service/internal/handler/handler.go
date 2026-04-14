package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

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
	mux.HandleFunc("POST /webhooks/midtrans", h.MidtransWebhook)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func (h *Handler) GetSnapToken(w http.ResponseWriter, r *http.Request) {
	bookingID := r.URL.Query().Get("booking_id")
	if bookingID == "" {
		http.Error(w, "booking_id is required", http.StatusBadRequest)
		return
	}

	payment, err := h.svc.GetPaymentByBookingID(r.Context(), bookingID)
	if err != nil {
		log.Printf("GetSnapToken failed for booking_id=%s: %v", bookingID, err)
		if errors.Is(err, service.ErrBookingNotFound) {
			http.Error(w, "booking not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to fetch payment", http.StatusInternalServerError)
		return
	}
	if payment == nil || payment.SnapToken == "" {
		http.Error(w, "payment not found", http.StatusNotFound)
		return
	}

	response := map[string]string{
		"token":      payment.SnapToken,
		"snap_token": payment.SnapToken,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
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
	if notification.OrderID == "" || notification.StatusCode == "" || notification.GrossAmount == "" || notification.TransactionStatus == "" {
		http.Error(w, "missing required webhook fields", http.StatusBadRequest)
		return
	}
	log.Printf("Received Midtrans webhook order_id=%s transaction_status=%s", notification.OrderID, notification.TransactionStatus)

	if err := h.svc.HandleWebhook(r.Context(), notification); err != nil {
		if err.Error() == "invalid midtrans signature" {
			log.Printf("Rejected Midtrans webhook due to invalid signature order_id=%s", notification.OrderID)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		log.Printf("Failed to process Midtrans webhook order_id=%s: %v", notification.OrderID, err)
		http.Error(w, "failed to process webhook", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
