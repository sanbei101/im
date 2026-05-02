package gateway

import (
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
)

type Gateway struct {
	sessions *SessionManager[*UserSession]
	redis    *db.Redis
	config   *config.Config
}

func New(cfg *config.Config) *Gateway {
	return &Gateway{
		sessions: NewSessionManager[*UserSession](),
		redis:    db.NewRedis(cfg),
		config:   cfg,
	}
}
