package api

import (
	"encoding/json/v2"
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
	if !requireRoomMember(a.store, w, r, roomID) {
		return
	}

	before, ok := parseBeforeSeq(w, q.Get("before_seq"))
	if !ok {
		return
	}
	limit, ok := parseLimit(w, q.Get("page_size"))
	if !ok {
		return
	}

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

	member, err := a.store.Member(r.Context(), roomID, userID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusForbidden, "not a room member")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	isOwnerOrAdmin := member.Role == store.RoleOwner || member.Role == store.RoleAdmin

	recalled, err := a.store.RecallMessage(r.Context(), roomID, msgID, userID, isOwnerOrAdmin)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			render.Error(w, http.StatusNotFound, "message not found")
			return
		}
		if errors.Is(err, store.ErrRecallTimeout) {
			render.Error(w, http.StatusBadRequest, "消息发送已超过2分钟,无法撤回")
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
	if a.streamHandler != nil {
		if reactions, err := a.store.Reactions(r.Context(), roomID, msgID); err == nil {
			if payload, err := json.Marshal(map[string]any{
				"room_id":   roomID.String(),
				"msg_id":    msgID.String(),
				"reactions": reactions,
			}); err == nil {
				if pushErr := a.streamHandler.BroadcastRoomNotification(
					r.Context(),
					roomID,
					uuid.Nil(),
					"reaction",
					payload,
				); pushErr != nil {
					log.Error().Err(pushErr).Msg("broadcast reaction failed")
				}
			}
		}
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
	if a.streamHandler != nil {
		if reactions, err := a.store.Reactions(r.Context(), roomID, msgID); err == nil {
			if payload, err := json.Marshal(map[string]any{
				"room_id":   roomID.String(),
				"msg_id":    msgID.String(),
				"reactions": reactions,
			}); err == nil {
				if pushErr := a.streamHandler.BroadcastRoomNotification(
					r.Context(),
					roomID,
					uuid.Nil(),
					"reaction",
					payload,
				); pushErr != nil {
					log.Error().Err(pushErr).Msg("broadcast reaction failed")
				}
			}
		}
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
	if !requireRoomMember(a.store, w, r, roomID) {
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
	if !requireRoomMember(a.store, w, r, roomID) {
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
	if !requireRoomMember(a.store, w, r, roomID) {
		return
	}

	q := r.URL.Query()
	keyword := q.Get("keyword")
	if keyword == "" {
		render.Success(w, "搜索聊天记录成功", []store.Message{})
		return
	}

	before, ok := parseBeforeSeq(w, q.Get("before_seq"))
	if !ok {
		return
	}
	limit, ok := parseLimit(w, q.Get("page_size"))
	if !ok {
		return
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

// parseBeforeSeq parses the before_seq pagination cursor; an empty value
// means 0 (newest page), a present-but-invalid value is rejected with 400.
func parseBeforeSeq(w http.ResponseWriter, raw string) (uint64, bool) {
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid before_seq")
		return 0, false
	}
	return v, true
}

// parseLimit parses page_size, defaulting to 20; out-of-range values are
// rejected with 400 instead of silently clamped.
func parseLimit(w http.ResponseWriter, raw string) (int, bool) {
	if raw == "" {
		return 20, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 || v > 100 {
		render.Error(w, http.StatusBadRequest, "invalid page_size")
		return 0, false
	}
	return v, true
}
