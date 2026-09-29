package handler

import (
	"net/http"
	"strconv"

	"github.com/sanbei101/im/internal/api/service"
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
	req := service.HistoryReq{
		RoomID: q.Get("room_id"),
		Before: parseBeforeSequence(q.Get("before_seq")),
		Limit:  parsePageSize(q.Get("page_size")),
	}

	resp, err := h.svc.GetHistory(r.Context(), req)
	if err != nil {
		render.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	render.Success(w, "获取历史消息成功", resp)
}

func parseBeforeSequence(s string) uint64 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// parsePageSize returns 20 when the value is missing or malformed.
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
