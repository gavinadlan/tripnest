package handler

import (
	"net/http"

	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc service.UserService
}

func NewHandler(svc service.UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/health", h.Health)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterUserRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.svc.Register(r.Context(), &req)
	if err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginUserRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, err)
		return
	}

	resp, err := h.svc.Login(r.Context(), &req)
	if err != nil {
		utils.WriteError(w, http.StatusUnauthorized, err)
		return
	}

	utils.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
