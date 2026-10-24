package engine

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

// Fixed key lengths: one prefix byte plus fixed-width fields. Stack buffers
// sized by these constants keep key construction allocation-free without
// spreading layout arithmetic across call sites. Reaction keys carry a
// variable-length trailing emoji and have no constant.
const (
	uuidLen = 16
	seqLen  = 8

	roomKeyLen            = 1 + uuidLen
	messagePrefixLen      = 1 + uuidLen
	messageKeyLen         = 1 + uuidLen + seqLen
	dedupKeyLen           = 1 + uuidLen*3
	msgIDIndexKeyLen      = 1 + uuidLen*2
	archiveIndexKeyLen    = 1 + uuidLen + seqLen
	archiveIndexPrefixLen = 1 + uuidLen
	reactionPrefixLen     = 1 + uuidLen*2
	reactionRoomPrefixLen = 1 + uuidLen
	pinKeyLen             = 1 + uuidLen*2
	pinPrefixLen          = 1 + uuidLen
	readKeyLen            = 1 + uuidLen*2
	readPrefixLen         = 1 + uuidLen
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
