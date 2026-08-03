package mq

import (
	"sync"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
)

// Envelope wraps a broker-assigned stream ID with a decoded payload.
// T is the protobuf message type carried on the wire. StreamID is opaque
// to callers and must be passed back to the same MQ implementation via
// the corresponding Ack method.
type Envelope[T any] struct {
	StreamID string
	Payload  T
}

// resettable is the optional contract for T that lets Envelope.Reset() release
// any internal allocations held by the payload. All protobuf-go-lite
// generated messages in this package satisfy it.
type resettable interface{ Reset() }

// 1. 上行消息信封 (Client -> Gateway -> MQ -> Worker)
// 代表客户端发送、网关接收并投递给 Worker 的上行消息.
type InboundMsgEnvelope = Envelope[*imv1.Message]

// 2. 下行投递任务信封 (Worker -> MQ -> Gateway -> Client)
// 代表 Worker 计算完成后,指令网关进行批量下行分发的投递任务.
type DeliverTaskEnvelope = Envelope[*imv1.GatewayDeliveryTask]

func (t *Envelope[T]) Reset() {
	t.StreamID = ""
	// 任何实现了 Reset() 的 payload 都会在这里被回收底层数组 (protobuf-go-lite 生成的消息全部满足).
	// 注意: 如果 T 是指针类型且 Payload 为 typed nil,调用 r.Reset() 会 panic —— 调用方需保证
	// Payload 非 nil (本包的 pool 永远预分配非空 proto,正常路径下不会触发).
	if r, ok := any(t.Payload).(resettable); ok {
		r.Reset()
	}
}

// deliveryTaskPool reuses envelopes to avoid per-batch allocations of the
// embedded GatewayDeliveryTask proto (which holds the full Message body).
var deliveryTaskPool = sync.Pool{
	New: func() any {
		return &DeliverTaskEnvelope{
			Payload: &imv1.GatewayDeliveryTask{},
		}
	},
}

// AcquireDeliveryTask pulls an envelope from the pool. Callers must release
// it via ReleaseDeliveryTask once they are done.
func AcquireDeliveryTask() *DeliverTaskEnvelope {
	return deliveryTaskPool.Get().(*DeliverTaskEnvelope)
}

// ReleaseDeliveryTask returns the envelope to the pool after clearing its
// fields.
func ReleaseDeliveryTask(t *DeliverTaskEnvelope) {
	t.Reset()
	deliveryTaskPool.Put(t)
}