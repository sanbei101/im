package service

import (
	"context"
	"uuid"

	"github.com/sanbei101/im/internal/store"
)

type MessageService struct{ store *store.Store }

type HistoryReq struct {
	RoomID string `query:"room_id"    validate:"required"`
	Before uint64 `query:"before_seq"`
	Limit  int    `query:"page_size"`
}

type HistoryResp struct {
	Messages []store.Message `json:"messages"`
	HasMore  bool            `json:"hasMore"`
}

func NewMessageService(s *store.Store) *MessageService { return &MessageService{store: s} }

func (s *MessageService) GetHistory(ctx context.Context, req HistoryReq) (*HistoryResp, error) {
	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		return nil, err
	}
	page, err := s.store.Messages(ctx, roomID, req.Before, req.Limit)
	if err != nil {
		return nil, err
	}
	return &HistoryResp{Messages: page.Messages, HasMore: page.HasMore}, nil
}
