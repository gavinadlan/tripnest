package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/catalog"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/service"
	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.BookingService
}

func NewHandler(svc service.BookingService) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes memasang rute yang butuh autentikasi. Middleware auth
// dipasang pemanggil, bukan di sini, supaya urutannya terlihat di main.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/bookings", h.CreateBooking)
	r.Get("/bookings", h.ListBookings)
	r.Get("/bookings/{id}", h.GetBooking)
	r.Post("/bookings/{id}/cancel", h.CancelBooking)
}

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, errors.New("token tidak valid"))
		return
	}

	var req model.CreateBookingRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		// ReadJSON memakai DisallowUnknownFields, jadi payload lama yang
		// masih mengirim user_id atau total_amount akan ditolak di sini.
		// Itu memang disengaja: harga dan identitas kini milik server.
		utils.WriteError(w, http.StatusBadRequest, errors.New("payload tidak valid: kirim {resource_id, quantity}"))
		return
	}

	booking, err := h.svc.CreateBooking(r.Context(), userID, &req)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusCreated, booking)
}

func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	booking, ok := h.lookup(w, r)
	if !ok {
		return
	}

	// Balas 404, bukan 403, untuk booking milik orang lain: 403 memberi
	// tahu penyerang bahwa id yang ditebaknya benar-benar ada.
	if !canAccess(r, booking.UserID) {
		utils.WriteError(w, http.StatusNotFound, errors.New("booking tidak ditemukan"))
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, booking)
}

// GetBookingInternal melayani panggilan antar-service.
//
// Sengaja TANPA cek kepemilikan: pemanggilnya payment-service, yang
// identitasnya bukan pemilik booking mana pun. Rutenya sudah dibatasi
// RequireServiceOrAdmin di main, jadi endpoint ini tidak pernah terbuka
// untuk token user biasa. Memakai ulang GetBooking di sini adalah bug
// yang membuat setiap fetch antar-service dibalas 404.
func (h *Handler) GetBookingInternal(w http.ResponseWriter, r *http.Request) {
	booking, ok := h.lookup(w, r)
	if !ok {
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, booking)
}

// lookup mengambil booking dari path param dan menuliskan respons error
// bila gagal. Mengembalikan ok=false bila pemanggil harus berhenti.
func (h *Handler) lookup(w http.ResponseWriter, r *http.Request) (*model.Booking, bool) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("id booking wajib diisi"))
		return nil, false
	}

	booking, err := h.svc.GetBooking(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return nil, false
	}
	if booking == nil {
		utils.WriteError(w, http.StatusNotFound, errors.New("booking tidak ditemukan"))
		return nil, false
	}
	return booking, true
}

func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, errors.New("token tidak valid"))
		return
	}

	query := r.URL.Query()
	filter := repository.ListFilter{
		Status:      query.Get("status"),
		CreatedDate: query.Get("date"),
		Limit:       atoiDefault(query.Get("limit"), 50),
		Offset:      atoiDefault(query.Get("offset"), 0),
	}

	// Admin melihat semua; user biasa selalu dibatasi pada miliknya sendiri,
	// apa pun yang dikirim di query string.
	if !auth.IsAdmin(r.Context()) {
		filter.UserID = userID
	} else if requested := query.Get("user_id"); requested != "" {
		filter.UserID = requested
	}

	bookings, err := h.svc.ListBookings(r.Context(), filter)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":   bookings,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *Handler) CancelBooking(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, errors.New("token tidak valid"))
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("id booking wajib diisi"))
		return
	}

	// Admin boleh membatalkan booking siapa pun; user hanya miliknya.
	var err error
	if auth.IsAdmin(r.Context()) {
		err = h.svc.CancelBooking(r.Context(), id)
	} else {
		err = h.svc.CancelOwnBooking(r.Context(), userID, id)
	}
	if err != nil {
		writeServiceError(w, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func canAccess(r *http.Request, ownerID string) bool {
	if auth.IsAdmin(r.Context()) {
		return true
	}
	userID, ok := auth.UserID(r.Context())
	return ok && userID == ownerID
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		utils.WriteError(w, http.StatusBadRequest, err)
	case errors.Is(err, service.ErrTripNotFound):
		utils.WriteError(w, http.StatusNotFound, err)
	case errors.Is(err, service.ErrTripUnavailable):
		utils.WriteError(w, http.StatusConflict, err)
	case errors.Is(err, service.ErrNotCancellable):
		utils.WriteError(w, http.StatusConflict, err)
	case errors.Is(err, service.ErrForbidden):
		// Sama seperti GetBooking: jangan konfirmasi keberadaan id.
		utils.WriteError(w, http.StatusNotFound, errors.New("booking tidak ditemukan"))
	case errors.Is(err, catalog.ErrUnavailable):
		utils.WriteError(w, http.StatusServiceUnavailable, errors.New("layanan katalog sedang tidak tersedia"))
	default:
		utils.WriteError(w, http.StatusInternalServerError, err)
	}
}

func atoiDefault(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
