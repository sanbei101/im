package api

import (
	"net/http"
	"strconv"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
)

type MessageAPI struct {
	store *store.Store
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
