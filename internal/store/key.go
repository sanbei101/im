package store

import (
	"encoding/binary"
	"fmt"
	"strings"
	"uuid"
)

func userKey(id uuid.UUID) string    { return "user/" + id.String() }
func usernameKey(name string) string { return "username/" + strings.ToLower(strings.TrimSpace(name)) }
func roomKey(id uuid.UUID) string    { return "room/" + id.String() }
func roomMemberKey(room, user uuid.UUID) string {
	return "member/" + room.String() + "/" + user.String()
}
func roomMemberPrefix(room uuid.UUID) string { return "member/" + room.String() + "/" }
func userRoomKey(user, room uuid.UUID) string {
	return "user_room/" + user.String() + "/" + room.String()
}
func userRoomPrefix(user uuid.UUID) string { return "user_room/" + user.String() + "/" }
func singleRoomKey(hash []byte) string     { return fmt.Sprintf("single/%x", hash) }

func appendMessagePrefix(dst []byte, room uuid.UUID) []byte {
	dst = append(dst, 'm')
	return append(dst, room[:]...)
}

func appendMessageKey(dst []byte, room uuid.UUID, seq uint64) []byte {
	var suffix [8]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return append(appendMessagePrefix(dst, room), suffix[:]...)
}

func appendDedupKey(dst []byte, room, sender, client uuid.UUID) []byte {
	dst = append(dst, 'd')
	dst = append(dst, room[:]...)
	dst = append(dst, sender[:]...)
	return append(dst, client[:]...)
}
