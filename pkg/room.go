package pkg

import (
	"crypto/sha256"
	"uuid"
)

// SingleChatHash is the stable lookup key for a one-to-one room. Sorting the
// user ids makes both directions produce the same hash.
func SingleChatHash(a, b uuid.UUID) [32]byte {
	if a.Compare(b) > 0 {
		a, b = b, a
	}
	var data [32]byte
	copy(data[:16], a[:])
	copy(data[16:], b[:])
	return sha256.Sum256(data[:])
}

// SingleRoomID derives the room id from the same unordered pair. Because the
// id is deterministic, either API node can compute the owner before creating
// the room.
func SingleRoomID(a, b uuid.UUID) uuid.UUID {
	hash := SingleChatHash(a, b)
	var id uuid.UUID
	copy(id[:], hash[:16])
	id[6] = (id[6] & 0x0f) | 0x80 // version 8 (custom, SHA-256 based)
	id[8] = (id[8] & 0x3f) | 0x80 // RFC 4122 variant
	return id
}
