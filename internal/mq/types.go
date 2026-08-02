package mq

import (
	"encoding/binary"
	"errors"
	"sync"
	"unsafe"

	"github.com/google/uuid"

	"github.com/sanbei101/im/internal/db"
)

// StreamMessage pairs a broker-assigned stream ID with the decoded message
// payload. The ID is opaque to callers and must be passed back to the same
// MQ implementation via the corresponding Ack method.
type StreamMessage struct {
	ID   string
	Data *db.Message
}

// GatewayPushTask is the worker's instruction to the gateway: deliver
// `Message` to every user in `TargetUserIDs`. StreamID is set by the broker
// on read and only the streaming protocol needs to interpret it.
type GatewayPushTask struct {
	StreamID      string
	RoomID        uuid.UUID
	TargetUserIDs []uuid.UUID
	Message       db.Message
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
// its backing array.
func (t *GatewayPushTask) Reset() {
	t.RoomID = uuid.UUID{}
	if t.TargetUserIDs != nil {
		t.TargetUserIDs = t.TargetUserIDs[:0]
	}
	t.Message.Reset()
}

// Marshal serializes the task to a length-prefixed binary blob. The encoding
// is MQ-implementation-private; it is not part of the MQ interface contract.
func (t *GatewayPushTask) Marshal() ([]byte, error) {
	msgSize := t.Message.Size()
	totalSize := 16 + 4 + (len(t.TargetUserIDs) * 16) + msgSize

	buf := make([]byte, totalSize)
	offset := 0

	copy(buf[offset:], t.RoomID[:])
	offset += 16

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(t.TargetUserIDs)))
	offset += 4

	if len(t.TargetUserIDs) > 0 {
		byteLen := len(t.TargetUserIDs) * 16
		src := unsafe.Slice((*byte)(unsafe.Pointer(&t.TargetUserIDs[0])), byteLen)
		copy(buf[offset:], src)
		offset += byteLen
	}

	t.Message.MarshalTo(buf[offset:])

	return buf, nil
}

// Unmarshal populates the task from a length-prefixed binary blob produced
// by Marshal.
func (t *GatewayPushTask) Unmarshal(data []byte) error {
	if len(data) < 20 {
		return errors.New("data too short for GatewayPushTask")
	}
	offset := 0

	copy(t.RoomID[:], data[offset:offset+16])
	offset += 16

	targetLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	offset += 4

	if targetLen > 0 {
		byteLen := targetLen * 16
		if len(data) < offset+byteLen {
			return errors.New("data too short for TargetUserIDs")
		}

		if cap(t.TargetUserIDs) >= targetLen {
			t.TargetUserIDs = t.TargetUserIDs[:targetLen]
		} else {
			t.TargetUserIDs = make([]uuid.UUID, targetLen)
		}
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&t.TargetUserIDs[0])), byteLen)
		copy(dst, data[offset:offset+byteLen])
		offset += byteLen
	} else {
		t.TargetUserIDs = t.TargetUserIDs[:0]
	}

	if offset < len(data) {
		return t.Message.Unmarshal(data[offset:])
	}
	return errors.New("data too short, missing Message")
}
