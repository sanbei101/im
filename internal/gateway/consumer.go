package gateway

import (
	"context"
	"encoding/json/v2"
	"time"

	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
)

func (gateway *Gateway) HandleWorkerMessages(ctx context.Context) {
	if err := gateway.redis.InitStreamGroups(ctx); err != nil {
		log.Warn().Err(err).Msg("gateway consumer group mkstream failed")
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
	messages, err := gateway.redis.GatewayPullMessage(ctx, 1000)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Error().Err(err).Msg("gateway pull message failed")
		time.Sleep(time.Second)
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

		roomID := msg.Data.RoomID
		members, err := gateway.queries.GetRoomMembers(ctx, roomID)
		if err != nil {
			log.Error().Err(err).Str("room_id", roomID.String()).Msg("gateway get room members failed")
			continue
		}

		for _, memberID := range members {
			gateway.deliverToClient(memberID.String(), bin)
		}
	}
	err := gateway.redis.GatewayAckMessage(ctx, msgIDs...)
	if err != nil {
		log.Error().Err(err).Msg("gateway ack messages failed")
	}
}

func (gateway *Gateway) deliverToClient(userID string, payload []byte) {
	if sessionIface, ok := gateway.sessions.Load(userID); ok {
		session := sessionIface.(*UserSession)
		session.Broadcast(payload)
	} else {
		log.Debug().Str("user_id", userID).Msg("user not connected to this gateway instance")
	}
}
