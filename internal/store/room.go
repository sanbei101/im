package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeRoom(room Room) []byte {
	var number [8]byte
	capacity := 16 + 4*5 + len(room.ChatType) + len(room.Name) + len(room.AvatarURL) + len(room.Notice) +
		len(room.SingleChatHash) + 8*3
	data := make([]byte, 0, capacity)
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
	notice, err := d.string()
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
		Notice:         notice,
		SingleChatHash: bytes.Clone(hash),
		LastSeq:        seq,
		CreatedAt:      time.UnixMicro(created),
		UpdatedAt:      time.UnixMicro(updated),
	}, nil
}

func encodeMember(member Member) []byte {
	data := make([]byte, 0, 16*2+4+len(member.Role)+3)
	data = putUUID(data, member.RoomID)
	data = putUUID(data, member.UserID)
	data = appendString(data, member.Role)
	data = append(data, boolByte(member.IsHidden), boolByte(member.IsMuted), boolByte(member.IsPinned))
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
		IsPinned: d.byte() != 0,
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
		if existing, err := s.getRecord(singleKey, getUUID); err == nil {
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
	s.roomsMu.Lock()
	s.rooms[room.RoomID] = room
	s.roomsMu.Unlock()
	return nil
}

func (s *Store) Room(ctx context.Context, roomID uuid.UUID) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	s.roomsMu.RLock()
	room, ok := s.rooms[roomID]
	s.roomsMu.RUnlock()
	if ok {
		return room, nil
	}
	var buf [32]byte
	room, err := s.getRecord(appendRoomKey(buf[:0], roomID), decodeRoom)
	if err != nil {
		return Room{}, err
	}
	s.roomsMu.Lock()
	s.rooms[roomID] = room
	s.roomsMu.Unlock()
	return room, nil
}

func (s *Store) RoomBySingleHash(ctx context.Context, hash []byte) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}
	var sKeyBuf [64]byte
	id, err := s.getRecord(appendSingleRoomKey(sKeyBuf[:0], hash), getUUID)
	if err != nil {
		return Room{}, err
	}
	var rKeyBuf [32]byte
	return s.getRecord(appendRoomKey(rKeyBuf[:0], id), decodeRoom)
}

func (s *Store) Members(ctx context.Context, roomID uuid.UUID) ([]Member, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	return s.scanPrefix(appendMemberPrefix(nil, roomID), decodeMember)
}

func (s *Store) RoomsByUser(ctx context.Context, userID uuid.UUID) ([]RoomInfo, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

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
		member, err := s.getRecord(appendMemberKey(mKeyBuf[:0], roomID, userID), decodeMember)
		if err != nil {
			return nil, err
		}
		result = append(result, RoomInfo{Room: room, Member: member})
	}
	return result, iter.Error()
}

func (s *Store) Member(ctx context.Context, roomID, userID uuid.UUID) (Member, error) {
	if err := contextErr(ctx); err != nil {
		return Member{}, err
	}

	var mKeyBuf [64]byte
	return s.getRecord(appendMemberKey(mKeyBuf[:0], roomID, userID), decodeMember)
}

func (s *Store) UpdateRoom(ctx context.Context, roomID uuid.UUID, name, avatarURL, notice string) (Room, error) {
	if err := contextErr(ctx); err != nil {
		return Room{}, err
	}

	var buf [32]byte
	key := appendRoomKey(buf[:0], roomID)
	room, err := s.getRecord(key, decodeRoom)
	if err != nil {
		return Room{}, err
	}

	if name != "" {
		room.Name = name
	}
	if avatarURL != "" {
		room.AvatarURL = avatarURL
	}
	if notice != "" {
		room.Notice = notice
	}
	room.UpdatedAt = time.Now()

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, key, encodeRoom(room)); err != nil {
		return Room{}, err
	}
	if err := commit(batch); err != nil {
		return Room{}, err
	}
	s.roomsMu.Lock()
	s.rooms[roomID] = room
	s.roomsMu.Unlock()
	return room, nil
}

func (s *Store) AddMembers(ctx context.Context, roomID uuid.UUID, newMembers []Member) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	var mKeyBuf [64]byte
	var uKeyBuf [64]byte
	for _, member := range newMembers {
		member.RoomID = roomID
		mKey := appendMemberKey(mKeyBuf[:0], roomID, member.UserID)
		if err := s.setBytes(batch, mKey, encodeMember(member)); err != nil {
			return err
		}
		uKey := appendUserRoomKey(uKeyBuf[:0], member.UserID, roomID)
		if err := s.setBytes(batch, uKey, roomID[:]); err != nil {
			return err
		}
	}
	return commit(batch)
}

