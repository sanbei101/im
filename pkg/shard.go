package pkg

import (
	"encoding/binary"
	"errors"
	"uuid"
)

// MaxSlots is the upper bound of shard slots: slot and node indexes stay in
// uint16 and the static topology remains enumerable.
const MaxSlots = 1 << 16

func RoomSlot(roomID uuid.UUID, slots int) (uint16, error) {
	if slots < 1 || slots > MaxSlots {
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

// UUIDShard hashes a UUID key into [0, shards), the uuid counterpart of
// StringShard. Used to elect exactly one API node per user for notification
// fan-out, so every node must agree on the result.
func UUIDShard(id uuid.UUID, shards int) int {
	fold := binary.BigEndian.Uint32(id[:4]) ^
		binary.BigEndian.Uint32(id[4:8]) ^
		binary.BigEndian.Uint32(id[8:12]) ^
		binary.BigEndian.Uint32(id[12:])
	return int(fold % uint32(shards))
}
