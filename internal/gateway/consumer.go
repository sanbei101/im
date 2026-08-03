package gateway

import (
	"context"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/mq"
)

func (gateway *Gateway) HandleWorkerMessages(ctx context.Context) {
	err := gateway.MQ.InitStreamGroups(context.Background())
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
	tasks, err := gateway.MQ.GatewayPullDeliveryTask(ctx, 1000)
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

func (gateway *Gateway) processTasks(ctx context.Context, tasks []*mq.DeliverTaskEnvelope) {
	streamIDs := make([]string, 0, len(tasks))
	userMessages := make(map[string][]byte)

	for _, task := range tasks {
		if task.Payload == nil || task.Payload.Message == nil {
			log.Error().Str("stream_id", task.StreamID).Msg("nil Message in delivery task")
			mq.ReleaseDeliveryTask(task)
			continue
		}
		streamIDs = append(streamIDs, task.StreamID)

		push := task.Payload.Message
		bin := make([]byte, push.SizeVT())
		n, err := push.MarshalToVT(bin)
		if err != nil {
			log.Error().Err(err).Str("stream_id", task.StreamID).Msg("marshal Message failed")
			continue
		}
		bin = bin[:n]

		for _, uid := range task.Payload.GetTargetUserIds() {
			if userSession, ok := gateway.UserSessionManager.Load(uid); ok {
				userSession.Broadcast(bin)
			}
		}

		mq.ReleaseDeliveryTask(task)
	}

	for uid, msgs := range userMessages {
		if userSession, ok := gateway.UserSessionManager.Load(uid); ok {
			userSession.Broadcast(msgs)
		}
	}

	if err := gateway.MQ.GatewayAckMessage(ctx, streamIDs...); err != nil {
		log.Error().Err(err).Msg("gateway ack messages failed")
	}
}
