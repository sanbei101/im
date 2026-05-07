package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/phuslu/log"
	"github.com/phuslu/lru"
	"github.com/sanbei101/im/internal/db"
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

func (s *Service) buildGatewayPushTasks(ctx context.Context, roomToMsgs map[uuid.UUID][]*db.Message) ([]*db.GatewayPushTask, error) {
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

	tasks := make([]*db.GatewayPushTask, 0, totalMsgs)

	for _, roomID := range roomIDs {
		msgs := roomToMsgs[roomID]
		memberIDs := roomMembers[roomID]

		if len(memberIDs) == 0 {
			continue
		}

		for _, msg := range msgs {
			task := db.AcquireGatewayPushTask()
			task.RoomID = msg.RoomID

			if cap(task.TargetUserIDs) < len(memberIDs) {
				task.TargetUserIDs = make([]uuid.UUID, 0, len(memberIDs))
			} else {
				task.TargetUserIDs = task.TargetUserIDs[:0]
			}
			task.TargetUserIDs = append(task.TargetUserIDs, memberIDs...)

			task.Message = *msg
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}
