package gateway

import (
	"context"

	"github.com/phuslu/log"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/mq"
)

func (gateway *Gateway) HandleWorkerMessages(ctx context.Context) {
	err := gateway.MQ.InitStreamGroups(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("gateway init stream groups failed")
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

	for _, task := range tasks {
		if task == nil {
			continue
		}
		streamIDs = append(streamIDs, task.StreamID)
		if task.Payload == nil {
			log.Error().Str("stream_id", task.StreamID).Msg("nil delivery task payload")
			mq.ReleaseDeliveryTask(task)
			continue
		}

		var frame *imv1.ServerFrame
		switch {
		case task.Payload.GetMessage() != nil:
			frame = &imv1.ServerFrame{Payload: &imv1.ServerFrame_Message{Message: task.Payload.GetMessage()}}
		case task.Payload.GetFailed() != nil:
			clientMsgID := task.Payload.GetClientMsgId()
			frame = &imv1.ServerFrame{
				ClientMsgId: &clientMsgID,
				Payload:     &imv1.ServerFrame_Failed{Failed: task.Payload.GetFailed()},
			}
		case task.Payload.GetPersisted() != nil:
			clientMsgID := task.Payload.GetClientMsgId()
			frame = &imv1.ServerFrame{
				ClientMsgId: &clientMsgID,
				Payload:     &imv1.ServerFrame_Persisted{Persisted: task.Payload.GetPersisted()},
			}
		default:
			log.Error().Str("stream_id", task.StreamID).Msg("delivery task has no payload")
			mq.ReleaseDeliveryTask(task)
			continue
		}

		bin, err := frame.MarshalVT()
		if err != nil {
			log.Error().Err(err).Str("stream_id", task.StreamID).Msg("marshal delivery ServerFrame failed")
			mq.ReleaseDeliveryTask(task)
			continue
		}
		for _, uid := range task.Payload.GetTargetUserIds() {
			if userSession, ok := gateway.UserSessionManager.Load(uid); ok {
				userSession.Broadcast(bin)
			}
		}
		mq.ReleaseDeliveryTask(task)
	}

	if err := gateway.MQ.GatewayAckMessage(ctx, streamIDs...); err != nil {
		log.Error().Err(err).Msg("gateway ack messages failed")
	}
}
