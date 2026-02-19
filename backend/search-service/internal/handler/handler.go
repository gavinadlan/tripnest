package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gavinadlan/tripnest/backend/search-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/search-service/internal/service"
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
