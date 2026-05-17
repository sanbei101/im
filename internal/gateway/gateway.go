package gateway

import (
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
)

type Gateway struct {
	UserSessionManager *UserSessionManager
	Redis              *db.Redis
	Config             *config.Config
}

func NewGateway(cfg *config.Config) *Gateway {
	return &Gateway{
		UserSessionManager: NewSessionManager(),
		Redis:              db.NewRedis(cfg),
		Config:             cfg,
	}
}
