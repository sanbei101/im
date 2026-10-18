package store

import (
	"bytes"
	"context"
	"time"
	"uuid"
)

func encodeRoom(room Room) []byte {
	return appendRoom(nil, room)
}

// appendRoom encodes into dst so batch writes can reuse one buffer.
func appendRoom(data []byte, room Room) []byte {
	var number [8]byte
	data = putUUID(data, room.RoomID)
	data = appendString(data, room.ChatType)
	data = appendString(data, room.Name)
	data = appendString(data, room.AvatarURL)
	data = appendString(data, room.Notice)
	data = appendBytes(data, room.SingleChatHash)
	putU64(number[:], room.LastSeq)
	data = append(data, number[:]...)
	putI64(number[:], room.CreatedAt.UnixMicro())
	data = append(data, number[:]...)
	putI64(number[:], room.UpdatedAt.UnixMicro())
	data = append(data, number[:]...)
	if room.LastMsg != nil {
		data = append(data, 1)
		data = appendBytes(data, encodeMessage(*room.LastMsg))
	} else {
		data = append(data, 0)
	}
	return data
}

func decodeRoom(data []byte) (Room, error) {
	d := decoder{data: data}
	room := Room{
		RoomID:         d.uuid(),
		ChatType:       internEnum(d.bytes()),
		Name:           d.string(),
		AvatarURL:      d.string(),
		Notice:         d.string(),
		SingleChatHash: bytes.Clone(d.bytes()),
		LastSeq:        d.u64(),
		CreatedAt:      time.UnixMicro(d.i64()),
		UpdatedAt:      time.UnixMicro(d.i64()),
	}
	if d.byte() == 1 {
		msg, err := decodeMessage(d.bytes())
		if err != nil {
			return Room{}, err
		}
		// The room outlives the Get closer inside the rooms LRU: its embedded
		// last message must not alias the pebble buffer.
		msg = ownedMessage(msg)
		room.LastMsg = &msg
	}
	if err := d.check("room"); err != nil {
		return Room{}, err
	}
	return room, nil
}

// decodeRoomHead parses only the leading fields of a room record, without
// materializing strings or decoding the embedded last message. It serves
// scans that need just the room id and LastSeq, such as the archiver.
func decodeRoomHead(data []byte) (Room, error) {
	// 字段读取顺序即编码顺序,不可调换。
	d := decoder{data: data}
	id := d.uuid()
	// chatType, name, avatarURL, notice, singleChatHash: length-prefixed,
	// skipped without copying.
	for range 5 {
		d.skip()
	}
	seq := d.u64()
	d.i64() // createdAt
	d.i64() // updatedAt
	if d.byte() == 1 {
		d.skip() // embedded last message record
	}
	if err := d.check("room"); err != nil {
		return Room{}, err
	}
	return Room{RoomID: id, LastSeq: seq}, nil
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
	s.getRoomShard(roomID).rooms.Set(roomID, room)
	return room, nil
}
