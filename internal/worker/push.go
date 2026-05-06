package worker

import (
	"context"

	"github.com/google/uuid"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
)

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

	// 批量查询房间成员
	memberRows, err := s.queries.GetMembersByRoomIDs(ctx, roomIDs)
	if err != nil {
		log.Error().Err(err).Msg("batch get room members failed")
		return nil, err
	}

	roomMembers := make(map[uuid.UUID][]uuid.UUID)
	for _, row := range memberRows {
		roomMembers[row.RoomID] = append(roomMembers[row.RoomID], row.UserID)
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

			// 确保有足够的容量
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
