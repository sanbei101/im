package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/phuslu/log"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/mq"
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

// messageToDBParams validates untrusted gateway data before it reaches SQL.
func messageToDBParams(p *imv1.Message) (db.InsertMessageParams, error) {
	if p == nil {
		return db.InsertMessageParams{}, errors.New("nil message")
	}
	msgID, err := uuid.Parse(p.GetMsgId())
	if err != nil {
		return db.InsertMessageParams{}, fmt.Errorf("invalid msg_id: %w", err)
	}
	clientMsgID, err := uuid.Parse(p.GetClientMsgId())
	if err != nil {
		return db.InsertMessageParams{}, fmt.Errorf("invalid client_msg_id: %w", err)
	}
	senderID, err := uuid.Parse(p.GetSenderId())
	if err != nil {
		return db.InsertMessageParams{}, fmt.Errorf("invalid sender_id: %w", err)
	}
	roomID, err := uuid.Parse(p.GetRoomId())
	if err != nil {
		return db.InsertMessageParams{}, fmt.Errorf("invalid room_id: %w", err)
	}
	msgType := protoToDBMessageType(p.GetMsgType())
	if msgType == "" {
		return db.InsertMessageParams{}, fmt.Errorf("invalid msg_type: %v", p.GetMsgType())
	}

	var replyTo *uuid.UUID
	if s := p.GetReplyToMsgId(); s != "" {
		u, err := uuid.Parse(s)
		if err != nil {
			return db.InsertMessageParams{}, fmt.Errorf("invalid reply_to_msg_id: %w", err)
		}
		replyTo = &u
	}

	return db.InsertMessageParams{
		MsgID:        msgID,
		ClientMsgID:  clientMsgID,
		SenderID:     senderID,
		RoomID:       roomID,
		MsgType:      msgType,
		ServerTime:   p.GetServerTime(),
		ReplyToMsgID: replyTo,
		Payload:      p.GetPayload(),
		Ext:          p.GetExt(),
	}, nil
}

func persistedMessage(
	msgID, clientMsgID, senderID, roomID uuid.UUID,
	msgType db.MessageType,
	serverTime int64,
	replyTo *uuid.UUID,
	payload, ext []byte,
) *imv1.Message {
	msgIDStr, clientMsgIDStr := msgID.String(), clientMsgID.String()
	senderIDStr, roomIDStr := senderID.String(), roomID.String()
	protoType := dbMessageTypeToProto(msgType)
	return &imv1.Message{
		MsgId:        &msgIDStr,
		ClientMsgId:  &clientMsgIDStr,
		SenderId:     &senderIDStr,
		RoomId:       &roomIDStr,
		MsgType:      &protoType,
		ServerTime:   &serverTime,
		ReplyToMsgId: uuidString(replyTo),
		Payload:      payload,
		Ext:          ext,
	}
}

func dbMessageTypeToProto(t db.MessageType) imv1.MessageType {
	switch t {
	case db.MessageTypeText:
		return imv1.MessageType_MESSAGE_TYPE_TEXT
	case db.MessageTypeImage:
		return imv1.MessageType_MESSAGE_TYPE_IMAGE
	case db.MessageTypeVideo:
		return imv1.MessageType_MESSAGE_TYPE_VIDEO
	case db.MessageTypeFile:
		return imv1.MessageType_MESSAGE_TYPE_FILE
	case db.MessageTypeSystem:
		return imv1.MessageType_MESSAGE_TYPE_SYSTEM
	default:
		return imv1.MessageType_MESSAGE_TYPE_UNSPECIFIED
	}
}

func uuidString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

