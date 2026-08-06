package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type FriendHandler struct {
	svc *service.FriendService
}

func NewFriendHandler(svc *service.FriendService) *FriendHandler {
	return &FriendHandler{svc: svc}
}

func (h *FriendHandler) SendRequest(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.UserIDReq](w, r)
	if err != nil {
		return
	}
	resp, err := h.svc.SendRequest(r.Context(), jwt.GetUserIDFromContext(r), req)
	if err != nil {
		handleFriendError(w, err)
		return
	}
	render.Success(w, "好友申请已发送", resp)
}

func (h *FriendHandler) ListReceivedRequests(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListReceivedRequests(r.Context(), jwt.GetUserIDFromContext(r))
	if err != nil {
		handleFriendError(w, err)
		return
	}
	render.Success(w, "获取好友申请成功", resp)
}

func (h *FriendHandler) AcceptRequest(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.AcceptRequest(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "requestID")); err != nil {
		handleFriendError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "已接受好友申请")
}

func (h *FriendHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RejectRequest(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "requestID")); err != nil {
		handleFriendError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "已拒绝好友申请")
}

func (h *FriendHandler) DeleteFriend(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteFriend(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "userID")); err != nil {
		handleFriendError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "已删除好友")
}

func (h *FriendHandler) ListFriends(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListFriends(r.Context(), jwt.GetUserIDFromContext(r))
	if err != nil {
		handleFriendError(w, err)
		return
	}
	render.Success(w, "获取好友列表成功", resp)
}

func (h *FriendHandler) Block(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.UserIDReq](w, r)
	if err != nil {
		return
	}
	if err := h.svc.Block(r.Context(), jwt.GetUserIDFromContext(r), req); err != nil {
		handleFriendError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "已拉黑用户")
}

func (h *FriendHandler) Unblock(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Unblock(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "userID")); err != nil {
		handleFriendError(w, err)
		return
	}
	render.SuccessNoData(w, http.StatusOK, "已取消拉黑")
}

func (h *FriendHandler) ListBlocks(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListBlocks(r.Context(), jwt.GetUserIDFromContext(r))
	if err != nil {
		handleFriendError(w, err)
		return
	}
	render.Success(w, "获取黑名单成功", resp)
}

func handleFriendError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrCannotRelateSelf):
		render.Error(w, http.StatusBadRequest, "invalid user relation")
	case errors.Is(err, service.ErrUserNotFound), errors.Is(err, service.ErrFriendRequestAbsent):
		render.Error(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, service.ErrUserBlocked), errors.Is(err, service.ErrFriendRequestDenied):
		render.Error(w, http.StatusForbidden, "relation access denied")
	case errors.Is(err, service.ErrAlreadyFriends),
		errors.Is(err, service.ErrFriendRequestExists),
		errors.Is(err, service.ErrFriendRequestClosed):
		render.Error(w, http.StatusConflict, "relation state conflict")
	default:
		log.Error().Err(err).Msg("friend request failed")
		render.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
