package handler

import (
	"errors"
	"net/http"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/pkg/jwt"
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
		handleUserError(w, err)
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
		handleUserError(w, err)
		return
	}
	render.Success(w, "登录成功", resp)
}

func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RefreshReq](w, r)
	if err != nil {
		return
	}
	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		handleUserError(w, err)
		return
	}
	render.Success(w, "刷新成功", resp)
}

func (h *UserHandler) Logout(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.LogoutReq](w, r)
	if err != nil {
		return
	}
	if err := h.svc.Logout(r.Context(), jwt.GetUserIDFromContext(r), req.RefreshToken); err != nil {
		handleUserError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "登出成功")
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetProfile(r.Context(), jwt.GetUserIDFromContext(r))
	if err != nil {
		handleUserError(w, err)
		return
	}
	render.Success(w, "获取资料成功", resp)
}

func (h *UserHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.SearchUsers(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		handleUserError(w, err)
		return
	}
	render.Success(w, "搜索用户成功", resp)
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.UpdateProfileReq](w, r)
	if err != nil {
		return
	}
	resp, err := h.svc.UpdateProfile(r.Context(), jwt.GetUserIDFromContext(r), req)
	if err != nil {
		handleUserError(w, err)
		return
	}
	render.Success(w, "更新资料成功", resp)
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.ChangePasswordReq](w, r)
	if err != nil {
		return
	}
	if err := h.svc.ChangePassword(r.Context(), jwt.GetUserIDFromContext(r), req); err != nil {
		handleUserError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "修改密码成功，请重新登录")
}

func handleUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUserExists):
		render.Error(w, http.StatusConflict, "username already exists")
	case errors.Is(err, service.ErrUserNotFound),
		errors.Is(err, service.ErrInvalidPassword),
		errors.Is(err, service.ErrInvalidSession):
		render.Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrSessionForbidden):
		render.Error(w, http.StatusForbidden, "session access denied")
	case errors.Is(err, service.ErrInvalidInput):
		render.Error(w, http.StatusBadRequest, "invalid request")
	default:
		log.Error().Err(err).Msg("user request failed")
		render.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
