package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeRoom(room Room) []byte {
	var number [8]byte
	capacity := 16 + 4*4 + len(room.ChatType) + len(room.Name) + len(room.AvatarURL) +
		len(room.SingleChatHash) + 8*3
	data := make([]byte, 0, capacity)
	data = putUUID(data, room.RoomID)
	data = appendString(data, room.ChatType)
	data = appendString(data, room.Name)
	data = appendString(data, room.AvatarURL)
	data = appendBytes(data, room.SingleChatHash)
	putU64(number[:], room.LastSeq)
	data = append(data, number[:]...)
	putI64(number[:], room.CreatedAt.UnixMicro())
	data = append(data, number[:]...)
	putI64(number[:], room.UpdatedAt.UnixMicro())
	return append(data, number[:]...)
}

func decodeRoom(data []byte) (Room, error) {
	d := decoder{data: data}
	id, err := d.uuid()
	if err != nil {
		return Room{}, err
	}
	chatType, err := d.string()
	if err != nil {
		return Room{}, err
	}
	name, err := d.string()
	if err != nil {
		return Room{}, err
	}
	avatar, err := d.string()
	if err != nil {
		return Room{}, err
	}
	hash, err := d.bytes()
	if err != nil {
		return Room{}, err
	}
	seq, err := d.u64()
	if err != nil {
		return Room{}, err
	}
	created, err := d.i64()
	if err != nil {
		return Room{}, err
	}
	updated, err := d.i64()
	if err != nil || !d.done() {
		return Room{}, errors.New("invalid room record")
	}
	return Room{
		RoomID:         id,
		ChatType:       chatType,
		Name:           name,
		AvatarURL:      avatar,
		SingleChatHash: bytes.Clone(hash),
		LastSeq:        seq,
		CreatedAt:      time.UnixMicro(created),
		UpdatedAt:      time.UnixMicro(updated),
	}, nil
}

func encodeMember(member Member) []byte {
	data := make([]byte, 0, 16*2+4+len(member.Role)+2)
	data = putUUID(data, member.RoomID)
	data = putUUID(data, member.UserID)
	data = appendString(data, member.Role)
	data = append(data, boolByte(member.IsHidden), boolByte(member.IsMuted))
	return data
}

func decodeMember(data []byte) (Member, error) {
	d := decoder{data: data}
	roomID, err := d.uuid()
	if err != nil {
		return Member{}, err
	}
	userID, err := d.uuid()
	if err != nil {
		return Member{}, err
	}
	role, err := d.string()
	if err != nil {
		return Member{}, errors.New("invalid member record")
	}
	member := Member{
		RoomID:   roomID,
		UserID:   userID,
		Role:     role,
		IsHidden: d.byte() != 0,
		IsMuted:  d.byte() != 0,
	}
	if !d.done() {
		return Member{}, errors.New("invalid member record")
	}
	return member, nil
}

func (s *Store) CreateRoom(ctx context.Context, room Room, members []Member) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if room.RoomID == uuid.Nil() {
		return errors.New("room id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var buf [64]byte
	roomKeyBytes := appendRoomKey(buf[:0], room.RoomID)
	exists, err := s.exists(roomKeyBytes)
	if err != nil {
		return fmt.Errorf("check room: %w", err)
	}
	if exists {
		return ErrAlreadyExists
	}

	var singleKey []byte
	if len(room.SingleChatHash) > 0 {
		var sKeyBuf [64]byte
		singleKey = appendSingleRoomKey(sKeyBuf[:0], room.SingleChatHash)
		if existing, err := getRecord(s, singleKey, getUUID); err == nil {
			if existing != room.RoomID {
				return ErrAlreadyExists
			}
		} else if !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("check single room index: %w", err)
		}
	}
	if room.CreatedAt.IsZero() {
		room.CreatedAt = time.Now()
	}
	if room.UpdatedAt.IsZero() {
		room.UpdatedAt = room.CreatedAt
	}
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, roomKeyBytes, encodeRoom(room)); err != nil {
		return err
	}
	if len(singleKey) > 0 {
		if err := s.setBytes(batch, singleKey, room.RoomID[:]); err != nil {
			return err
		}
	}
	var mKeyBuf [64]byte
	var uKeyBuf [64]byte
	for _, member := range members {
		member.RoomID = room.RoomID
		mKey := appendMemberKey(mKeyBuf[:0], room.RoomID, member.UserID)
		if err := s.setBytes(batch, mKey, encodeMember(member)); err != nil {
			return err
		}
		uKey := appendUserRoomKey(uKeyBuf[:0], member.UserID, room.RoomID)
		if err := s.setBytes(batch, uKey, room.RoomID[:]); err != nil {
			return err
		}
	}
	if err := commit(batch); err != nil {
		return err
	}
	s.rooms[room.RoomID] = room
	return nil
}

func (s *Store) Room(ctx context.Context, roomID uuid.UUID) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	s.mu.RLock()
	room, ok := s.rooms[roomID]
	s.mu.RUnlock()
	if ok {
		return room, nil
	}
	var buf [32]byte
	room, err := getRecord(s, appendRoomKey(buf[:0], roomID), decodeRoom)
	if err != nil {
		return Room{}, err
	}
	s.mu.Lock()
	s.rooms[roomID] = room
	s.mu.Unlock()
	return room, nil
}

func (s *Store) RoomBySingleHash(ctx context.Context, hash []byte) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var sKeyBuf [64]byte
	id, err := getRecord(s, appendSingleRoomKey(sKeyBuf[:0], hash), getUUID)
	if err != nil {
		return Room{}, err
	}
	var rKeyBuf [32]byte
	return getRecord(s, appendRoomKey(rKeyBuf[:0], id), decodeRoom)
}

func (s *Store) Members(ctx context.Context, roomID uuid.UUID) ([]Member, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := appendMemberPrefix(nil, roomID)
	upper := prefixUpperBound(prefix)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upper})
	if err != nil {
		return nil, fmt.Errorf("create member iterator: %w", err)
	}
	defer iter.Close()

	var result []Member
	for iter.First(); iter.Valid(); iter.Next() {
		member, err := decodeMember(iter.Value())
		if err != nil {
			return nil, err
		}
		result = append(result, member)
	}
	return result, iter.Error()
}

func (s *Store) RoomsByUser(ctx context.Context, userID uuid.UUID) ([]RoomInfo, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := appendUserRoomPrefix(nil, userID)
	upper := prefixUpperBound(prefix)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var result []RoomInfo
	var mKeyBuf [64]byte
	for iter.First(); iter.Valid(); iter.Next() {
		roomID, err := getUUID(iter.Value())
		if err != nil {
			return nil, err
		}
		room, err := s.Room(ctx, roomID)
		if err != nil {
			return nil, err
		}
		member, err := getRecord(s, appendMemberKey(mKeyBuf[:0], roomID, userID), decodeMember)
		if err != nil {
			return nil, err
		}
		result = append(result, RoomInfo{Room: room, Member: member})
	}
	return result, iter.Error()
}
