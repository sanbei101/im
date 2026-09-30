package store

import (
	"encoding/binary"
	"strings"
	"uuid"
)

// Single-byte prefix definitions for unified, zero-alloc binary keys:
// 'u' - User
// 'n' - Username index
// 'r' - Room
// 'e' - Room Member ('e' for m-e-mber)
// 'x' - User Room index ('x' for user-room-inde-x)
// 's' - Single Chat hash index
// 'm' - Message (ordered by seq)
// 'd' - Message Dedup

func appendUserKey(dst []byte, id uuid.UUID) []byte {
	return append(append(dst, 'u'), id[:]...)
}

func appendUsernameKey(dst []byte, name string) []byte {
	return append(append(dst, 'n'), strings.ToLower(strings.TrimSpace(name))...)
}

func appendRoomKey(dst []byte, id uuid.UUID) []byte {
	return append(append(dst, 'r'), id[:]...)
}

func appendMemberKey(dst []byte, room, user uuid.UUID) []byte {
	return append(append(append(dst, 'e'), room[:]...), user[:]...)
}

func appendMemberPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, 'e'), room[:]...)
}

func appendUserRoomKey(dst []byte, user, room uuid.UUID) []byte {
	return append(append(append(dst, 'x'), user[:]...), room[:]...)
}

func appendUserRoomPrefix(dst []byte, user uuid.UUID) []byte {
	return append(append(dst, 'x'), user[:]...)
}

func appendSingleRoomKey(dst, hash []byte) []byte {
	return append(append(dst, 's'), hash...)
}

func appendMessagePrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, 'm'), room[:]...)
}

func appendMessageKey(dst []byte, room uuid.UUID, seq uint64) []byte {
	dst = append(append(dst, 'm'), room[:]...)
	var suffix [8]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return append(dst, suffix[:]...)
}

func appendDedupKey(dst []byte, room, sender, client uuid.UUID) []byte {
	return append(append(append(append(dst, 'd'), room[:]...), sender[:]...), client[:]...)
}

func prefixUpperBound(prefix []byte) []byte {
	upperBound := append([]byte(nil), prefix...)
	for i := len(upperBound) - 1; i >= 0; i-- {
		if upperBound[i] < 0xff {
			upperBound[i]++
			return upperBound[:i+1]
		}
	}
	return nil
}
