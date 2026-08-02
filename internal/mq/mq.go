// Package mq defines the message queue abstraction used by gateway and worker
// processes. The concrete transport (Redis Streams today, possibly Kafka / NATS
// in the future) lives behind this interface, so business code never depends
// on a specific broker.
package mq

import (
	"context"

	"github.com/sanbei101/im/internal/db"
)

// MQ is the message queue boundary between gateway and worker.
//
// Messages flow in two directions:
//   - client -> gateway -> MQ -> worker -> MQ -> gateway -> client
//
// The interface is intentionally transport-agnostic. Stream IDs are opaque
// tokens returned by the broker and are only meaningful to the same
// implementation that produced them.
type MQ interface {
	// InitStreamGroups prepares any broker-side resources (consumer groups,
	// topics, etc.). It must be idempotent.
	InitStreamGroups(ctx context.Context) error

	// WorkerPullMessage reads up to `batch` inbound messages produced by
	// gateway clients. Worker is the consumer.
	WorkerPullMessage(ctx context.Context, batch int64) ([]*StreamMessage, error)

	// WorkerPushGatewayTask publishes deliver tasks that the gateway will
	// fan out to online clients. Worker is the producer.
	WorkerPushGatewayTask(ctx context.Context, tasks []*GatewayPushTask) error

	// WorkerAckMessage confirms that the worker has successfully processed
	// the given inbound stream IDs.
	WorkerAckMessage(ctx context.Context, ids ...string) error

	// GatewayPullTask reads up to `batch` deliver tasks produced by the
	// worker. Gateway is the consumer.
	GatewayPullTask(ctx context.Context, batch int64) ([]*GatewayPushTask, error)

	// GatewayPushMessage publishes inbound messages from clients. Gateway is
	// the producer.
	GatewayPushMessage(ctx context.Context, messages []*db.Message) error

	// GatewayAckMessage confirms that the gateway has successfully delivered
	// the given deliver stream IDs.
	GatewayAckMessage(ctx context.Context, ids ...string) error
}
