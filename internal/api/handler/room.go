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

type RoomHandler struct {
	svc *service.RoomService
}

func NewRoomHandler(svc *service.RoomService) *RoomHandler {
	return &RoomHandler{svc: svc}
}

func (h *RoomHandler) CreateOrGetSingleChatRoom(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.CreateRoomReq](w, r)
	if err != nil {
		return
	}
	userID := jwt.GetUserIDFromContext(r)
	if userID == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := h.svc.CreateOrGetSingleChatRoom(r.Context(), userID, req)
	if err != nil {
		handleRoomError(w, err)
		return
	}

	render.Success(w, "获取或创建单聊房间成功", resp)
}

func (h *RoomHandler) CreateGroupRoom(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.CreateGroupRoomReq](w, r)
	if err != nil {
		return
	}

	userID := jwt.GetUserIDFromContext(r)
	if userID == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}

	resp, err := h.svc.CreateGroupRoom(r.Context(), userID, req)
	if err != nil {
		handleRoomError(w, err)
		return
	}

	render.Success(w, "创建群聊房间成功", resp)
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	userID := jwt.GetUserIDFromContext(r)
	resp, err := h.svc.ListRooms(r.Context(), userID)
	if err != nil {
		handleRoomError(w, err)
		return
	}

	render.Success(w, "获取房间列表成功", resp)
}

func handleRoomError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput), errors.Is(err, service.ErrCannotRelateSelf):
		render.Error(w, http.StatusBadRequest, "invalid room request")
	case errors.Is(err, service.ErrUserNotFound):
		render.Error(w, http.StatusNotFound, "user not found")
	case errors.Is(err, service.ErrUsersNotFriends),
		errors.Is(err, service.ErrUserBlocked),
		errors.Is(err, service.ErrRoomAccessDenied),
		errors.Is(err, service.ErrRoomDenied),
		errors.Is(err, service.ErrRoomOwnerRequired),
		errors.Is(err, service.ErrRoomOwnerCannotLeave):
		render.Error(w, http.StatusForbidden, "room access denied")
	case errors.Is(err, service.ErrRoomNotFound), errors.Is(err, service.ErrRoomMemberNotFound):
		render.Error(w, http.StatusNotFound, "room or member not found")
	case errors.Is(err, service.ErrRoomNotGroup):
		render.Error(w, http.StatusBadRequest, "operation requires group room")
	default:
		log.Error().Err(err).Msg("room request failed")
		render.Error(w, http.StatusInternalServerError, "internal server error")
	}
}

func (h *RoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.GetRoom(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"))
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "获取房间详情成功", resp)
}

func (h *RoomHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.ListMembers(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"))
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "获取房间成员成功", resp)
}

func (h *RoomHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RoomMemberReq](w, r)
	if err != nil {
		return
	}
	err = h.svc.AddMember(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"), req)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "添加房间成员成功", struct{}{})
}

func (h *RoomHandler) KickMember(w http.ResponseWriter, r *http.Request) {
	err := h.svc.KickMember(
		r.Context(),
		jwt.GetUserIDFromContext(r),
		chi.URLParam(r, "roomID"),
		chi.URLParam(r, "userID"),
	)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "移除房间成员成功", struct{}{})
}

func (h *RoomHandler) Leave(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Leave(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"))
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "退出房间成功", struct{}{})
}

func (h *RoomHandler) Dissolve(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Dissolve(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"))
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "解散房间成功", struct{}{})
}

func (h *RoomHandler) TransferOwnership(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RoomTransferReq](w, r)
	if err != nil {
		return
	}
	err = h.svc.TransferOwnership(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"), req)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "转让群主成功", struct{}{})
}

func (h *RoomHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RoomRoleReq](w, r)
	if err != nil {
		return
	}
	err = h.svc.SetRole(
		r.Context(),
		jwt.GetUserIDFromContext(r),
		chi.URLParam(r, "roomID"),
		chi.URLParam(r, "userID"),
		req,
	)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "调整成员角色成功", struct{}{})
}

func (h *RoomHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.RoomSettingsReq](w, r)
	if err != nil {
		return
	}
	err = h.svc.UpdateSettings(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "roomID"), req)
	if err != nil {
		handleRoomError(w, err)
		return
	}
	render.Success(w, "更新房间设置成功", struct{}{})
}
