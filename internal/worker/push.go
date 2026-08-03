package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/phuslu/log"
	"github.com/phuslu/lru"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/mq"
)

var roomMembersCache = lru.NewTTLCache[uuid.UUID, []uuid.UUID](10000)

func (s *Service) getRoomMembersWithCache(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	result := make(map[uuid.UUID][]uuid.UUID, len(roomIDs))
	var missingRoomIDs []uuid.UUID

	for _, roomID := range roomIDs {
		if members, ok := roomMembersCache.Get(roomID); ok {
			result[roomID] = members
		} else {
			missingRoomIDs = append(missingRoomIDs, roomID)
		}
	}

	if len(missingRoomIDs) == 0 {
		return result, nil
	}

	memberRows, err := s.queries.GetMembersByRoomIDs(ctx, missingRoomIDs)
	if err != nil {
		return nil, err
	}

	missingMap := make(map[uuid.UUID][]uuid.UUID)
	for _, row := range memberRows {
		missingMap[row.RoomID] = append(missingMap[row.RoomID], row.UserID)
	}

	for _, roomID := range missingRoomIDs {
		members := missingMap[roomID]
		roomMembersCache.Set(roomID, members, 5*time.Minute)
		result[roomID] = members
	}
	return result, nil
}

// buildDeliveryTasks 为每个 room 内的消息生成一条 DeliveryTask,目标用户
// 已剔除发送者本人 (sender 通过 SendMessageAck 同步拿到自己的消息,无需重复推送).
func (s *Service) buildDeliveryTasks(
	ctx context.Context,
	roomToMsgs map[uuid.UUID][]*imv1.Message,
) ([]*mq.DeliverTaskEnvelope, error) {
	var totalMsgs int
	roomIDs := make([]uuid.UUID, 0, len(roomToMsgs))
	for roomID, msgs := range roomToMsgs {
		if len(msgs) > 0 {
			totalMsgs += len(msgs)
			roomIDs = append(roomIDs, roomID)
		}
	}

	if len(roomIDs) == 0 {
		return nil, nil
	}

	roomMembers, err := s.getRoomMembersWithCache(ctx, roomIDs)
	if err != nil {
		log.Error().Err(err).Msg("batch get room members with cache failed")
		return nil, err
	}

	tasks := make([]*mq.DeliverTaskEnvelope, 0, totalMsgs)

	for _, roomID := range roomIDs {
		msgs := roomToMsgs[roomID]
		memberIDs := roomMembers[roomID]

		if len(memberIDs) == 0 {
			continue
		}

		for _, msg := range msgs {
			senderID, err := uuid.Parse(msg.GetSenderId())
			if err != nil {
				log.Warn().Err(err).Str("msg_id", msg.GetMsgId()).Msg("skip message with invalid sender_id")
				continue
			}

			// 剔除 sender
			targets := make([]string, 0, len(memberIDs))
			for _, mid := range memberIDs {
				if mid != senderID {
					targets = append(targets, mid.String())
				}
			}
			if len(targets) == 0 {
				continue
			}

			roomIDStr := roomID.String()
			task := mq.AcquireDeliveryTask()
			task.Payload.RoomId = &roomIDStr
			task.Payload.TargetUserIds = targets
			task.Payload.Message = msg

			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}
