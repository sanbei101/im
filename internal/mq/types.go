package mq

import (
	"sync"

	"github.com/google/uuid"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
)

// StreamMessage pairs a broker-assigned stream ID with the decoded message
// payload. The ID is opaque to callers and must be passed back to the same
// MQ implementation via the corresponding Ack method.
type StreamMessage struct {
	ID   string
	Data *imv1.MessagePush
}

// GatewayPushTask is the worker's instruction to the gateway: deliver
// `Message` to every user in `TargetUserIDs`. StreamID is set by the broker
// on read and only the streaming protocol needs to interpret it.
//
// Wire format: the proto representation (imv1.GatewayPushTask) is used
// end-to-end via vtproto Marshal/Unmarshal — there is no separate
// implementation-private encoding.
type GatewayPushTask struct {
	StreamID      string
	RoomID        uuid.UUID
	TargetUserIDs []uuid.UUID
	Message       *imv1.MessagePush
}

var gatewayPushTaskPool = sync.Pool{
	New: func() any {
		return &GatewayPushTask{
			TargetUserIDs: make([]uuid.UUID, 0, 16),
		}
	},
}

// AcquireGatewayPushTask pulls a zeroed task from the pool. Callers must
// release it via ReleaseGatewayPushTask once they are done.
func AcquireGatewayPushTask() *GatewayPushTask {
	return gatewayPushTaskPool.Get().(*GatewayPushTask)
}

// ReleaseGatewayPushTask returns the task to the pool after clearing its
// fields.
func ReleaseGatewayPushTask(t *GatewayPushTask) {
	t.Reset()
	gatewayPushTaskPool.Put(t)
}

// Reset clears the task so it can be reused. The TargetUserIDs slice keeps
// its backing array; the embedded MessagePush is reset via Reset().
func (t *GatewayPushTask) Reset() {
	t.RoomID = uuid.UUID{}
	if t.TargetUserIDs != nil {
		t.TargetUserIDs = t.TargetUserIDs[:0]
	}
	if t.Message != nil {
		t.Message.Reset()
		t.Message = nil
	}
}

// Proto converts the task to its wire representation.
func (t *GatewayPushTask) Proto() *imv1.GatewayPushTask {
	roomID := t.RoomID.String()
	p := &imv1.GatewayPushTask{
		RoomId:  &roomID,
		Message: t.Message,
	}
	if len(t.TargetUserIDs) > 0 {
		ids := make([]string, len(t.TargetUserIDs))
		for i, u := range t.TargetUserIDs {
			ids[i] = u.String()
		}
		p.TargetUserIds = ids
	}
	return p
}

// FromProto populates the task from its wire representation. The proto
// message is consumed; caller may safely reset/release it afterwards.
func (t *GatewayPushTask) FromProto(p *imv1.GatewayPushTask) error {
	if p == nil {
		return nil
	}
	if p.GetRoomId() != "" {
		roomID, err := uuid.Parse(p.GetRoomId())
		if err != nil {
			return err
		}
		t.RoomID = roomID
	}
	if ids := p.GetTargetUserIds(); len(ids) > 0 {
		if cap(t.TargetUserIDs) >= len(ids) {
			t.TargetUserIDs = t.TargetUserIDs[:0]
		} else {
			t.TargetUserIDs = make([]uuid.UUID, 0, len(ids))
		}
		for _, s := range ids {
			u, err := uuid.Parse(s)
			if err != nil {
				return err
			}
			t.TargetUserIDs = append(t.TargetUserIDs, u)
		}
	}
	if p.GetMessage() != nil {
		t.Message = p.GetMessage()
	}
	return nil
}