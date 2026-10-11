package api

import (
	"errors"
	"net/http"

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
	myID, ok := requireUser(w, r)
	if !ok {
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
	myID, ok := requireUser(w, r)
	if !ok {
		return
	}
	roomID, ok := roomIDFromPath(w, r)
	if !ok {
		return
	}
	if !requireRoomMember(a.store, w, r, roomID) {
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
	myID, ok := requireUser(w, r)
	if !ok {
		return
	}
	roomID, ok := roomIDFromPath(w, r)
	if !ok {
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
	if !requireRoomMember(a.store, w, r, roomID) {
		return
	}

	if err := a.store.MarkRoomRead(r.Context(), myID, roomID, room.LastSeq); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, "清空未读成功")
}

// updateMemberSettings applies one conversation toggle for the caller and
// writes the response; Pin and Mute differ only in which switch they flip.
func (a *ConversationAPI) updateMemberSettings(
	w http.ResponseWriter,
	r *http.Request,
	pinned, muted *bool,
	successMsg string,
) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	roomID, ok := roomIDFromPath(w, r)
	if !ok {
		return
	}

	if err := a.store.UpdateMemberSettings(r.Context(), roomID, userID, pinned, muted); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "conversation not found")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.SuccessNoData(w, http.StatusOK, successMsg)
}

func (a *ConversationAPI) PinConversation(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[PinConversationReq](w, r)
	if err != nil {
		return
	}
	a.updateMemberSettings(w, r, &req.IsPinned, nil, "更新置顶设置成功")
}

func (a *ConversationAPI) MuteConversation(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[MuteConversationReq](w, r)
	if err != nil {
		return
	}
	a.updateMemberSettings(w, r, nil, &req.IsMuted, "更新免打扰设置成功")
}
