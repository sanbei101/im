package gateway

import (
	"sync"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
)

type Gateway struct {
	sessions sync.Map
	redis    *db.Redis
	config   *config.Config
}

func New(cfg *config.Config) *Gateway {
	return &Gateway{
		redis:  db.NewRedis(cfg),
		config: cfg,
	}
}
