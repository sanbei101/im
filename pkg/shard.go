package pkg

import (
	"encoding/binary"
	"errors"
	"uuid"
)

func RoomSlot(roomID uuid.UUID, slots int) (uint16, error) {
	if slots < 1 || slots > 1<<16 {
		return 0, errors.New("invalid shard slot count")
	}
	hash := roomID
	value := binary.BigEndian.Uint32(
		hash[:4],
	) ^ binary.BigEndian.Uint32(
		hash[4:8],
	) ^ binary.BigEndian.Uint32(
		hash[8:12],
	) ^ binary.BigEndian.Uint32(
		hash[12:],
	)
	return uint16(value % uint32(slots)), nil
}

func NodeIndex(slot uint16, slots, nodes int) (int, error) {
	if slots < 1 || nodes < 1 || int(slot) >= slots {
		return 0, errors.New("invalid shard topology")
	}
	return int(uint32(slot) * uint32(nodes) / uint32(slots)), nil
}
