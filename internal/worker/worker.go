package worker

import (
	"context"
	"errors"
	"fmt"

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

func (s *Service) ProcessInbound(ctx context.Context, batchSize int64) error {
	streamMsgs, err := s.redis.WorkerPullMessage(ctx, batchSize)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Info().Msg("worker 收到退出信号,停止读取消息")
			return nil
		}
		return fmt.Errorf("worker xread failed: %w", err)
	}
	if len(streamMsgs) == 0 {
		return nil
	}

	params := make([]db.BatchCopyMessagesParams, 0, batchSize)
	msgIDs := make([]string, 0, batchSize)
	msgs := make([]*db.Message, 0, batchSize)

	for _, sm := range streamMsgs {
		msgIDs = append(msgIDs, sm.ID)
		chatMsg := sm.Data
		msgs = append(msgs, chatMsg)

		params = append(params, db.BatchCopyMessagesParams{
			MsgID:        chatMsg.MsgID,
			ClientMsgID:  chatMsg.ClientMsgID,
			SenderID:     chatMsg.SenderID,
			RoomID:       chatMsg.RoomID,
			ChatType:     chatMsg.ChatType,
			MsgType:      chatMsg.MsgType,
			ServerTime:   chatMsg.ServerTime,
			ReplyToMsgID: chatMsg.ReplyToMsgID,
			Payload:      chatMsg.Payload,
			Ext:          chatMsg.Ext,
		})
	}

	_, err = s.queries.BatchCopyMessages(ctx, params)
	if err != nil {
		return fmt.Errorf("batch copy messages failed: %w", err)
	}

	if err := s.redis.WorkerPushMessage(ctx, msgs); err != nil {
		return fmt.Errorf("worker publish deliver batch failed: %w", err)
	}

	if err := s.redis.WorkerAckMessage(ctx, msgIDs...); err != nil {
		return fmt.Errorf("worker ack messages failed: %w", err)
	}
	return nil
}
