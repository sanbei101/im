package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/api/service"
	"github.com/sanbei101/im/pkg/jwt"
	"github.com/sanbei101/im/pkg/render"
)

type MessageHandler struct {
	svc *service.MessageService
}

func NewMessageHandler(svc *service.MessageService) *MessageHandler {
	return &MessageHandler{svc: svc}
}

func (h *MessageHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	before, err := parseCursor(q.Get("before_server_time"), time.Now().UnixMicro())
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid before_server_time")
		return
	}
	resp, err := h.svc.GetHistory(r.Context(), jwt.GetUserIDFromContext(r), service.HistoryReq{
		RoomID: q.Get("room_id"), BeforeServerTime: before, PageSize: parsePageSize(q.Get("page_size")),
	})
	if err != nil {
		handleMessageError(w, err)
		return
	}
	render.Success(w, "获取历史消息成功", resp)
}

func (h *MessageHandler) Sync(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	after, err := parseCursor(q.Get("after_server_time"), 0)
	if err != nil {
		render.Error(w, http.StatusBadRequest, "invalid after_server_time")
		return
	}
	resp, err := h.svc.Sync(r.Context(), jwt.GetUserIDFromContext(r), service.SyncReq{
		AfterServerTime: after, PageSize: parsePageSize(q.Get("page_size")),
	})
	if err != nil {
		handleMessageError(w, err)
		return
	}
	render.Success(w, "同步消息成功", resp)
}

func parseCursor(s string, fallback int64) (int64, error) {
	if s == "" {
		return fallback, nil
	}
	return strconv.ParseInt(s, 10, 64)
}

func parsePageSize(s string) int {
	if s == "" {
		return 20
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return 20
	}
	return v
}

func handleMessageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrRoomAccessDenied):
		render.Error(w, http.StatusForbidden, "room access denied")
	case errors.Is(err, service.ErrInvalidInput):
		render.Error(w, http.StatusBadRequest, "invalid message query")
	case errors.Is(err, service.ErrRecallNotAllowed):
		render.Error(w, http.StatusConflict, "message cannot be recalled")
	default:
		log.Error().Err(err).Msg("message request failed")
		render.Error(w, http.StatusInternalServerError, "failed to load messages")
	}
}

func (h *MessageHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	req, err := render.ReadBody[service.ReadReq](w, r)
	if err != nil {
		return
	}
	if err := h.svc.MarkRead(r.Context(), jwt.GetUserIDFromContext(r), req); err != nil {
		handleMessageError(w, err)
		return
	}
	render.Success(w, "标记已读成功", struct{}{})
}

func (h *MessageHandler) Recall(w http.ResponseWriter, r *http.Request) {
	resp, err := h.svc.Recall(r.Context(), jwt.GetUserIDFromContext(r), chi.URLParam(r, "msgID"))
	if err != nil {
		handleMessageError(w, err)
		return
	}
	render.Success(w, "撤回消息成功", resp)
}
