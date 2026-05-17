package gateway

import (
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
)

type Gateway struct {
	sessions *UserSessionManager
	redis    *db.Redis
	config   *config.Config
}

func New(cfg *config.Config) *Gateway {
	return &Gateway{
		sessions: NewSessionManager(),
		redis:    db.NewRedis(cfg),
		config:   cfg,
	}
}
