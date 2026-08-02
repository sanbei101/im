package handler

import (
	"net/http"

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
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取或创建单聊房间成功", resp)
}

func (h *RoomHandler) CreateGroupRoom(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.CreateGroupRoomReq](w, r)
	if err != nil {
		return
	}

	resp, err := h.svc.CreateGroupRoom(r.Context(), req)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "创建群聊房间成功", resp)
}

func (h *RoomHandler) ListRooms(w http.ResponseWriter, r *http.Request) {
	userID := jwt.GetUserIDFromContext(r)
	resp, err := h.svc.ListRooms(r.Context(), userID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取房间列表成功", resp)
}