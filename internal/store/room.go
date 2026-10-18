package store

import (
	"context"
	"encoding/binary"
	"errors"
	"uuid"
)

// appendRoomHead encodes the only state kept in Pebble: the room's last
// committed sequence. Room metadata lives in PostgreSQL.
func appendRoomHead(dst []byte, seq uint64) []byte {
	return binary.BigEndian.AppendUint64(dst, seq)
}

func decodeRoom(data []byte) (Room, error) {
	if len(data) != 8 {
		return Room{}, errors.New("invalid room record")
	}
	return Room{LastSeq: binary.BigEndian.Uint64(data)}, nil
}

func (s *Store) Room(ctx context.Context, roomID uuid.UUID) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	if room, ok := s.getRoomShard(roomID).rooms.Get(roomID); ok {
		return room, nil
	}
	var buf [32]byte
	room, err := s.getRecord(appendRoomKey(buf[:0], roomID), decodeRoom)
	if err != nil {
		return Room{}, err
	}
	room.RoomID = roomID
	s.getRoomShard(roomID).rooms.Set(roomID, room)
	return room, nil
}
