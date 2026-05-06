package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
)

func (s *Service) ProcessInbound(ctx context.Context, batchSize int64) error {
	streamMsgs, err := s.redis.WorkerPullMessage(ctx, batchSize)
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

	roomToMsgs := make(map[uuid.UUID][]*db.Message)

	for _, sm := range streamMsgs {
		msgIDs = append(msgIDs, sm.ID)
		chatMsg := sm.Data

		params = append(params, db.BatchCopyMessagesParams{
			MsgID:        chatMsg.MsgID,
			ClientMsgID:  chatMsg.ClientMsgID,
			SenderID:     chatMsg.SenderID,
			RoomID:       chatMsg.RoomID,
			MsgType:      chatMsg.MsgType,
			ServerTime:   chatMsg.ServerTime,
			ReplyToMsgID: chatMsg.ReplyToMsgID,
			Payload:      chatMsg.Payload,
			Ext:          chatMsg.Ext,
		})

		roomToMsgs[chatMsg.RoomID] = append(roomToMsgs[chatMsg.RoomID], chatMsg)
	}

	_, err = s.queries.BatchCopyMessages(ctx, params)
	if err != nil {
		return fmt.Errorf("batch copy messages failed: %w", err)
	}

	tasks, err := s.buildGatewayPushTasks(ctx, roomToMsgs)
	if err != nil {
		return fmt.Errorf("build gateway push tasks failed: %w", err)
	}

	if err := s.redis.WorkerPushGatewayTask(ctx, tasks); err != nil {
		return fmt.Errorf("worker publish deliver batch failed: %w", err)
	}

	if err := s.redis.WorkerAckMessage(ctx, msgIDs...); err != nil {
		return fmt.Errorf("worker ack messages failed: %w", err)
	}
	return nil
}
