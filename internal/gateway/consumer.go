package gateway

import (
	"context"
	"encoding/json/v2"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
)

func (gateway *Gateway) HandleWorkerMessages(ctx context.Context) {
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
	messages, err := gateway.redis.GatewayPullMessage(ctx, 1000)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(err).Msg("gateway pull message failed")
		return
	}

	if len(messages) == 0 {
		return
	}

	gateway.processMessages(ctx, messages)
}

func (gateway *Gateway) processMessages(ctx context.Context, messages []*db.StreamMessage) {
	var msgIDs []string
	for _, msg := range messages {
		msgIDs = append(msgIDs, msg.ID)

		bin, marshalErr := json.Marshal(msg.Data)
		if marshalErr != nil {
			log.Error().Err(marshalErr).Msg("gateway marshal message failed")
			continue
		}

		userIDStr := msg.Data.SenderID.String()

		if userSession, ok := gateway.sessions.Load(userIDStr); ok {
			userSession.Broadcast(bin)
		}
	}

	err := gateway.redis.GatewayAckMessage(ctx, msgIDs...)
	if err != nil {
		log.Error().Err(err).Msg("gateway ack messages failed")
	}
}
