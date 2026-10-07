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

// StringShard hashes a string key into [0, shards), used to shard in-memory
// maps (gateway pending requests, session tables).
func StringShard(key string, shards int) int {
	var h uint32
	for i := 0; i < len(key); i++ {
		h = h*31 + uint32(key[i])
	}
	return int(h % uint32(shards))
}
