package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

// MessageAPI serves the message plane. Room membership is resolved through
// the shared roster cache, so no message-plane endpoint waits on PostgreSQL.
type MessageAPI struct {
	store         *store.Store
	members       *Membership
	streamHandler *StreamHandler
}

// Every room-scoped route carries its room id in the path or as the room_id
// query parameter; request bodies never carry routing fields.
type RecallReq struct {
	MsgID string `json:"msg_id" validate:"required,uuid"`
}

type ReactionReq struct {
	Emoji string `json:"emoji" validate:"required"`
}

// maxReactionEmojiLen bounds the emoji carried in a reaction key so a client
// cannot stuff arbitrary bytes into the LSM.
const maxReactionEmojiLen = 32

// ReadUsersResp lists the members that have read up to a message.
type ReadUsersResp struct {
	ReadUserIDs []uuid.UUID `json:"read_user_ids"`
	Count       int         `json:"count"`
}

func (a *MessageAPI) GetHistory(w http.ResponseWriter, r *http.Request) {
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	if !requireRoomMember(a.members, w, r, roomID) {
		return
	}
	q := r.URL.Query()
	after, ok := parseSeq(w, "after_seq", q.Get("after_seq"))
	if !ok {
		return
	}
	before, ok := parseSeq(w, "before_seq", q.Get("before_seq"))
	if !ok {
		return
	}
	limit, ok := parseLimit(w, q.Get("page_size"))
	if !ok {
		return
	}
	if after > 0 && before > 0 {
		render.Error(w, http.StatusBadRequest, "after_seq and before_seq cannot be combined")
		return
	}

	if after > 0 {
		page, err := a.store.MessagesAfter(r.Context(), roomID, after, limit)
		if err != nil {
			render.Error(w, http.StatusInternalServerError, err.Error())
			return
		}
		render.Success(w, "获取历史消息成功", page)
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
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	req, err := render.ReadBody[RecallReq](w, r)
	if err != nil {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	msgID, err := uuid.Parse(req.MsgID)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid msg_id")
		return
	}

	member, err := a.members.Member(r.Context(), roomID, userID)
	if errors.Is(err, store.ErrNotFound) {
		render.Error(w, http.StatusForbidden, "not a room member")
		return
	}
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	recalled, err := a.store.RecallMessage(r.Context(), roomID, msgID, userID, isManager(member.Role))
	if err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			render.Error(w, http.StatusNotFound, "message not found")
		case errors.Is(err, store.ErrRecallTimeout):
			render.Error(w, http.StatusBadRequest, "消息发送已超过2分钟,无法撤回")
		case errors.Is(err, store.ErrForbidden):
			render.Error(w, http.StatusForbidden, "无权撤回该消息")
		default:
			render.Error(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	if a.streamHandler != nil {
		if pushErr := a.streamHandler.PushExcluding(r.Context(), &recalled, userID); pushErr != nil {
			log.Error().Err(pushErr).Msg("push recalled message failed")
		}
	}

	render.Success(w, "撤回消息成功", recalled)
}

func (a *MessageAPI) AddReaction(w http.ResponseWriter, r *http.Request) {
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	req, err := render.ReadBody[ReactionReq](w, r)
	if err != nil {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	if len(req.Emoji) > maxReactionEmojiLen {
		render.Error(w, http.StatusBadRequest, "emoji too long")
		return
	}
	if !requireRoomMember(a.members, w, r, roomID) {
		return
	}

	if err := a.store.AddReaction(r.Context(), roomID, msgID, userID, req.Emoji); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.notifyReaction(r.Context(), roomID, msgID)
	render.SuccessNoData(w, http.StatusOK, "添加表情表态成功")
}

func (a *MessageAPI) RemoveReaction(w http.ResponseWriter, r *http.Request) {
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	emoji := r.URL.Query().Get("emoji")
	if emoji == "" || len(emoji) > maxReactionEmojiLen {
		render.Error(w, http.StatusBadRequest, "invalid emoji")
		return
	}
	if !requireRoomMember(a.members, w, r, roomID) {
		return
	}

	if err := a.store.RemoveReaction(r.Context(), roomID, msgID, userID, emoji); err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	a.notifyReaction(r.Context(), roomID, msgID)
	render.SuccessNoData(w, http.StatusOK, "取消表情表态成功")
}

// notifyReaction broadcasts the message's current reaction groups to the room.
func (a *MessageAPI) notifyReaction(ctx context.Context, roomID, msgID uuid.UUID) {
	if a.streamHandler == nil {
		return
	}
	reactions, err := a.store.Reactions(ctx, roomID, msgID)
	if err != nil {
		log.Error().Err(err).Str("room_id", roomID.String()).Msg("load reactions for notification failed")
		return
	}
	a.streamHandler.NotifyRoom(ctx, roomID, uuid.Nil(), ReactionUpdated{
		RoomID: roomID, MsgID: msgID, Reactions: reactions,
	})
}

func (a *MessageAPI) GetReactions(w http.ResponseWriter, r *http.Request) {
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	if !requireRoomMember(a.members, w, r, roomID) {
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
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid message id")
		return
	}
	if !requireRoomMember(a.members, w, r, roomID) {
		return
	}

	members, err := a.members.Members(r.Context(), roomID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	message, err := a.store.MessageByID(r.Context(), roomID, msgID)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	users, err := a.store.ReadUsers(r.Context(), roomID, message.RoomSeq, members)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	render.Success(w, "获取已读成员成功", ReadUsersResp{
		ReadUserIDs: users,
		Count:       len(users),
	})
}

func (a *MessageAPI) Search(w http.ResponseWriter, r *http.Request) {
	roomID, ok := roomIDFromRequest(w, r)
	if !ok {
		return
	}
	if !requireRoomMember(a.members, w, r, roomID) {
		return
	}

	keyword := r.URL.Query().Get("keyword")
	if keyword == "" {
		render.Success(w, "搜索聊天记录成功", []store.Message{})
		return
	}
	before, ok := parseSeq(w, "before_seq", r.URL.Query().Get("before_seq"))
	if !ok {
		return
	}
	limit, ok := parseLimit(w, r.URL.Query().Get("page_size"))
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

// roomIDFromRequest reads the room id shared by every message-plane endpoint:
// the room_id query parameter. Invalid values are rejected with 400 so a typo
// cannot silently fall back to another lookup path.
func roomIDFromRequest(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	roomID, err := uuid.Parse(r.URL.Query().Get("room_id"))
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid room_id")
		return uuid.Nil(), false
	}
	return roomID, true
}

// parseSeq parses a pagination cursor (after_seq or before_seq); an empty
// value means 0, a present-but-invalid value is rejected with 400.
func parseSeq(w http.ResponseWriter, name, raw string) (uint64, bool) {
	if raw == "" {
		return 0, true
	}
	v, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid "+name)
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