const maxInboundRetries int64 = 3

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

	// ponytail: sequential inserts keep idempotency and permission checks in one
	// obvious path; batch transactions can be added after throughput is measured.
	ackIDs := make([]string, 0, len(streamMsgs))
	roomToMsgs := make(map[uuid.UUID][]*imv1.Message)
	failedTasks := make([]*mq.DeliverTaskEnvelope, 0)
	persistedTasks := make([]*mq.DeliverTaskEnvelope, 0)
	for _, sm := range streamMsgs {
		if sm == nil {
			continue
		}
		if sm.Payload == nil {
			if sm.RetryCount >= maxInboundRetries {
				if err := s.deadLetter(ctx, sm, "invalid protobuf payload"); err != nil {
					return err
				}
				ackIDs = append(ackIDs, sm.StreamID)
				continue
			}
			return fmt.Errorf("inbound stream %s has invalid payload (retry %d)", sm.StreamID, sm.RetryCount)
		}

		params, err := messageToDBParams(sm.Payload)
		if err != nil {
			log.Error().
				Err(err).
				Str("stream_id", sm.StreamID).
				Str("client_msg_id", sm.Payload.GetClientMsgId()).
				Msg("invalid message payload")
			failedTasks = appendFailureTask(failedTasks, sm, "invalid_message", "invalid message")
			ackIDs = append(ackIDs, sm.StreamID)
			continue
		}

		member, err := s.queries.IsRoomMember(
			ctx,
			db.IsRoomMemberParams{RoomID: params.RoomID, UserID: params.SenderID},
		)
		if err != nil {
			if sm.RetryCount >= maxInboundRetries {
				if dlErr := s.deadLetter(ctx, sm, "room membership check failed: "+err.Error()); dlErr != nil {
					return dlErr
				}
				failedTasks = appendFailureTask(failedTasks, sm, "unavailable", "message processing unavailable")
				ackIDs = append(ackIDs, sm.StreamID)
				continue
			}
			return fmt.Errorf("check room membership: %w", err)
		}
		if !member {
			log.Warn().
				Str("stream_id", sm.StreamID).
				Str("sender_id", params.SenderID.String()).
				Str("room_id", params.RoomID.String()).
				Msg("reject message from non-member")
			failedTasks = appendFailureTask(failedTasks, sm, "room_access_denied", "room access denied")
			ackIDs = append(ackIDs, sm.StreamID)
			continue
		}

		row, err := s.queries.InsertMessage(ctx, params)
		if errors.Is(err, pgx.ErrNoRows) {
			// ponytail: re-deliver the idempotent row on retry; clients dedupe by
			// msg_id, avoiding a second outbox table in the MVP.
			existing, getErr := s.queries.GetMessageByClientID(ctx, db.GetMessageByClientIDParams{
				SenderID: params.SenderID, ClientMsgID: params.ClientMsgID,
			})
			if getErr != nil {
				if sm.RetryCount >= maxInboundRetries {
					if dlErr := s.deadLetter(ctx, sm, "load idempotent message failed: "+getErr.Error()); dlErr != nil {
						return dlErr
					}
					failedTasks = appendFailureTask(failedTasks, sm, "unavailable", "message processing unavailable")
					ackIDs = append(ackIDs, sm.StreamID)
					continue
				}
				return fmt.Errorf("load idempotent message: %w", getErr)
			}
			persisted := persistedMessage(
				existing.MsgID, existing.ClientMsgID, existing.SenderID, existing.RoomID,
				existing.MsgType, existing.ServerTime, existing.ReplyToMsgID, existing.Payload, existing.Ext,
			)
			roomToMsgs[existing.RoomID] = append(roomToMsgs[existing.RoomID], persisted)
			persistedTasks = append(persistedTasks, persistedDeliveryTask(persisted))
			ackIDs = append(ackIDs, sm.StreamID)
			continue
		}
		if err != nil {
			if sm.RetryCount >= maxInboundRetries {
				if dlErr := s.deadLetter(ctx, sm, "insert message failed: "+err.Error()); dlErr != nil {
					return dlErr
				}
				failedTasks = appendFailureTask(failedTasks, sm, "unavailable", "message processing unavailable")
				ackIDs = append(ackIDs, sm.StreamID)
				continue
			}
			return fmt.Errorf("insert message: %w", err)
		}
		persisted := persistedMessage(
			row.MsgID, row.ClientMsgID, row.SenderID, row.RoomID,
			row.MsgType, row.ServerTime, row.ReplyToMsgID, row.Payload, row.Ext,
		)
		roomToMsgs[row.RoomID] = append(roomToMsgs[row.RoomID], persisted)
		persistedTasks = append(persistedTasks, persistedDeliveryTask(persisted))
		ackIDs = append(ackIDs, sm.StreamID)
	}

	tasks, err := s.buildDeliveryTasks(ctx, roomToMsgs)
	if err != nil {
		return fmt.Errorf("build delivery tasks failed: %w", err)
	}
	tasks = append(tasks, persistedTasks...)
	tasks = append(tasks, failedTasks...)
	if err := s.mq.WorkerEnqueueDeliveryTask(ctx, tasks); err != nil {
		releaseDeliveryTasks(tasks)
		return fmt.Errorf("worker publish deliver batch failed: %w", err)
	}
	releaseDeliveryTasks(tasks)
	if err := s.mq.WorkerAckMessage(ctx, ackIDs...); err != nil {
		return fmt.Errorf("worker ack messages failed: %w", err)
	}
	return nil
}

func (s *Service) deadLetter(ctx context.Context, msg *mq.InboundMsgEnvelope, reason string) error {
	if err := s.mq.WorkerEnqueueDeadLetter(ctx, msg, reason); err != nil {
		return fmt.Errorf("write dead-letter for stream %s: %w", msg.StreamID, err)
	}
	log.Error().
		Str("stream_id", msg.StreamID).
		Int64("retry_count", msg.RetryCount).
		Msg("inbound message moved to dead-letter stream")
	return nil
}

func persistedDeliveryTask(msg *imv1.Message) *mq.DeliverTaskEnvelope {
	clientMsgID := msg.GetClientMsgId()
	msgID := msg.GetMsgId()
	serverTime := msg.GetServerTime()
	task := mq.AcquireDeliveryTask()
	task.Payload.TargetUserIds = []string{msg.GetSenderId()}
	task.Payload.ClientMsgId = &clientMsgID
	task.Payload.Persisted = &imv1.MessagePersisted{MsgId: &msgID, ServerTime: &serverTime}
	return task
}

func appendFailureTask(
	tasks []*mq.DeliverTaskEnvelope,
	msg *mq.InboundMsgEnvelope,
	code, message string,
) []*mq.DeliverTaskEnvelope {
	if msg == nil || msg.Payload == nil || msg.Payload.GetSenderId() == "" {
		return tasks
	}
	senderID, err := uuid.Parse(msg.Payload.GetSenderId())
	if err != nil {
		return tasks
	}
	clientMsgID := msg.Payload.GetClientMsgId()
	roomID := msg.Payload.GetRoomId()
	task := mq.AcquireDeliveryTask()
	task.Payload.RoomId = &roomID
	task.Payload.TargetUserIds = []string{senderID.String()}
	task.Payload.ClientMsgId = &clientMsgID
	task.Payload.Failed = &imv1.MessageFailed{Code: &code, Message: &message}
	return append(tasks, task)
}

func releaseDeliveryTasks(tasks []*mq.DeliverTaskEnvelope) {
	for _, task := range tasks {
		if task != nil {
			mq.ReleaseDeliveryTask(task)
		}
	}
}
