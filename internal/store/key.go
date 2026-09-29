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
func messageKey(room uuid.UUID, seq uint64) string {
	var suffix [8]byte
	binary.BigEndian.PutUint64(suffix[:], seq)
	return "message/" + room.String() + "/" + string(suffix[:])
}
func messagePrefix(room uuid.UUID) string { return "message/" + room.String() + "/" }
func dedupKey(room, sender, client uuid.UUID) string {
	return "dedup/" + room.String() + "/" + sender.String() + "/" + client.String()
}
