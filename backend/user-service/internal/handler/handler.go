package handler

import (
	"errors"
	"net/http"

	"github.com/gavinadlan/tripnest/backend/common/auth"
	"github.com/gavinadlan/tripnest/backend/common/utils"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/model"
	"github.com/gavinadlan/tripnest/backend/user-service/internal/service"
)

type Handler struct {
	svc service.UserService
}

func NewHandler(svc service.UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req model.RegisterUserRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, errors.New("payload tidak valid"))
		return
	}

	user, err := h.svc.Register(r.Context(), &req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEmailTaken):
			utils.WriteError(w, http.StatusConflict, err)
		case errors.Is(err, service.ErrValidation):
			utils.WriteError(w, http.StatusBadRequest, err)
		default:
			utils.WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	_ = utils.WriteJSON(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req model.LoginUserRequest
	if err := utils.ReadJSON(r, &req); err != nil {
		utils.WriteError(w, http.StatusBadRequest, errors.New("payload tidak valid"))
		return
	}

	resp, err := h.svc.Login(r.Context(), &req)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			utils.WriteError(w, http.StatusUnauthorized, err)
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, resp)
}

// Me mengembalikan profil pemanggil berdasarkan token, bukan berdasarkan
// id yang dikirim client.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, errors.New("token tidak valid"))
		return
	}

	user, err := h.svc.Profile(r.Context(), userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			utils.WriteError(w, http.StatusNotFound, err)
			return
		}
		utils.WriteError(w, http.StatusInternalServerError, err)
		return
	}

	_ = utils.WriteJSON(w, http.StatusOK, user)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
