package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"uuid"
)

// appendRoomHead encodes the only state kept in Pebble: the room's last
// committed sequence. Room metadata lives in the control plane.
func appendRoomHead(dst []byte, seq uint64) []byte {
	return binary.BigEndian.AppendUint64(dst, seq)
}

func decodeLastSeq(data []byte) (uint64, error) {
	if len(data) != 8 {
		return 0, errors.New("invalid room head record")
	}
	return binary.BigEndian.Uint64(data), nil
}

// LastSeq returns the room's last committed sequence. ErrNotFound means the
// room has never received a message: the head is created lazily by the first
// committed write.
func (e *Engine) LastSeq(ctx context.Context, roomID uuid.UUID) (uint64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	if seq, ok := e.getRoomShard(roomID).rooms.Get(roomID); ok {
		return seq, nil
	}
	var buf [32]byte
	seq, err := e.getRecord(appendRoomKey(buf[:0], roomID), decodeLastSeq)
	if err != nil {
		return 0, err
	}
	e.getRoomShard(roomID).rooms.Set(roomID, seq)
	return seq, nil
}
