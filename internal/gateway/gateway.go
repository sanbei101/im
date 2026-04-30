package gateway

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
)

type Gateway struct {
	sessions sync.Map
	redis    *db.Redis
	config   *config.Config
	queries  *db.Queries
}

func New(cfg *config.Config) *Gateway {
	pool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("gateway connect postgres failed")
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("gateway ping postgres failed")
	}
	return &Gateway{
		redis:   db.NewRedis(cfg),
		config:  cfg,
		queries: db.New(pool),
	}
}
