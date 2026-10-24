package store

import (
	"encoding/binary"
	"uuid"
)

// Single-byte prefixes for the remaining Pebble keys.
const (
	prefixRoom       = 'r'
	prefixMessage    = 'm'
	prefixDedup      = 'd'
	prefixArchive    = 'A'
	prefixMsgIDIndex = 'i'
	prefixReaction   = 'R'
	prefixPin        = 'P'
	prefixRead       = 'u'
)

func appendRoomKey(dst []byte, id uuid.UUID) []byte {
	return append(append(dst, prefixRoom), id[:]...)
}

func appendRoomPrefix(dst []byte) []byte {
	return append(dst, prefixRoom)
}

func appendMessagePrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixMessage), room[:]...)
}

func appendMessageKey(dst []byte, room uuid.UUID, seq uint64) []byte {
	dst = append(append(dst, prefixMessage), room[:]...)
	var suffix [8]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return append(dst, suffix[:]...)
}

func appendDedupKey(dst []byte, room, sender, client uuid.UUID) []byte {
	return append(append(append(append(dst, prefixDedup), room[:]...), sender[:]...), client[:]...)
}

func appendArchiveIndexKey(dst []byte, room uuid.UUID, startSeq uint64) []byte {
	dst = append(append(dst, prefixArchive), room[:]...)
	var suffix [8]byte
	binary.BigEndian.PutUint64(suffix[:], startSeq)
	return append(dst, suffix[:]...)
}

func appendArchiveIndexPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixArchive), room[:]...)
}

func appendMsgIDIndexKey(dst []byte, room, msgID uuid.UUID) []byte {
	return append(append(append(dst, prefixMsgIDIndex), room[:]...), msgID[:]...)
}

func appendReactionKey(dst []byte, room, msgID, user uuid.UUID, emoji string) []byte {
	return append(append(append(append(append(dst, prefixReaction), room[:]...), msgID[:]...), user[:]...), emoji...)
}

func appendReactionPrefix(dst []byte, room, msgID uuid.UUID) []byte {
	return append(append(append(dst, prefixReaction), room[:]...), msgID[:]...)
}

func appendReactionRoomPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixReaction), room[:]...)
}

func appendPinKey(dst []byte, room, msgID uuid.UUID) []byte {
	return append(append(append(dst, prefixPin), room[:]...), msgID[:]...)
}

func appendPinPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixPin), room[:]...)
}

func appendReadKey(dst []byte, room, user uuid.UUID) []byte {
	return append(append(append(dst, prefixRead), room[:]...), user[:]...)
}

func appendReadPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixRead), room[:]...)
}

func prefixUpperBoundBuf(buf, prefix []byte) []byte {
	buf = append(buf[:0], prefix...)
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] < 0xff {
			buf[i]++
			return buf[:i+1]
		}
	}
	return append(buf, 0)
}

func prefixUpperBound(prefix []byte) []byte {
	return prefixUpperBoundBuf(nil, prefix)
}
