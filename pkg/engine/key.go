package engine

import (
	"encoding/binary"
	"time"
	"uuid"
)

// Single-byte prefixes for the Pebble keys.
const (
	prefixRoom     = 'r'
	prefixMessage  = 'm'
	prefixDedup    = 'd'
	prefixArchive  = 'A'
	prefixReaction = 'R'
	prefixPin      = 'P'
	prefixRead     = 'u'
)

// Fixed key lengths: one prefix byte plus fixed-width fields. Stack buffers
// sized by these constants keep key construction allocation-free without
// spreading layout arithmetic across call sites. Reaction keys carry a
// variable-length trailing emoji and have no constant.
const (
	uuidLen = 16
	seqLen  = 8
	dayLen  = 2

	roomKeyLen            = 1 + uuidLen
	messagePrefixLen      = 1 + uuidLen
	messageKeyLen         = 1 + uuidLen + seqLen
	dedupKeyLen           = 1 + dayLen + uuidLen*3
	dedupDayPrefixLen     = 1 + dayLen
	archiveIndexKeyLen    = 1 + uuidLen + seqLen
	archiveIndexPrefixLen = 1 + uuidLen
	reactionPrefixLen     = 1 + uuidLen + seqLen
	reactionRoomPrefixLen = 1 + uuidLen
	pinKeyLen             = 1 + uuidLen + seqLen
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
	var suffix [seqLen]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return append(dst, suffix[:]...)
}

// dedupDay buckets a dedup key by its write day: idempotency covers network
// retries, not eternity, so expired buckets are range-deleted whole.
func dedupDay(now time.Time) uint16 {
	return uint16(now.Unix() / 86400)
}

// dedupProbeDays returns the buckets a retry can live in: always the write
// day, plus the client id's own day for UUIDv7 senders - which makes the
// common same-day retry a single point read - or the previous day for
// opaque ids, covering midnight crossings. The slice is stack-shaped; no
// allocation lands on the write path.
func dedupProbeDays(client uuid.UUID, today uint16) (days [2]uint16, count int) {
	days[0] = today
	// RFC 9562: the version is the high nibble of byte 6; a v7 id's first
	// 48 bits are its unix-millisecond creation time.
	if client[6]>>4 == 7 {
		ms := int64(binary.BigEndian.Uint64(client[:8]) >> 16)
		if day := uint16(ms / 86400000); day != today {
			days[1], count = day, 2
			return days, count
		}
		return days, 1
	}
	days[1], count = today-1, 2
	return days, count
}

func appendDedupKey(dst []byte, day uint16, room, sender, client uuid.UUID) []byte {
	dst = append(dst, prefixDedup, byte(day>>8), byte(day))
	dst = append(dst, room[:]...)
	dst = append(dst, sender[:]...)
	return append(dst, client[:]...)
}

func appendDedupDayPrefix(dst []byte, day uint16) []byte {
	return append(append(dst, prefixDedup), byte(day>>8), byte(day))
}

func appendArchiveIndexKey(dst []byte, room uuid.UUID, startSeq uint64) []byte {
	dst = append(append(dst, prefixArchive), room[:]...)
	var suffix [seqLen]byte
	binary.BigEndian.PutUint64(suffix[:], startSeq)
	return append(dst, suffix[:]...)
}

func appendArchiveIndexPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixArchive), room[:]...)
}

func appendReactionKey(dst []byte, room uuid.UUID, seq uint64, user uuid.UUID, emoji string) []byte {
	dst = append(append(dst, prefixReaction), room[:]...)
	var suffix [seqLen]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	dst = append(dst, suffix[:]...)
	dst = append(dst, user[:]...)
	return append(dst, emoji...)
}

func appendReactionPrefix(dst []byte, room uuid.UUID, seq uint64) []byte {
	dst = append(append(dst, prefixReaction), room[:]...)
	var suffix [seqLen]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return append(dst, suffix[:]...)
}

func appendReactionRoomPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, prefixReaction), room[:]...)
}

func appendPinKey(dst []byte, room uuid.UUID, seq uint64) []byte {
	dst = append(append(dst, prefixPin), room[:]...)
	var suffix [seqLen]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return append(dst, suffix[:]...)
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
