package worker

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/pkg/config"
)

const (
	BatchReadSize = 100
)

type Service struct {
	redis   *db.Redis
	queries *db.Queries
}

func New(cfg *config.Config) *Service {
	pool, err := pgxpool.New(context.Background(), cfg.Postgres.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("worker connect postgres failed")
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("worker ping postgres failed")
	}
	return &Service{
		redis:   db.NewRedis(cfg),
		queries: db.New(pool),
	}
}

func (s *Service) Run(ctx context.Context) {
	if err := s.redis.InitStreamGroups(ctx); err != nil {
		log.Warn().Err(err).Msg("worker consume group init failed")
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := s.ProcessInbound(ctx, BatchReadSize)
			if err != nil {
				log.Error().Err(err).Msg("worker process inbound failed")
			}
		}
	}
}
