package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sanbei101/im/internal/db"
)

type MessageService struct {
	query *db.Queries
}

var (
	ErrRoomAccessDenied = errors.New("room access denied")
	ErrRecallNotAllowed = errors.New("message cannot be recalled")
)

type HistoryReq struct {
	RoomID           string
	BeforeServerTime int64
	PageSize         int
}

type SyncReq struct {
	AfterServerTime int64
	PageSize        int
}

type MessageResp struct {
	MsgID        string `json:"msg_id"`
	ClientMsgID  string `json:"client_msg_id"`
	SenderID     string `json:"sender_id"`
	RoomID       string `json:"room_id"`
	ServerTime   int64  `json:"server_time"`
	ReplyToMsgID string `json:"reply_to_msg_id,omitempty"`
	MsgType      string `json:"msg_type"`
	Payload      []byte `json:"payload"`
	Ext          []byte `json:"ext,omitempty"`
	Recalled     bool   `json:"recalled"`
}

type HistoryResp struct {
	Messages []*MessageResp `json:"messages"`
	HasMore  bool           `json:"has_more"`
}

type SyncResp struct {
	Messages []*MessageResp `json:"messages"`
	HasMore  bool           `json:"has_more"`
}

func NewMessageService(query *db.Queries) *MessageService {
	return &MessageService{query: query}
}

func (s *MessageService) GetHistory(ctx context.Context, userID string, req HistoryReq) (*HistoryResp, error) {
	roomID, memberID, err := parseRoomAndUser(req.RoomID, userID)
	if err != nil {
		return nil, err
	}
	isMember, err := s.query.IsRoomMember(ctx, db.IsRoomMemberParams{RoomID: roomID, UserID: memberID})
	if err != nil {
		return nil, fmt.Errorf("check history membership: %w", err)
	}
	if !isMember {
		return nil, ErrRoomAccessDenied
	}
	beforeTime := req.BeforeServerTime
	if beforeTime == 0 {
		beforeTime = time.Now().UnixMicro()
	}
	pageSize := normalizePageSize(req.PageSize)
	messages, err := s.query.ListMessagesByRoom(ctx, db.ListMessagesByRoomParams{
		RoomID:           roomID,
		BeforeServerTime: beforeTime,
		PageSize:         int32(pageSize + 1),
	})
	if err != nil {
		return nil, fmt.Errorf("list room history: %w", err)
	}
	hasMore := len(messages) > pageSize
	if hasMore {
		messages = messages[:pageSize]
	}
	result := make([]*MessageResp, 0, len(messages))
	for _, message := range messages {
		result = append(
			result,
			messageResponse(
				message.MsgID,
				message.ClientMsgID,
				message.SenderID,
				message.RoomID,
				message.ServerTime,
				message.ReplyToMsgID,
				message.MsgType,
				message.Payload,
				message.Ext,
				message.IsRecalled,
			),
		)
	}
	return &HistoryResp{Messages: result, HasMore: hasMore}, nil
}

func (s *MessageService) Sync(ctx context.Context, userID string, req SyncReq) (*SyncResp, error) {
	memberID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	pageSize := normalizePageSize(req.PageSize)
	messages, err := s.query.ListMessagesAfterTime(ctx, db.ListMessagesAfterTimeParams{
		UserID:          memberID,
		AfterServerTime: req.AfterServerTime,
		PageSize:        int32(pageSize + 1),
	})
	if err != nil {
		return nil, fmt.Errorf("sync messages: %w", err)
	}
	hasMore := len(messages) > pageSize
	if hasMore {
		messages = messages[:pageSize]
	}
	result := make([]*MessageResp, 0, len(messages))
	for _, message := range messages {
		result = append(
			result,
			messageResponse(
				message.MsgID,
				message.ClientMsgID,
				message.SenderID,
				message.RoomID,
				message.ServerTime,
				message.ReplyToMsgID,
				message.MsgType,
				message.Payload,
				message.Ext,
				message.IsRecalled,
			),
		)
	}
	return &SyncResp{Messages: result, HasMore: hasMore}, nil
}

func parseRoomAndUser(roomID, userID string) (uuid.UUID, uuid.UUID, error) {
	room, err := uuid.Parse(roomID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	user, err := uuid.Parse(userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalidInput
	}
	return room, user, nil
}

func normalizePageSize(pageSize int) int {
	if pageSize <= 0 {
		return 20
	}
	if pageSize > 100 {
		return 100
	}
	return pageSize
}

func messageResponse(
	msgID, clientMsgID, senderID, roomID uuid.UUID,
	serverTime int64,
	replyTo *uuid.UUID,
	msgType db.MessageType,
	payload, ext []byte,
	recalled bool,
) *MessageResp {
	response := &MessageResp{
		MsgID:       msgID.String(),
		ClientMsgID: clientMsgID.String(),
		SenderID:    senderID.String(),
		RoomID:      roomID.String(),
		ServerTime:  serverTime,
		MsgType:     string(msgType),
		Payload:     payload,
		Ext:         ext,
		Recalled:    recalled,
	}
	if replyTo != nil {
		response.ReplyToMsgID = replyTo.String()
	}
	return response
}

type ReadReq struct {
	RoomID             string `json:"room_id"               validate:"required,uuid"`
	LastReadServerTime int64  `json:"last_read_server_time" validate:"gte=0"`
}

type RecallResp struct {
	MsgID      string `json:"msg_id"`
	RoomID     string `json:"room_id"`
	ServerTime int64  `json:"server_time"`
}

func (s *MessageService) MarkRead(ctx context.Context, userID string, req ReadReq) error {
	room, user, err := parseRoomAndUser(req.RoomID, userID)
	if err != nil || req.LastReadServerTime < 0 {
		return ErrInvalidInput
	}
	member, err := s.query.IsRoomMember(ctx, db.IsRoomMemberParams{RoomID: room, UserID: user})
	if err != nil {
		return fmt.Errorf("check read membership: %w", err)
	}
	if !member {
		return ErrRoomAccessDenied
	}
	if err := s.query.UpdateRoomReadCursor(
		ctx,
		db.UpdateRoomReadCursorParams{RoomID: room, UserID: user, LastReadServerTime: req.LastReadServerTime},
	); err != nil {
		return fmt.Errorf("update read cursor: %w", err)
	}
	return nil
}

func (s *MessageService) Recall(ctx context.Context, userID, msgID string) (*RecallResp, error) {
	sender, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	msg, err := uuid.Parse(msgID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	row, err := s.query.RecallMessage(ctx, db.RecallMessageParams{MsgID: msg, SenderID: sender})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrRecallNotAllowed
	}
	if err != nil {
		return nil, fmt.Errorf("recall message: %w", err)
	}
	return &RecallResp{MsgID: row.MsgID.String(), RoomID: row.RoomID.String(), ServerTime: row.ServerTime}, nil
}
