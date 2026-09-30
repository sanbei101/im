package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeRoom(room Room) []byte {
	var id [16]byte
	putUUID(id[:], room.RoomID)
	data := make([]byte, 0, 16+4+len(room.ChatType)+4+len(room.Name)+4+len(room.AvatarURL)+4+len(room.SingleChatHash)+8*3)
	data = append(data, id[:]...)
	data = appendString(data, room.ChatType)
	data = appendString(data, room.Name)
	data = appendString(data, room.AvatarURL)
	data = appendBytes(data, room.SingleChatHash)
	var number [8]byte
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
		SingleChatHash: hash,
		LastSeq:        seq,
		CreatedAt:      time.UnixMicro(created),
		UpdatedAt:      time.UnixMicro(updated),
	}, nil
}

func encodeMember(member Member) []byte {
	var id [16]byte
	putUUID(id[:], member.RoomID)
	data := make([]byte, 0, 16*2+4+len(member.Role)+2)
	data = append(data, id[:]...)
	putUUID(id[:], member.UserID)
	data = append(data, id[:]...)
	data = appendString(data, member.Role)
	if member.IsHidden {
		data = append(data, 1)
	} else {
		data = append(data, 0)
	}
	if member.IsMuted {
		data = append(data, 1)
	} else {
		data = append(data, 0)
	}
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
	if err != nil || d.pos+2 > len(d.data) {
		return Member{}, errors.New("invalid member record")
	}
	member := Member{
		RoomID:   roomID,
		UserID:   userID,
		Role:     role,
		IsHidden: d.data[d.pos] != 0,
		IsMuted:  d.data[d.pos+1] != 0,
	}
	d.pos += 2
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
	if _, closer, err := s.db.Get([]byte(roomKey(room.RoomID))); err == nil {
		closer.Close()
		return ErrAlreadyExists
	} else if !errors.Is(err, pebble.ErrNotFound) {
		return fmt.Errorf("check room: %w", err)
	}
	if len(room.SingleChatHash) > 0 {
		if value, closer, err := s.db.Get([]byte(singleRoomKey(room.SingleChatHash))); err == nil {
			closer.Close()
			if existing, parseErr := getUUID(value); parseErr == nil && existing != room.RoomID {
				return ErrAlreadyExists
			}
		} else if !errors.Is(err, pebble.ErrNotFound) {
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
	if err := s.set(batch, roomKey(room.RoomID), encodeRoom(room)); err != nil {
		return err
	}
	if len(room.SingleChatHash) > 0 {
		if err := s.set(batch, singleRoomKey(room.SingleChatHash), room.RoomID[:]); err != nil {
			return err
		}
	}
	for _, member := range members {
		member.RoomID = room.RoomID
		if err := s.set(batch, roomMemberKey(room.RoomID, member.UserID), encodeMember(member)); err != nil {
			return err
		}
		if err := s.set(batch, userRoomKey(member.UserID, room.RoomID), room.RoomID[:]); err != nil {
			return err
		}
	}
	return commit(batch)
}

func (s *Store) Room(ctx context.Context, roomID uuid.UUID) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var room Room
	if err := s.get([]byte(roomKey(roomID)), func(data []byte) error {
		var err error
		room, err = decodeRoom(data)
		return err
	}); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (s *Store) RoomBySingleHash(ctx context.Context, hash []byte) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	var id uuid.UUID
	if err := s.get([]byte(singleRoomKey(hash)), func(data []byte) error {
		var err error
		id, err = getUUID(data)
		return err
	}); err != nil {
		return Room{}, err
	}
	var room Room
	if err := s.get([]byte(roomKey(id)), func(data []byte) error {
		var err error
		room, err = decodeRoom(data)
		return err
	}); err != nil {
		return Room{}, err
	}
	return room, nil
}

func (s *Store) Members(ctx context.Context, roomID uuid.UUID) ([]Member, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	prefix := []byte(roomMemberPrefix(roomID))
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix})
	if err != nil {
		return nil, fmt.Errorf("create member iterator: %w", err)
	}
	defer iter.Close()
	var result []Member
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), string(prefix)) {
			break
		}
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
	prefix := []byte(userRoomPrefix(userID))
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix})
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var result []RoomInfo
	for iter.First(); iter.Valid(); iter.Next() {
		if !strings.HasPrefix(string(iter.Key()), string(prefix)) {
			break
		}
		roomID, err := getUUID(iter.Value())
		if err != nil {
			return nil, err
		}
		var room Room
		if err := s.get([]byte(roomKey(roomID)), func(data []byte) error {
			var err error
			room, err = decodeRoom(data)
			return err
		}); err != nil {
			return nil, err
		}
		var member Member
		if err := s.get([]byte(roomMemberKey(roomID, userID)), func(data []byte) error {
			var err error
			member, err = decodeMember(data)
			return err
		}); err != nil {
			return nil, err
		}
		result = append(result, RoomInfo{Room: room, Member: member})
	}
	return result, iter.Error()
}
