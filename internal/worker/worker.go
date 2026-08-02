package worker

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/mq"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/logger"
)

const (
	BatchReadSize = 1000
)

type Service struct {
	mq      mq.MQ
	queries *db.Queries
}

func New(cfg *config.Config, m mq.MQ) *Service {
	config, err := pgxpool.ParseConfig(cfg.Postgres.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("postgres parse config failed")
	}
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   logger.NewPgxLogger(),
		LogLevel: tracelog.LogLevelDebug,
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal().Err(err).Msg("worker connect postgres failed")
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("worker ping postgres failed")
	}
	return &Service{
		mq:      m,
		queries: db.New(pool),
	}
}

func (s *Service) Run(ctx context.Context) {
	if err := s.mq.InitStreamGroups(ctx); err != nil {
		log.Panic().Err(err).Msg("worker consume group init failed")
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
