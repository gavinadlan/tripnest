package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/service"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/mongo"
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

func (h *Handler) Seed(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.SeedListings(r.Context()); err != nil {
		http.Error(w, `{"error": "seeding failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "seeded successfully"}`))
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

// GetTrip mengembalikan satu trip. booking-service memakai endpoint ini
// untuk mengambil harga resmi, jadi bentuk responsnya harus stabil.
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

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	var req model.Listing
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}
	if err := validateListing(&req); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.CreateTrip(r.Context(), &req); err != nil {
		http.Error(w, `{"error":"failed to create trip"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(req)
}

func (h *Handler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
		return
	}
	var req model.Listing
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid payload"}`, http.StatusBadRequest)
		return
	}
	if err := validateListing(&req); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.UpdateTrip(r.Context(), id, &req); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.WriteError(w, http.StatusNotFound, errors.New("trip tidak ditemukan"))
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) DeleteTrip(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, `{"error":"missing id"}`, http.StatusBadRequest)
		return
	}
	if err := h.svc.DeleteTrip(r.Context(), id); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			utils.WriteError(w, http.StatusNotFound, errors.New("trip tidak ditemukan"))
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateListing(listing *model.Listing) error {
	if listing.Title == "" || listing.Destination == "" || listing.Date == "" {
		return errors.New("title, destination, dan date wajib diisi")
	}
	if listing.Price <= 0 {
		return errors.New("price harus lebih dari 0")
	}
	if listing.AvailableSlots < 0 {
		return errors.New("available_slots tidak boleh negatif")
	}

	// Tanggal disimpan sebagai string YYYY-MM-DD, jadi formatnya harus
	// divalidasi di sini -- kalau tidak, filter pencarian per tanggal
	// diam-diam tidak pernah cocok.
	parsed, err := time.Parse("2006-01-02", listing.Date)
	if err != nil {
		return errors.New("date harus berformat YYYY-MM-DD")
	}
	if parsed.Before(time.Now().UTC().Truncate(24 * time.Hour)) {
		return errors.New("date tidak boleh di masa lalu")
	}
	return nil
}
