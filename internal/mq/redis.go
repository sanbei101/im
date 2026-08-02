package mq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/phuslu/log"
	"github.com/redis/go-redis/v9"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/pkg/config"
)

// WorkerName / GatewayName identify this process to the broker. They are
// derived from a stable machine fingerprint so consumers stick to the same
// pending entry list across restarts.
var (
	WorkerName  = "worker-" + machineFingerprint
	GatewayName = "gateway-" + machineFingerprint
)

const (
	streamInbound = "message:inbound"
	streamDeliver = "message:deliver"
	workerGroup   = "worker_group"
	gatewayGroup  = "gateway_group"
	streamMaxLen  = 1_000_000
)

// RedisMQ is the Redis Streams implementation of MQ.
type RedisMQ struct {
	client *redis.Client
}

// NewRedisMQ builds a RedisMQ backed by the Redis instance configured in
// `cfg`. Pool size is fixed at 50 connections: enough for the gateway's
// publish path and the worker's batch consumer without oversubscription.
func NewRedisMQ(cfg *config.Config) *RedisMQ {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: 50,
	})

	return &RedisMQ{client: client}
}

// InitStreamGroups creates the consumer groups for both streams. BUSYGROUP
// is tolerated so restarts are safe.
func (r *RedisMQ) InitStreamGroups(ctx context.Context) error {
	groups := map[string]string{
		streamInbound: workerGroup,
		streamDeliver: gatewayGroup,
	}
	for stream, group := range groups {
		err := r.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create group %s failed: %w", group, err)
		}
	}
	return nil
}

// WorkerPullMessage reads up to `batch` inbound messages from the worker
// group. Returns nil, nil if there is nothing to read.
func (r *RedisMQ) WorkerPullMessage(ctx context.Context, batch int64) ([]*StreamMessage, error) {
	return r.pullFromStream(ctx, streamInbound, workerGroup, WorkerName, batch)
}

// WorkerPushGatewayTask pipelines each task into the deliver stream with an
// approximate max length cap.
func (r *RedisMQ) WorkerPushGatewayTask(ctx context.Context, tasks []*GatewayPushTask) error {
	if len(tasks) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, task := range tasks {
		if task == nil {
			log.Error().Msg("Skipping nil task pointer")
			continue
		}
		p := task.Proto()
		bin, err := p.MarshalVT()
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal GatewayPushTask")
			continue
		}
		p.Reset()
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: streamDeliver,
			MaxLen: streamMaxLen,
			Approx: true,
			Values: map[string]any{"data": unsafe.String(unsafe.SliceData(bin), len(bin))},
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}

// WorkerAckMessage confirms the given inbound stream IDs.
func (r *RedisMQ) WorkerAckMessage(ctx context.Context, ids ...string) error {
	return r.ack(ctx, streamInbound, workerGroup, ids...)
}

// GatewayPullTask reads up to `batch` deliver tasks for the gateway.
func (r *RedisMQ) GatewayPullTask(ctx context.Context, batch int64) ([]*GatewayPushTask, error) {
	result, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    gatewayGroup,
		Consumer: GatewayName,
		Streams:  []string{streamDeliver, ">"},
		Count:    batch,
		Block:    5 * time.Second,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("xread group failed: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}

	tasks := make([]*GatewayPushTask, 0, len(result[0].Messages))
	for _, msg := range result[0].Messages {
		data, ok := msg.Values["data"].(string)
		if !ok {
			log.Error().Str("id", msg.ID).Msg("Missing 'data' field in stream message")
			continue
		}
		p := &imv1.GatewayPushTask{}
		if err := p.UnmarshalVT(unsafe.Slice(unsafe.StringData(data), len(data))); err != nil {
			log.Error().Str("id", msg.ID).Err(err).Msg("Failed to unmarshal GatewayPushTask")
			continue
		}
		task := AcquireGatewayPushTask()
		if err := task.FromProto(p); err != nil {
			log.Error().Str("id", msg.ID).Err(err).Msg("Failed to convert GatewayPushTask")
			ReleaseGatewayPushTask(task)
			p.Reset()
			continue
		}
		task.StreamID = msg.ID
		tasks = append(tasks, task)
		p.Reset()
	}
	return tasks, nil
}

// GatewayPushMessage publishes inbound messages from clients into the
// inbound stream.
func (r *RedisMQ) GatewayPushMessage(ctx context.Context, messages []*imv1.MessagePush) error {
	return r.pushMessageToStream(ctx, streamInbound, messages)
}

// GatewayAckMessage confirms the given deliver stream IDs.
func (r *RedisMQ) GatewayAckMessage(ctx context.Context, ids ...string) error {
	return r.ack(ctx, streamDeliver, gatewayGroup, ids...)
}

// pullFromStream is the shared read path used by the worker consumer.
func (r *RedisMQ) pullFromStream(ctx context.Context, stream, group, consumer string, batch int64) ([]*StreamMessage, error) {
	result, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    batch,
		Block:    5 * time.Second,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("xread group failed: %w", err)
	}
	if len(result) == 0 {
		return nil, nil
	}

	messages := make([]*StreamMessage, 0, len(result[0].Messages))
	for _, msg := range result[0].Messages {
		data, ok := msg.Values["data"].(string)
		if !ok {
			log.Error().Str("id", msg.ID).Msg("Missing 'data' field in stream message")
			continue
		}
		p := &imv1.MessagePush{}
		if err := p.UnmarshalVT(unsafe.Slice(unsafe.StringData(data), len(data))); err != nil {
			log.Error().Str("id", msg.ID).Err(err).Msg("Failed to unmarshal MessagePush")
			continue
		}
		messages = append(messages, &StreamMessage{ID: msg.ID, Data: p})
	}
	return messages, nil
}

// pushMessageToStream pipelines an XAdd for each message into the given
// stream.
func (r *RedisMQ) pushMessageToStream(ctx context.Context, stream string, messages []*imv1.MessagePush) error {
	if len(messages) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, msg := range messages {
		if msg == nil {
			log.Error().Msg("Skipping nil message pointer")
			continue
		}
		bin, err := msg.MarshalVT()
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal MessagePush for stream")
			continue
		}
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			MaxLen: streamMaxLen,
			Approx: true,
			Values: map[string]any{"data": unsafe.String(unsafe.SliceData(bin), len(bin))},
		})
	}
	_, err := pipe.Exec(ctx)
	return err
}

// ack acknowledges the given stream IDs in the named group.
func (r *RedisMQ) ack(ctx context.Context, stream, group string, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	return r.client.XAck(ctx, stream, group, ids...).Err()
}

// Compile-time check that *RedisMQ satisfies the MQ contract.
var _ MQ = (*RedisMQ)(nil)