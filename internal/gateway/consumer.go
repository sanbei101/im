package gateway

import (
	"context"
	"encoding/json/v2"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
)

func (gateway *Gateway) HandleWorkerMessages(ctx context.Context) {
	err := gateway.redis.InitStreamGroups(context.Background())
	if err != nil {
		log.Panic().Err(err).Msg("gateway init stream groups failed")
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			gateway.pollAndProcess(ctx)
		}
	}
}

func (gateway *Gateway) pollAndProcess(ctx context.Context) {
	tasks, err := gateway.redis.GatewayPullTask(ctx, 1000)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(err).Msg("gateway pull task failed")
		return
	}

	if len(tasks) == 0 {
		return
	}

	gateway.processTasks(ctx, tasks)
}

func (gateway *Gateway) processTasks(ctx context.Context, tasks []*db.GatewayPushTask) {
	streamIDs := make([]string, 0, len(tasks))

	// 按用户分组,收集每个用户的消息
	userMessages := make(map[string][][]byte)
	for _, task := range tasks {
		streamIDs = append(streamIDs, task.StreamID)

		bin, marshalErr := json.Marshal(task.Message)
		if marshalErr != nil {
			log.Error().Err(marshalErr).Msg("gateway marshal message failed")
			continue
		}

		for _, userID := range task.TargetUserIDs {
			userIDStr := userID.String()
			userMessages[userIDStr] = append(userMessages[userIDStr], bin)
		}
	}

	// 按用户批量广播
	for userIDStr, msgs := range userMessages {
		if userSession, ok := gateway.sessions.Load(userIDStr); ok {
			userSession.Broadcast(msgs)
		}
	}

	err := gateway.redis.GatewayAckMessage(ctx, streamIDs...)
	if err != nil {
		log.Error().Err(err).Msg("gateway ack messages failed")
	}
}
