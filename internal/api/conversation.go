package api

import (
	"errors"
	"net/http"
	"slices"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

type ConversationAPI struct {
	db *store.DB
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

type ConversationListResp struct {
	Conversations []store.ConversationInfo `json:"conversations"`
}

func (a *ConversationAPI) ListConversations(w http.ResponseWriter, r *http.Request) {
	myID, ok := requireUser(w, r)
	if !ok {
		return
	}
	rooms, err := a.db.RoomsByUser(r.Context(), myID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	conversations := make([]store.ConversationInfo, 0, len(rooms))
	for i := range rooms {
		info := &rooms[i]
		head, err := a.db.RoomHead(r.Context(), info.Room.RoomID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		var unread uint64
		readSeq, err := a.db.ReadSeq(r.Context(), myID, info.Room.RoomID)
		if err != nil && !errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		if head.LastSeq > readSeq {
			unread = head.LastSeq - readSeq
		}
		info.Room.LastSeq = head.LastSeq
		if !head.UpdatedAt.IsZero() {
			info.Room.UpdatedAt = head.UpdatedAt
		}
		conversations = append(conversations, store.ConversationInfo{
			Room: info.Room, Member: info.Member,
			UnreadCount: unread, LastMessage: head.LastMsg,
		})
	}
	slices.SortFunc(conversations, func(a, b store.ConversationInfo) int {
		at, bt := conversationTime(a), conversationTime(b)
		switch {
		case at > bt:
			return -1
		case at < bt:
			return 1
		default:
			return 0
		}
	})
	render.Success(w, "获取会话列表成功", ConversationListResp{Conversations: conversations})
}

func conversationTime(info store.ConversationInfo) int64 {
	if info.LastMessage != nil {
		return info.LastMessage.ServerTime
	}
	return info.Room.UpdatedAt.UnixMicro()
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
	if !requireRoomMember(a.db, w, r, roomID) {
		return
	}
	req, err := render.ReadBody[MarkReadReq](w, r)
	if err != nil {
		return
	}
	if err := a.db.MarkRead(r.Context(), myID, roomID, req.ReadSeq); err != nil {
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
	if !requireRoomMember(a.db, w, r, roomID) {
		return
	}
	head, err := a.db.RoomHead(r.Context(), roomID)
	if errors.Is(err, store.ErrNotFound) {
		render.SuccessNoData(w, http.StatusOK, "清空未读成功")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.db.MarkRead(r.Context(), myID, roomID, head.LastSeq); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.SuccessNoData(w, http.StatusOK, "清空未读成功")
}

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
	if err := a.db.UpdateMemberSettings(r.Context(), roomID, userID, pinned, muted); err != nil {
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
