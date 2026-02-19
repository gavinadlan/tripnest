package handler

import (
	"errors"
	"log"
	"net/http"

	"github.com/gavinadlan/tripnest/backend/booking-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/booking-service/internal/service"
	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.BookingService
}

func NewHandler(svc service.BookingService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/bookings", h.CreateBooking)
	r.Get("/bookings/{id}", h.GetBooking)
	r.Get("/health", h.Health)
}

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req model.CreateBookingRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	// Validate (simple)
	if req.UserID == "" || req.ResourceID == "" || req.TotalAmount <= 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("invalid booking request: missing required fields"))
		return
	}

	booking, err := h.svc.CreateBooking(r.Context(), &req)
	if err != nil {
		log.Printf("CreateBooking failed: %v", err)
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, booking)
}

func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("missing booking id"))
		return
	}

	booking, err := h.svc.GetBooking(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err) // Or NotFound
		return
	}
	if booking == nil {
		utils.WriteError(w, http.StatusNotFound, errors.New("booking not found"))
		return
	}

	utils.WriteJSON(w, http.StatusOK, booking)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
