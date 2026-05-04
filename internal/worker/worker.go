package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
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

	roomToMsgs := make(map[uuid.UUID][]*db.Message)

	for _, sm := range streamMsgs {
		msgIDs = append(msgIDs, sm.ID)
		chatMsg := sm.Data

		params = append(params, db.BatchCopyMessagesParams{
			MsgID:        chatMsg.MsgID,
			ClientMsgID:  chatMsg.ClientMsgID,
			SenderID:     chatMsg.SenderID,
			RoomID:       chatMsg.RoomID,
			MsgType:      chatMsg.MsgType,
			ServerTime:   chatMsg.ServerTime,
			ReplyToMsgID: chatMsg.ReplyToMsgID,
			Payload:      chatMsg.Payload,
			Ext:          chatMsg.Ext,
		})

		roomToMsgs[chatMsg.RoomID] = append(roomToMsgs[chatMsg.RoomID], chatMsg)
	}

	_, err = s.queries.BatchCopyMessages(ctx, params)
	if err != nil {
		return fmt.Errorf("batch copy messages failed: %w", err)
	}

	tasks, err := s.buildGatewayPushTasks(ctx, roomToMsgs)
	if err != nil {
		return fmt.Errorf("build gateway push tasks failed: %w", err)
	}

	if err := s.redis.WorkerPushGatewayTask(ctx, tasks); err != nil {
		return fmt.Errorf("worker publish deliver batch failed: %w", err)
	}

	if err := s.redis.WorkerAckMessage(ctx, msgIDs...); err != nil {
		return fmt.Errorf("worker ack messages failed: %w", err)
	}
	return nil
}

func (s *Service) buildGatewayPushTasks(ctx context.Context, roomToMsgs map[uuid.UUID][]*db.Message) ([]*db.GatewayPushTask, error) {
	tasks := make([]*db.GatewayPushTask, 0, len(roomToMsgs))

	for roomID, msgs := range roomToMsgs {
		if len(msgs) == 0 {
			continue
		}

		memberIDs, err := s.queries.GetRoomMembers(ctx, roomID)
		if err != nil {
			log.Error().Err(err).Str("room_id", roomID.String()).Msg("get room members failed")
			continue
		}

		if len(memberIDs) == 0 {
			continue
		}

		for _, msg := range msgs {
			task := db.AcquireGatewayPushTask()
			task.RoomID = msg.RoomID
			task.TargetUserIDs = append(task.TargetUserIDs, memberIDs...)
			task.Message = *msg
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}
