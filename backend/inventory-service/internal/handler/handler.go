package handler

import (
	"errors"
	"math"
	"net/http"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/repository"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/service"
)

type Handler struct {
	svc      service.CatalogService
	verifier *auth.Verifier
}

func NewHandler(svc service.CatalogService, verifier *auth.Verifier) *Handler {
	return &Handler{svc: svc, verifier: verifier}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.Health)

	// Katalog dan ketersediaan kursi boleh dibaca siapa saja: itu etalase.
	mux.HandleFunc("GET /trips", h.ListTrips)
	mux.HandleFunc("GET /trips/{id}", h.GetTrip)
	mux.HandleFunc("GET /inventory", h.GetInventory)

	// Endpoint internal untuk booking-service: sumber resmi harga dan
	// sisa kursi saat booking dibuat. Dibatasi token service/admin supaya
	// bentuk responsnya bebas berubah tanpa memutus klien publik.
	mux.Handle("GET /internal/trips/{id}", h.verifier.RequireServiceOrAdmin(http.HandlerFunc(h.GetTrip)))

	// Menulis katalog hanya untuk admin. Trip dan kapasitasnya dibuat
	// dalam satu transaksi, jadi tidak ada lagi trip tanpa kursi.
	mux.Handle("POST /trips", h.verifier.AdminOnly(h.CreateTrip))
	mux.Handle("PUT /trips/{id}", h.verifier.AdminOnly(h.UpdateTrip))
	mux.Handle("DELETE /trips/{id}", h.verifier.AdminOnly(h.DeleteTrip))
	mux.Handle("POST /seed", h.verifier.AdminOnly(h.Seed))
	mux.Handle("PATCH /inventory/{resource_id}", h.verifier.AdminOnly(h.UpdateInventory))
}

func (h *Handler) ListTrips(w http.ResponseWriter, r *http.Request) {
	trips, err := h.svc.ListTrips(r.Context())
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, map[string]any{"data": trips})
}

func (h *Handler) GetTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
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
	req, ok := readTripRequest(w, r)
	if !ok {
		return
	}
	trip, err := h.svc.CreateTrip(r.Context(), req)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	_ = utils.WriteJSON(w, http.StatusCreated, trip)
}

func (h *Handler) UpdateTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("id wajib diisi"))
		return
	}
	req, ok := readTripRequest(w, r)
	if !ok {
		return
	}
	trip, err := h.svc.UpdateTrip(r.Context(), id, req)
	if err != nil {
		writeCatalogError(w, err)
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, trip)
}

func (h *Handler) DeleteTrip(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("id wajib diisi"))
		return
	}
	if err := h.svc.DeleteTrip(r.Context(), id); err != nil {
		writeCatalogError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Seed(w http.ResponseWriter, r *http.Request) {
	created, err := h.svc.SeedTrips(r.Context())
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, map[string]any{"created": created})
}

func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	resourceID := r.URL.Query().Get("resource_id")
	if resourceID == "" {
		items, err := h.svc.ListInventory(r.Context())
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err)
			return
		}
		_ = utils.WriteJSON(w, http.StatusOK, map[string]any{"data": items})
		return
	}
	inventory, err := h.svc.GetInventoryByResourceID(r.Context(), resourceID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if inventory == nil {
		utils.WriteError(w, http.StatusNotFound, errors.New("inventory tidak ditemukan"))
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, inventory)
}

// UpdateInventory hanya menyesuaikan kapasitas trip yang sudah ada.
// Membuat kapasitas lepas dari trip tidak lagi mungkin -- itulah keadaan
// yang dulu membuat inventory dan katalog saling tidak kenal.
func (h *Handler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	resourceID := r.PathValue("resource_id")
	if resourceID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("resource_id wajib diisi"))
		return
	}

	var req map[string]any
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	totalRaw, ok := req["total_slots"]
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots wajib diisi"))
		return
	}
	totalFloat, ok := totalRaw.(float64)
	if !ok || totalFloat != math.Trunc(totalFloat) {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots harus bilangan bulat"))
		return
	}
	if totalFloat < 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots tidak boleh negatif"))
		return
	}

	if err := h.svc.UpdateInventory(r.Context(), resourceID, int(totalFloat)); err != nil {
		writeCatalogError(w, err)
		return
	}
	_ = utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func readTripRequest(w http.ResponseWriter, r *http.Request) (model.TripRequest, bool) {
	var req model.TripRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, errors.New("payload tidak valid: kirim {title, destination, price, date, total_slots}"))
		return req, false
	}
	req.Normalize()
	if err := req.Validate(); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return req, false
	}
	return req, true
}

func writeCatalogError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrTripNotFound):
		utils.WriteError(w, http.StatusNotFound, err)
	case errors.Is(err, repository.ErrTripHasReservations):
		utils.WriteError(w, http.StatusConflict, err)
	default:
		utils.WriteError(w, http.StatusInternalServerError, err)
	}
}
