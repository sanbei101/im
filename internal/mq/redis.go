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
func (r *RedisMQ) WorkerPullMessage(ctx context.Context, batch int64) ([]*InboundMsgEnvelope, error) {
	return r.pullFromStream(ctx, streamInbound, workerGroup, WorkerName, batch)
}

// WorkerEnqueueDeliveryTask pipelines each task into the deliver stream with an
// approximate max length cap.
func (r *RedisMQ) WorkerEnqueueDeliveryTask(ctx context.Context, tasks []*DeliverTaskEnvelope) error {
	if len(tasks) == 0 {
		return nil
	}

	pipe := r.client.Pipeline()
	for _, task := range tasks {
		if task == nil || task.Payload == nil {
			log.Error().Msg("Skipping nil delivery task or payload")
			continue
		}
		bin, err := task.Payload.MarshalVT()
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal GatewayDeliveryTask")
			continue
		}
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

// GatewayPullDeliveryTask reads up to `batch` delivery tasks for the gateway.
func (r *RedisMQ) GatewayPullDeliveryTask(ctx context.Context, batch int64) ([]*DeliverTaskEnvelope, error) {
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

	tasks := make([]*DeliverTaskEnvelope, 0, len(result[0].Messages))
	for _, msg := range result[0].Messages {
		data, ok := msg.Values["data"].(string)
		if !ok {
			log.Error().Str("id", msg.ID).Msg("Missing 'data' field in stream message")
			continue
		}
		task := AcquireDeliveryTask()
		if err := task.Payload.UnmarshalVT(unsafe.Slice(unsafe.StringData(data), len(data))); err != nil {
			log.Error().Str("id", msg.ID).Err(err).Msg("Failed to unmarshal GatewayDeliveryTask")
			ReleaseDeliveryTask(task)
			continue
		}
		task.StreamID = msg.ID
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// GatewayEnqueueMessage publishes inbound messages from clients into the
// inbound stream.
func (r *RedisMQ) GatewayEnqueueMessage(ctx context.Context, messages []*imv1.Message) error {
	return r.pushMessageToStream(ctx, streamInbound, messages)
}

// GatewayAckMessage confirms the given deliver stream IDs.
func (r *RedisMQ) GatewayAckMessage(ctx context.Context, ids ...string) error {
	return r.ack(ctx, streamDeliver, gatewayGroup, ids...)
}

// pullFromStream is the shared read path used by the worker consumer.
func (r *RedisMQ) pullFromStream(
	ctx context.Context,
	stream, group, consumer string,
	batch int64,
) ([]*InboundMsgEnvelope, error) {
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

	messages := make([]*InboundMsgEnvelope, 0, len(result[0].Messages))
	for _, msg := range result[0].Messages {
		data, ok := msg.Values["data"].(string)
		if !ok {
			log.Error().Str("id", msg.ID).Msg("Missing 'data' field in stream message")
			continue
		}
		p := &imv1.Message{}
		if err := p.UnmarshalVT(unsafe.Slice(unsafe.StringData(data), len(data))); err != nil {
			log.Error().Str("id", msg.ID).Err(err).Msg("Failed to unmarshal Message")
			continue
		}
		messages = append(messages, &InboundMsgEnvelope{StreamID: msg.ID, Payload: p})
	}
	return messages, nil
}

// pushMessageToStream pipelines an XAdd for each message into the given
// stream.
func (r *RedisMQ) pushMessageToStream(ctx context.Context, stream string, messages []*imv1.Message) error {
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
			log.Error().Err(err).Msg("Failed to marshal Message for stream")
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