func (s *Store) RemoveMember(ctx context.Context, roomID, userID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	var mKeyBuf [64]byte
	var uKeyBuf [64]byte
	if err := batch.Delete(appendMemberKey(mKeyBuf[:0], roomID, userID), nil); err != nil {
		return err
	}
	if err := batch.Delete(appendUserRoomKey(uKeyBuf[:0], userID, roomID), nil); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) UpdateMemberRole(ctx context.Context, roomID, userID uuid.UUID, role string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var mKeyBuf [64]byte
	mKey := appendMemberKey(mKeyBuf[:0], roomID, userID)
	member, err := s.getRecord(mKey, decodeMember)
	if err != nil {
		return err
	}

	member.Role = role
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, mKey, encodeMember(member)); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) UpdateMemberSettings(ctx context.Context, roomID, userID uuid.UUID, isPinned, isMuted *bool) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var mKeyBuf [64]byte
	mKey := appendMemberKey(mKeyBuf[:0], roomID, userID)
	member, err := s.getRecord(mKey, decodeMember)
	if err != nil {
		return err
	}

	if isPinned != nil {
		member.IsPinned = *isPinned
	}
	if isMuted != nil {
		member.IsMuted = *isMuted
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, mKey, encodeMember(member)); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) DissolveRoom(ctx context.Context, roomID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	members, err := s.Members(ctx, roomID)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	var rKeyBuf [32]byte
	if err := batch.Delete(appendRoomKey(rKeyBuf[:0], roomID), nil); err != nil {
		return err
	}

	var mKeyBuf [64]byte
	var uKeyBuf [64]byte
	for _, m := range members {
		if err := batch.Delete(appendMemberKey(mKeyBuf[:0], roomID, m.UserID), nil); err != nil {
			return err
		}
		if err := batch.Delete(appendUserRoomKey(uKeyBuf[:0], m.UserID, roomID), nil); err != nil {
			return err
		}
	}
	if err := commit(batch); err != nil {
		return err
	}
	s.roomsMu.Lock()
	delete(s.rooms, roomID)
	s.roomsMu.Unlock()
	return nil
}

func (s *Store) MarkRoomRead(ctx context.Context, userID, roomID uuid.UUID, readSeq uint64) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var buf [64]byte
	key := appendReadSeqKey(buf[:0], userID, roomID)
	var val [8]byte
	putU64(val[:], readSeq)

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, key, val[:]); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) ReadSeq(ctx context.Context, userID, roomID uuid.UUID) (uint64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}

	var buf [64]byte
	key := appendReadSeqKey(buf[:0], userID, roomID)
	return s.getRecord(key, getU64)
}

func (s *Store) Conversations(ctx context.Context, userID uuid.UUID) ([]ConversationInfo, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	rooms, err := s.RoomsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]ConversationInfo, 0, len(rooms))
	var qKeyBuf [64]byte
	var mKeyBuf [64]byte

	for i := range rooms {
		room := rooms[i].Room
		member := rooms[i].Member

		readSeqKey := appendReadSeqKey(qKeyBuf[:0], userID, room.RoomID)
		readSeq, err := s.getRecord(readSeqKey, getU64)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, err
		}

		var unread uint64
		if room.LastSeq > readSeq {
			unread = room.LastSeq - readSeq
		}

		info := ConversationInfo{
			Room:        room,
			Member:      member,
			UnreadCount: unread,
		}

		if room.LastSeq > 0 {
			msgKey := appendMessageKey(mKeyBuf[:0], room.RoomID, room.LastSeq)
			if lastMsg, err := s.getRecord(msgKey, decodeMessage); err == nil {
				info.LastMessage = &lastMsg
			}
		}

		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Member.IsPinned != result[j].Member.IsPinned {
			return result[i].Member.IsPinned
		}
		timeI := result[i].Room.UpdatedAt.UnixMicro()
		if result[i].LastMessage != nil && result[i].LastMessage.ServerTime > timeI {
			timeI = result[i].LastMessage.ServerTime
		}
		timeJ := result[j].Room.UpdatedAt.UnixMicro()
		if result[j].LastMessage != nil && result[j].LastMessage.ServerTime > timeJ {
			timeJ = result[j].LastMessage.ServerTime
		}
		return timeI > timeJ
	})

	return result, nil
}
