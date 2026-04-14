package handler

import (
	"errors"
	"math"
	"net/http"

	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/inventory-service/internal/service"
)

type Handler struct {
	svc service.InventoryService
}

func NewHandler(svc service.InventoryService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /inventory", h.UpsertInventory)
	mux.HandleFunc("GET /inventory", h.GetInventory)
	mux.HandleFunc("PATCH /inventory/{resource_id}", h.UpdateInventory)
	mux.HandleFunc("GET /health", h.Health)
}

func (h *Handler) UpsertInventory(w http.ResponseWriter, r *http.Request) {
	var req model.UpsertInventoryRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}
	if req.ResourceID == "" || req.TotalSlots < 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("resource_id and non-negative total_slots are required"))
		return
	}
	if err := h.svc.UpsertInventory(r.Context(), req); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) GetInventory(w http.ResponseWriter, r *http.Request) {
	resourceID := r.URL.Query().Get("resource_id")
	if resourceID == "" {
		items, err := h.svc.ListInventory(r.Context())
		if err != nil {
			utils.WriteError(w, http.StatusInternalServerError, err)
			return
		}
		utils.WriteJSON(w, http.StatusOK, map[string]interface{}{"data": items})
		return
	}
	inventory, err := h.svc.GetInventoryByResourceID(r.Context(), resourceID)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	if inventory == nil {
		utils.WriteError(w, http.StatusNotFound, errors.New("inventory not found"))
		return
	}
	utils.WriteJSON(w, http.StatusOK, inventory)
}

func (h *Handler) UpdateInventory(w http.ResponseWriter, r *http.Request) {
	resourceID := r.PathValue("resource_id")
	if resourceID == "" {
		utils.WriteError(w, http.StatusBadRequest, errors.New("resource_id path param is required"))
		return
	}

	var req map[string]interface{}
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	totalRaw, ok := req["total_slots"]
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots is required"))
		return
	}
	totalFloat, ok := totalRaw.(float64)
	if !ok {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots must be a number"))
		return
	}
	if totalFloat != math.Trunc(totalFloat) {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots must be an integer"))
		return
	}
	totalSlots := int(totalFloat)
	if totalSlots < 0 {
		utils.WriteError(w, http.StatusBadRequest, errors.New("total_slots must be non-negative"))
		return
	}

	if err := h.svc.UpdateInventory(r.Context(), resourceID, totalSlots); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
