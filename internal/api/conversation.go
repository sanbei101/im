package api

import (
	"errors"
	"net/http"
	"uuid"

	"github.com/go-chi/chi/v5"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

type ConversationAPI struct {
	store *store.Store
}

type MarkReadReq struct {
	ReadSeq uint64 `json:"read_seq" validate:"required"`
}

type PinConversationReq struct {
	IsPinned bool `json:"is_pinned"`
}

type MuteConversationReq struct {
	IsMuted bool `json:"is_muted"`
}

func (a *ConversationAPI) ListConversations(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	convs, err := a.store.Conversations(r.Context(), myID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取会话列表成功", map[string]any{"conversations": convs})
}

func (a *ConversationAPI) MarkRead(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	req, err := render.ReadBody[MarkReadReq](w, r)
	if err != nil {
		return
	}

	if err := a.store.MarkRoomRead(r.Context(), myID, roomID, req.ReadSeq); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "已读上报成功")
}

func (a *ConversationAPI) ClearUnread(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	room, err := a.store.Room(r.Context(), roomID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusNotFound, "room not found")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if err := a.store.MarkRoomRead(r.Context(), myID, roomID, room.LastSeq); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "清空未读成功")
}

func (a *ConversationAPI) PinConversation(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	req, err := render.ReadBody[PinConversationReq](w, r)
	if err != nil {
		return
	}

	if err := a.store.UpdateMemberSettings(r.Context(), roomID, myID, &req.IsPinned, nil); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "conversation not found")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "更新置顶设置成功")
}

func (a *ConversationAPI) MuteConversation(w http.ResponseWriter, r *http.Request) {
	myID, err := getContextUserID(r)
	if err != nil {
		render.Error(w, http.StatusUnauthorized, err.Error())
		return
	}
	roomID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room id")
		return
	}

	req, err := render.ReadBody[MuteConversationReq](w, r)
	if err != nil {
		return
	}

	if err := a.store.UpdateMemberSettings(r.Context(), roomID, myID, nil, &req.IsMuted); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "conversation not found")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "更新免打扰设置成功")
}
