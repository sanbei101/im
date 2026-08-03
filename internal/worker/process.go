package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/phuslu/log"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/db"
)

// protoToDBMessageType maps the proto MessageType enum back to the sqlc
// string enum used by the DB layer.
func protoToDBMessageType(t imv1.MessageType) db.MessageType {
	switch t {
	case imv1.MessageType_MESSAGE_TYPE_TEXT:
		return db.MessageTypeText
	case imv1.MessageType_MESSAGE_TYPE_IMAGE:
		return db.MessageTypeImage
	case imv1.MessageType_MESSAGE_TYPE_VIDEO:
		return db.MessageTypeVideo
	case imv1.MessageType_MESSAGE_TYPE_FILE:
		return db.MessageTypeFile
	case imv1.MessageType_MESSAGE_TYPE_SYSTEM:
		return db.MessageTypeSystem
	}
	return ""
}

// messageToDBParams extracts the DB insert params from a proto message.
func messageToDBParams(p *imv1.Message) (db.BatchCopyMessagesParams, error) {
	msgID, err := uuid.Parse(p.GetMsgId())
	if err != nil {
		return db.BatchCopyMessagesParams{}, fmt.Errorf("invalid msg_id: %w", err)
	}
	clientMsgID, err := uuid.Parse(p.GetClientMsgId())
	if err != nil {
		return db.BatchCopyMessagesParams{}, fmt.Errorf("invalid client_msg_id: %w", err)
	}
	senderID, err := uuid.Parse(p.GetSenderId())
	if err != nil {
		return db.BatchCopyMessagesParams{}, fmt.Errorf("invalid sender_id: %w", err)
	}
	roomID, err := uuid.Parse(p.GetRoomId())
	if err != nil {
		return db.BatchCopyMessagesParams{}, fmt.Errorf("invalid room_id: %w", err)
	}

	var replyTo *uuid.UUID
	if s := p.GetReplyToMsgId(); s != "" {
		u, err := uuid.Parse(s)
		if err != nil {
			return db.BatchCopyMessagesParams{}, fmt.Errorf("invalid reply_to_msg_id: %w", err)
		}
		replyTo = &u
	}

	return db.BatchCopyMessagesParams{
		MsgID:        msgID,
		ClientMsgID:  clientMsgID,
		SenderID:     senderID,
		RoomID:       roomID,
		MsgType:      protoToDBMessageType(p.GetMsgType()),
		ServerTime:   p.GetServerTime(),
		ReplyToMsgID: replyTo,
		Payload:      p.GetPayload(),
		Ext:          p.GetExt(),
	}, nil
}

func (s *Service) ProcessInbound(ctx context.Context, batchSize int64) error {
	streamMsgs, err := s.mq.WorkerPullMessage(ctx, batchSize)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Info().Msg("worker 收到退出信号,停止读取消息")
			return nil
		}
		return fmt.Errorf("worker xread failed: %w", err)
	}
	if len(streamMsgs) == 0 {
		return nil
	}

	params := make([]db.BatchCopyMessagesParams, 0, batchSize)
	msgIDs := make([]string, 0, batchSize)

	roomToMsgs := make(map[uuid.UUID][]*imv1.Message)

	for _, sm := range streamMsgs {
		msgIDs = append(msgIDs, sm.StreamID)

		p, err := messageToDBParams(sm.Payload)
		if err != nil {
			log.Error().Err(err).Str("stream_id", sm.StreamID).Msg("invalid message payload, dropping")
			continue
		}
		params = append(params, p)
		roomToMsgs[p.RoomID] = append(roomToMsgs[p.RoomID], sm.Payload)
	}

	rowsInserted, err := s.queries.BatchCopyMessages(ctx, params)
	if err != nil {
		log.Error().Err(err).Msg("批量插入消息失败")
		return fmt.Errorf("batch copy messages failed: %w", err)
	}
	if rowsInserted != int64(len(params)) {
		log.Warn().
			Int64("rowsInserted", rowsInserted).
			Int("paramsLength", len(params)).
			Any("params", params).
			Msg("批量插入消息行数与参数长度不匹配")
	}

	tasks, err := s.buildDeliveryTasks(ctx, roomToMsgs)
	if err != nil {
		return fmt.Errorf("build delivery tasks failed: %w", err)
	}

	if err := s.mq.WorkerEnqueueDeliveryTask(ctx, tasks); err != nil {
		return fmt.Errorf("worker publish deliver batch failed: %w", err)
	}

	if err := s.mq.WorkerAckMessage(ctx, msgIDs...); err != nil {
		return fmt.Errorf("worker ack messages failed: %w", err)
	}
	return nil
}
