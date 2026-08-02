package gateway

import (
	"github.com/sanbei101/im/internal/mq"
	"github.com/sanbei101/im/pkg/config"
)

type Gateway struct {
	UserSessionManager *UserSessionManager
	MQ                 mq.MQ
	Config             *config.Config
}

func NewGateway(cfg *config.Config, m mq.MQ) *Gateway {
	return &Gateway{
		UserSessionManager: NewSessionManager(),
		MQ:                 m,
		Config:             cfg,
	}
}
