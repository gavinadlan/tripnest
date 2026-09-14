package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.SearchService
}

func NewHandler(svc service.SearchService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse Query Params
	query := r.URL.Query()
	page, _ := strconv.Atoi(query.Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(query.Get("limit"))
	if limit < 1 || limit > 50 {
		limit = 10
	}
	minPrice, _ := strconv.ParseFloat(query.Get("min_price"), 64)
	maxPrice, _ := strconv.ParseFloat(query.Get("max_price"), 64)

	params := &model.SearchParams{
		Destination: query.Get("destination"),
		Date:        query.Get("date"),
		MinPrice:    minPrice,
		MaxPrice:    maxPrice,
		Page:        page,
		Limit:       limit,
	}

	listings, total, err := h.svc.SearchListings(r.Context(), params)
	if err != nil {
		http.Error(w, `{"error": "search failed"}`, http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"data":  listings,
		"page":  page,
		"limit": limit,
		"total": total,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *Handler) ListTrips(w http.ResponseWriter, r *http.Request) {
	trips, err := h.svc.ListTrips(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list trips"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": trips})
}

// GetTrip melayani halaman detail di frontend. booking-service TIDAK
// memakai endpoint ini lagi: harga resmi diambil dari catalog-service,
// pemilik datanya, bukan dari read model yang bisa tertinggal.
func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("id wajib diisi"))
		return
	}

	trip, err := h.svc.GetTrip(r.Context(), id)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if trip == nil {
		utils.WriteError(w, http.StatusNotFound, errors.New("trip tidak ditemukan"))
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, trip)
}

