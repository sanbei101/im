package api

import (
	"errors"
	"net/http"
	"strconv"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type MessageAPI struct {
	store         *store.Store
	streamHandler *StreamHandler
}

type RecallReq struct {
	RoomID string `json:"room_id" validate:"required,uuid"`
	MsgID  string `json:"msg_id"  validate:"required,uuid"`
}

type ReactionReq struct {
	RoomID string `json:"room_id" validate:"required,uuid"`
	Emoji  string `json:"emoji"   validate:"required"`
}

func (a *MessageAPI) GetHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	roomID, err := uuid.Parse(q.Get("room_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}

	before := parseBeforeSeq(q.Get("before_seq"))
	limit := parseLimit(q.Get("page_size"))

	page, err := a.store.Messages(r.Context(), roomID, before, limit)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取历史消息成功", page)
}

func (a *MessageAPI) Recall(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[RecallReq](w, r)
	if err != nil {
		return
	}
	userIDStr := jwt.GetUserIDFromContext(r)
	if userIDStr == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}
	msgID, err := uuid.Parse(req.MsgID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid msg_id")
		return
	}

	members, err := a.store.Members(r.Context(), roomID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	var isOwnerOrAdmin bool
	var isMember bool
	for _, m := range members {
		if m.UserID == userID {
			isMember = true
			if m.Role == "owner" || m.Role == "admin" {
				isOwnerOrAdmin = true
			}
			break
		}
	}
	if !isMember {
		render.Error(w, http.StatusForbidden, "not a room member")
		return
	}

	recalled, err := a.store.RecallMessage(r.Context(), roomID, msgID, userID, isOwnerOrAdmin)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "message not found")
			return
		}
		if errors.Is(err, store.ErrRecallTimeout) {
			render.Error(w, http.StatusBadRequest, "消息发送已超过2分钟，无法撤回")
			return
		}
		if errors.Is(err, store.ErrForbidden) {
			render.Error(w, http.StatusForbidden, "无权撤回该消息")
			return
		}
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if a.streamHandler != nil {
		if pushErr := a.streamHandler.Push(r.Context(), &recalled); pushErr != nil {
			log.Error().Err(pushErr).Msg("push recalled message failed")
		}
	}

	render.Success(w, "撤回消息成功", recalled)
}

func (a *MessageAPI) AddReaction(w http.ResponseWriter, r *http.Request) {
	msgIDStr := chi.URLParam(r, "id")
	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	req, err := render.ReadBody[ReactionReq](w, r)
	if err != nil {
		return
	}
	userIDStr := jwt.GetUserIDFromContext(r)
	if userIDStr == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}

	if err := a.store.AddReaction(r.Context(), roomID, msgID, userID, req.Emoji); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.SuccessNoData(w, http.StatusOK, "添加表情表态成功")
}

func (a *MessageAPI) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	msgIDStr := chi.URLParam(r, "id")
	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	userIDStr := jwt.GetUserIDFromContext(r)
	if userIDStr == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}
	roomIDStr := r.URL.Query().Get("room_id")
	emoji := r.URL.Query().Get("emoji")
	if roomIDStr == "" || emoji == "" {
		req, err := render.ReadBody[ReactionReq](w, r)
		if err == nil {
			roomIDStr = req.RoomID
			emoji = req.Emoji
		}
	}
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}

	if err := a.store.RemoveReaction(r.Context(), roomID, msgID, userID, emoji); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.SuccessNoData(w, http.StatusOK, "取消表情表态成功")
}

func (a *MessageAPI) GetReactions(w http.ResponseWriter, r *http.Request) {
	msgIDStr := chi.URLParam(r, "id")
	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	roomID, err := uuid.Parse(r.URL.Query().Get("room_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}

	reactions, err := a.store.Reactions(r.Context(), roomID, msgID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(w, "获取表情表态成功", reactions)
}

func (a *MessageAPI) GetReadUsers(w http.ResponseWriter, r *http.Request) {
	msgIDStr := chi.URLParam(r, "id")
	msgID, err := uuid.Parse(msgIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	roomID, err := uuid.Parse(r.URL.Query().Get("room_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}

	users, err := a.store.ReadUsers(r.Context(), roomID, msgID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(w, "获取已读成员成功", map[string]any{
		"read_user_ids": users,
		"count":         len(users),
	})
}

func (a *MessageAPI) Search(w http.ResponseWriter, r *http.Request) {
	roomIDStr := chi.URLParam(r, "id")
	if roomIDStr == "" {
		roomIDStr = r.URL.Query().Get("room_id")
	}
	roomID, err := uuid.Parse(roomIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return
	}

	userIDStr := jwt.GetUserIDFromContext(r)
	if userIDStr == "" {
		render.Error(w, http.StatusUnauthorized, "user not authenticated")
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	if _, err := a.store.Member(r.Context(), roomID, userID); err != nil {
		render.Error(w, http.StatusForbidden, "not a member of this room")
		return
	}

	q := r.URL.Query()
	keyword := q.Get("keyword")
	if keyword == "" {
		render.Success(w, "搜索聊天记录成功", []store.Message{})
		return
	}

	before := parseBeforeSeq(q.Get("before"))
	if before == 0 {
		before = parseBeforeSeq(q.Get("before_seq"))
	}
	limit := parseLimit(q.Get("limit"))
	if q.Get("limit") == "" && q.Get("page_size") != "" {
		limit = parseLimit(q.Get("page_size"))
	}

	messages, err := a.store.SearchRoomMessages(r.Context(), roomID, keyword, before, limit)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if messages == nil {
		messages = []store.Message{}
	}

	render.Success(w, "搜索聊天记录成功", messages)
}

func parseBeforeSeq(s string) uint64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseLimit(s string) int {
	if s == "" {
		return 20
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 || v > 100 {
		return 20
	}
	return v
}
