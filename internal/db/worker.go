package db

import (
	"encoding/binary"
	"errors"
	"sync"
	"unsafe"

	"github.com/google/uuid"
)

type GatewayPushTask struct {
	StreamID       string
	RoomID         uuid.UUID
	TargetUserIDs  []uuid.UUID
	Message        Message
}

var GatewayPushTaskPool = sync.Pool{
	New: func() any {
		return &GatewayPushTask{
			TargetUserIDs: make([]uuid.UUID, 0, 16),
		}
	},
}

func AcquireGatewayPushTask() *GatewayPushTask {
	return GatewayPushTaskPool.Get().(*GatewayPushTask)
}

func ReleaseGatewayPushTask(t *GatewayPushTask) {
	t.Reset()
	GatewayPushTaskPool.Put(t)
}

func (t *GatewayPushTask) Reset() {
	t.RoomID = uuid.UUID{}
	if t.TargetUserIDs != nil {
		t.TargetUserIDs = t.TargetUserIDs[:0]
	}
	t.Message.Reset()
}

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
