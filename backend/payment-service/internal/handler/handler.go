package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/payment-service/internal/service"
)

type Handler struct {
	svc      service.PaymentService
	verifier *auth.Verifier
}

func NewHandler(svc service.PaymentService, verifier *auth.Verifier) *Handler {
	return &Handler{svc: svc, verifier: verifier}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)

	// Daftar pembayaran hanya untuk admin, dan hanya field yang aman.
	mux.Handle("GET /payments", h.verifier.AdminOnly(h.ListPayments))

	// GET hanya membaca; POST yang boleh membuat transaksi Snap.
	mux.Handle("GET /payments/snap-token", h.verifier.AuthedOnly(h.GetSnapToken))
	mux.Handle("POST /payments/snap-token", h.verifier.AuthedOnly(h.InitiatePayment))

	// Webhook tidak boleh pakai JWT: pemanggilnya Midtrans, bukan browser.
	// Keasliannya dijamin verifikasi signature SHA-512 di service.
	mux.HandleFunc("POST /payments/webhook", h.MidtransWebhook)
	mux.HandleFunc("POST /webhooks/midtrans", h.MidtransWebhook)
	mux.HandleFunc("POST /api/payments/webhook/midtrans", h.MidtransWebhook)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func callerFrom(r *http.Request) service.Caller {
	userID, _ := auth.UserID(r.Context())
	return service.Caller{UserID: userID, IsAdmin: auth.IsAdmin(r.Context())}
}

func (h *Handler) GetSnapToken(w http.ResponseWriter, r *http.Request) {
	bookingID := strings.TrimSpace(r.URL.Query().Get("booking_id"))
	if bookingID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("booking_id wajib diisi"))
		return
	}

	payment, err := h.svc.GetSnapToken(r.Context(), callerFrom(r), bookingID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, map[string]string{
		"token":      payment.SnapToken,
		"snap_token": payment.SnapToken,
	})
}

func (h *Handler) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BookingID string `json:"booking_id"`
	}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, errors.New("payload tidak valid: kirim {booking_id}"))
		return
	}

	bookingID := strings.TrimSpace(req.BookingID)
	if bookingID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("booking_id wajib diisi"))
		return
	}

	payment, err := h.svc.InitiatePayment(r.Context(), callerFrom(r), bookingID)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, map[string]string{
		"token":      payment.SnapToken,
		"snap_token": payment.SnapToken,
	})
}

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	payments, err := h.svc.ListPayments(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	// Summaries membuang snap_token dan raw_notification.
	_ = utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data": model.Summaries(payments),
	})
}

func (h *Handler) MidtransWebhook(w http.ResponseWriter, r *http.Request) {
	var notification model.MidtransNotification
	if err := json.NewDecoder(r.Body).Decode(&notification); err != nil {
		utils.WriteError(w, http.StatusBadRequest, errors.New("payload webhook tidak valid"))
		return
	}

	if notification.OrderID == "" ||
		notification.StatusCode == "" ||
		notification.GrossAmount == "" ||
		notification.TransactionStatus == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("field webhook tidak lengkap"))
		return
	}

	log.Printf(
		"Webhook Midtrans order_id=%s transaction_status=%s status_code=%s",
		notification.OrderID,
		notification.TransactionStatus,
		notification.StatusCode,
	)

	if err := h.svc.HandleWebhook(r.Context(), notification); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidSignature):
			log.Printf("Webhook ditolak, signature tidak valid order_id=%s", notification.OrderID)
			utils.WriteError(w, http.StatusUnauthorized, err)
		case errors.Is(err, service.ErrBookingNotFound):
			utils.WriteError(w, http.StatusNotFound, err)
		default:
			log.Printf("Gagal memproses webhook order_id=%s: %v", notification.OrderID, err)
			utils.WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrBookingNotFound),
		errors.Is(err, service.ErrForbidden):
		// Jangan bedakan "tidak ada" dari "bukan milikmu": jawaban yang
		// berbeda memberi tahu penyerang bahwa id tebakannya benar.
		utils.WriteError(w, http.StatusNotFound, errors.New("booking tidak ditemukan"))
	case errors.Is(err, service.ErrPaymentNotReady):
		utils.WriteError(w, http.StatusNotFound, err)
	case errors.Is(err, service.ErrBookingNotPayable):
		utils.WriteError(w, http.StatusConflict, err)
	default:
		utils.WriteError(w, http.StatusInternalServerError, err)
	}
}
