package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sanbei101/im/internal/db"
)

type MessageService struct {
	query *db.Queries
}

func NewMessageService(query *db.Queries) *MessageService {
	return &MessageService{query: query}
}

type HistoryReq struct {
	RoomID           string `query:"room_id" validate:"required"`
	BeforeServerTime int64  `query:"before_server_time" validate:"required"`
	PageSize         int    `query:"page_size" default:"20" validate:"min=1,max=100"`
}

type HistoryResp struct {
	Messages []*db.ListMessagesByRoomRow `json:"messages"`
	HasMore  bool                        `json:"hasMore"`
}

func (s *MessageService) GetHistory(ctx context.Context, req HistoryReq) (*HistoryResp, error) {
	roomID, err := uuid.Parse(req.RoomID)
	if err != nil {
		return nil, err
	}

	beforeTime := req.BeforeServerTime
	if beforeTime == 0 {
		beforeTime = time.Now().UnixMicro()
	}

	pageSize := int32(req.PageSize)
	if pageSize == 0 {
		pageSize = 20
	}

	messages, err := s.query.ListMessagesByRoom(ctx, db.ListMessagesByRoomParams{
		RoomID:           roomID,
		BeforeServerTime: beforeTime,
		PageSize:         pageSize + 1,
	})
	if err != nil {
		return nil, err
	}

	hasMore := len(messages) > int(pageSize)
	if hasMore {
		messages = messages[:pageSize]
	}

	return &HistoryResp{
		Messages: messages,
		HasMore:  hasMore,
	}, nil
}
