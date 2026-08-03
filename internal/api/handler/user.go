package handler

import (
	"errors"
	"net/http"

	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/pkg/render"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RegisterReq](w, r)
	if err != nil {
		return
	}

	resp, err := h.svc.Register(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			render.Error(w, http.StatusBadRequest, "username already exists")
		case errors.Is(err, service.ErrInvalidInput):
			render.Error(w, http.StatusBadRequest, "invalid username or password")
		default:
			render.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	render.Success(w, "注册成功", resp)
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RegisterReq](w, r)
	if err != nil {
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			render.Error(w, http.StatusUnauthorized, "user not found")
		case errors.Is(err, service.ErrInvalidPassword):
			render.Error(w, http.StatusUnauthorized, "invalid password")
		case errors.Is(err, service.ErrInvalidInput):
			render.Error(w, http.StatusBadRequest, "invalid username or password")
		default:
			render.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	render.Success(w, "登录成功", resp)
}
