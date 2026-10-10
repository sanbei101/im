package store

import (
	"encoding/binary"
	"strings"
	"uuid"
)

// Single-byte prefix definitions for unified, zero-alloc binary keys:
// 'u' - User
// 'n' - Username index
// 'N' - Nickname index (lower nickname + user id, value is the user id again)
// 'r' - Room
// 'e' - Room Member ('e' for m-e-mber)
// 'x' - User Room index, the value is the member record
// 's' - Single Chat hash index
// 'm' - Message
// 'd' - Message Dedup
// 'A' - Message archive manifest

func appendUserKey(dst []byte, id uuid.UUID) []byte {
	return append(append(dst, 'u'), id[:]...)
}

func appendUserPrefix(dst []byte) []byte {
	return append(dst, 'u')
}

func appendUsernameKey(dst []byte, name string) []byte {
	return append(append(dst, 'n'), strings.ToLower(strings.TrimSpace(name))...)
}

func appendUsernamePrefix(dst []byte) []byte {
	return append(dst, 'n')
}

// 'N' mirrors the username index for nicknames: key is
// 'N' + lower(nickname) + user id (nicknames are not unique, so the id is
// part of the key), value is the user id again.
func appendNicknameKey(dst []byte, lower string, id uuid.UUID) []byte {
	return append(append(append(dst, 'N'), lower...), id[:]...)
}

func appendNicknamePrefix(dst []byte) []byte {
	return append(dst, 'N')
}

func appendRoomKey(dst []byte, id uuid.UUID) []byte {
	return append(append(dst, 'r'), id[:]...)
}

func appendRoomPrefix(dst []byte) []byte {
	return append(dst, 'r')
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

func appendArchiveIndexKey(dst []byte, room uuid.UUID, startSeq uint64) []byte {
	dst = append(append(dst, 'A'), room[:]...)
	var suffix [8]byte
	binary.BigEndian.PutUint64(suffix[:], startSeq)
	return append(dst, suffix[:]...)
}

func appendArchiveIndexPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, 'A'), room[:]...)
}

func appendFriendKey(dst []byte, user, friend uuid.UUID) []byte {
	return append(append(append(dst, 'f'), user[:]...), friend[:]...)
}

func appendFriendPrefix(dst []byte, user uuid.UUID) []byte {
	return append(append(dst, 'f'), user[:]...)
}

func appendApplicationKey(dst []byte, toUser, fromUser uuid.UUID) []byte {
	return append(append(append(dst, 'a'), toUser[:]...), fromUser[:]...)
}

func appendApplicationPrefix(dst []byte, toUser uuid.UUID) []byte {
	return append(append(dst, 'a'), toUser[:]...)
}

func appendBlacklistKey(dst []byte, user, target uuid.UUID) []byte {
	return append(append(append(dst, 'b'), user[:]...), target[:]...)
}

func appendBlacklistPrefix(dst []byte, user uuid.UUID) []byte {
	return append(append(dst, 'b'), user[:]...)
}

func appendReadSeqKey(dst []byte, user, room uuid.UUID) []byte {
	return append(append(append(dst, 'q'), user[:]...), room[:]...)
}

func appendReadSeqPrefix(dst []byte, user uuid.UUID) []byte {
	return append(append(dst, 'q'), user[:]...)
}

// 'Q' mirrors the read-seq marker room-major ('q' is user-major): ReadUsers
// scans one room's markers in a single range instead of a get per member.
func appendRoomReadSeqKey(dst []byte, room, user uuid.UUID) []byte {
	return append(append(append(dst, 'Q'), room[:]...), user[:]...)
}

func appendRoomReadSeqPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, 'Q'), room[:]...)
}

func appendMsgIDIndexKey(dst []byte, room, msgID uuid.UUID) []byte {
	return append(append(append(dst, 'i'), room[:]...), msgID[:]...)
}

func appendReactionKey(dst []byte, room, msgID, user uuid.UUID, emoji string) []byte {
	return append(append(append(append(append(dst, 'R'), room[:]...), msgID[:]...), user[:]...), emoji...)
}

func appendReactionPrefix(dst []byte, room, msgID uuid.UUID) []byte {
	return append(append(append(dst, 'R'), room[:]...), msgID[:]...)
}

func appendReactionRoomPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, 'R'), room[:]...)
}

func appendPinKey(dst []byte, room, msgID uuid.UUID) []byte {
	return append(append(append(dst, 'P'), room[:]...), msgID[:]...)
}

func appendPinPrefix(dst []byte, room uuid.UUID) []byte {
	return append(append(dst, 'P'), room[:]...)
}

func appendDeviceKey(dst []byte, user uuid.UUID) []byte {
	return append(append(dst, 't'), user[:]...)
}

func prefixUpperBoundBuf(buf, prefix []byte) []byte {
	buf = append(buf[:0], prefix...)
	for i := len(buf) - 1; i >= 0; i-- {
		if buf[i] < 0xff {
			buf[i]++
			return buf[:i+1]
		}
	}
	return nil
}

func prefixUpperBound(prefix []byte) []byte {
	return prefixUpperBoundBuf(nil, prefix)
}
