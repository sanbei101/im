package gateway

import (
	"context"

	"github.com/sanbei101/im/internal/mq"
	"github.com/sanbei101/im/pkg/config"
)

type RoomAccessChecker interface {
	CanSend(ctx context.Context, roomID, userID string) (bool, error)
}

type Gateway struct {
	UserSessionManager *UserSessionManager
	MQ                 mq.MQ
	RoomAccess         RoomAccessChecker
	Config             *config.Config
}

func NewGateway(cfg *config.Config, m mq.MQ) *Gateway {
	access, _ := m.(RoomAccessChecker)
	return &Gateway{
		UserSessionManager: NewSessionManager(),
		MQ:                 m,
		RoomAccess:         access,
		Config:             cfg,
	}
}
