package store

import (
	"bytes"
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
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
	id, err := d.uuid()
	if err != nil {
		return Room{}, err
	}
	chatType, err := d.bytes()
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
	if err != nil {
		return Room{}, err
	}
	var lastMsg *Message
	if d.byte() == 1 {
		record, err := d.bytes()
		if err != nil {
			return Room{}, err
		}
		msg, err := decodeMessage(record)
		if err != nil {
			return Room{}, err
		}
		lastMsg = &msg
	}
	if !d.done() {
		return Room{}, errors.New("invalid room record")
	}
	return Room{
		RoomID:         id,
		ChatType:       internEnum(chatType),
		Name:           name,
		AvatarURL:      avatar,
		Notice:         notice,
		SingleChatHash: bytes.Clone(hash),
		LastSeq:        seq,
		LastMsg:        lastMsg,
		CreatedAt:      time.UnixMicro(created),
		UpdatedAt:      time.UnixMicro(updated),
	}, nil
}

// decodeRoomHead parses only the leading fields of a room record, without
// materializing strings or decoding the embedded last message. It serves
// scans that need just the room id and LastSeq, such as the archiver.
func decodeRoomHead(data []byte) (Room, error) {
	d := decoder{data: data}
	id, err := d.uuid()
	if err != nil {
		return Room{}, err
	}
	// chatType, name, avatarURL, notice, singleChatHash: length-prefixed,
	// skipped without copying.
	for range 5 {
		if err := skipBytes(&d); err != nil {
			return Room{}, err
		}
	}
	seq, err := d.u64()
	if err != nil {
		return Room{}, err
	}
	if _, err = d.i64(); err != nil {
		return Room{}, err
	}
	if _, err = d.i64(); err != nil {
		return Room{}, err
	}
	if d.byte() == 1 {
		if err := skipBytes(&d); err != nil {
			return Room{}, err
		}
	}
	if !d.done() {
		return Room{}, errors.New("invalid room record")
	}
	return Room{RoomID: id, LastSeq: seq}, nil
}

// skipBytes consumes one length-prefixed byte slice without copying it.
func skipBytes(d *decoder) error {
	_, err := d.bytes()
	return err
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
	role, err := d.bytes()
	if err != nil {
		return Member{}, errors.New("invalid member record")
	}
	member := Member{
		RoomID:   roomID,
		UserID:   userID,
		Role:     internEnum(role),
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
		// The user-room index stores the member record itself, so
		// RoomsByUser reads members in one range scan instead of a get per room.
		uKey := appendUserRoomKey(uKeyBuf[:0], member.UserID, room.RoomID)
		if err := s.setBytes(batch, uKey, encodeMember(member)); err != nil {
			return err
		}
	}
	if err := commit(batch); err != nil {
		return err
	}
	s.getRoomShard(room.RoomID).rooms.Set(room.RoomID, room)
	return nil
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

	var buf [17]byte
	return s.scanPrefix(appendMemberPrefix(buf[:0], roomID), decodeMember)
}

func (s *Store) RoomsByUser(ctx context.Context, userID uuid.UUID) ([]RoomInfo, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	var pBuf [17]byte
	prefix := appendUserRoomPrefix(pBuf[:0], userID)
	var upBuf [17]byte
	upper := prefixUpperBoundBuf(upBuf[:0], prefix)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	result := make([]RoomInfo, 0, 16)
	for iter.First(); iter.Valid(); iter.Next() {
		member, err := decodeMember(iter.Value())
		if err != nil {
			return nil, err
		}
		room, err := s.Room(ctx, member.RoomID)
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
	s.getRoomShard(roomID).rooms.Set(roomID, room)
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
		if err := s.setBytes(batch, uKey, encodeMember(member)); err != nil {
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
	var qKeyBuf [33]byte
	var qRoomBuf [33]byte
	if err := batch.Delete(appendMemberKey(mKeyBuf[:0], roomID, userID), nil); err != nil {
		return err
	}
	if err := batch.Delete(appendUserRoomKey(uKeyBuf[:0], userID, roomID), nil); err != nil {
		return err
	}
	if err := batch.Delete(appendReadSeqKey(qKeyBuf[:0], userID, roomID), nil); err != nil {
		return err
	}
	if err := batch.Delete(appendRoomReadSeqKey(qRoomBuf[:0], roomID, userID), nil); err != nil {
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
	return s.commitMember(member)
}

// TransferOwnership swaps the owner role between two members in one batch, so
// a crash can never leave a room with zero or two owners.
func (s *Store) TransferOwnership(ctx context.Context, roomID, from, to uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var fromBuf, toBuf [64]byte
	fromMember, err := s.getRecord(appendMemberKey(fromBuf[:0], roomID, from), decodeMember)
	if err != nil {
		return err
	}
	toMember, err := s.getRecord(appendMemberKey(toBuf[:0], roomID, to), decodeMember)
	if err != nil {
		return err
	}

	fromMember.Role = RoleMember
	toMember.Role = RoleOwner

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setMemberBatch(batch, fromMember); err != nil {
		return err
	}
	if err := s.setMemberBatch(batch, toMember); err != nil {
		return err
	}
	return commit(batch)
}

// commitMember persists a member record and its user-room index copy in one batch.
func (s *Store) commitMember(member Member) error {
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setMemberBatch(batch, member); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) setMemberBatch(batch *pebble.Batch, member Member) error {
	var mKeyBuf [64]byte
	mKey := appendMemberKey(mKeyBuf[:0], member.RoomID, member.UserID)
	if err := s.setBytes(batch, mKey, encodeMember(member)); err != nil {
		return err
	}
	// Keep the user-room index copy in sync with the member record.
	var uKeyBuf [64]byte
	uKey := appendUserRoomKey(uKeyBuf[:0], member.UserID, member.RoomID)
	return s.setBytes(batch, uKey, encodeMember(member))
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

	return s.commitMember(member)
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
	var qKeyBuf [33]byte
	var qRoomBuf [33]byte
	for _, m := range members {
		if err := batch.Delete(appendMemberKey(mKeyBuf[:0], roomID, m.UserID), nil); err != nil {
			return err
		}
		if err := batch.Delete(appendUserRoomKey(uKeyBuf[:0], m.UserID, roomID), nil); err != nil {
			return err
		}
		if err := batch.Delete(appendReadSeqKey(qKeyBuf[:0], m.UserID, roomID), nil); err != nil {
			return err
		}
		if err := batch.Delete(appendRoomReadSeqKey(qRoomBuf[:0], roomID, m.UserID), nil); err != nil {
			return err
		}
	}
	if err := commit(batch); err != nil {
		return err
	}
	s.getRoomShard(roomID).rooms.Delete(roomID)
	return nil
}

// MarkRoomRead records the user's read watermark idempotently in both key
// orders: 'q' (user-major) serves Conversations, 'Q' (room-major) serves the
// ReadUsers block scan. Both land in one batch, so the two views stay atomic.
func (s *Store) MarkRoomRead(ctx context.Context, userID, roomID uuid.UUID, readSeq uint64) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var userBuf [33]byte
	var roomBuf [33]byte
	userKey := appendReadSeqKey(userBuf[:0], userID, roomID)
	roomKey := appendRoomReadSeqKey(roomBuf[:0], roomID, userID)
	var val [8]byte
	putU64(val[:], readSeq)

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, userKey, val[:]); err != nil {
		return err
	}
	if err := s.setBytes(batch, roomKey, val[:]); err != nil {
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

// userReadSeqs returns every read marker of the user in one range scan over
// the user-major 'q' index, keyed by room.
func (s *Store) userReadSeqs(userID uuid.UUID) (map[uuid.UUID]uint64, error) {
	var buf [17]byte
	prefix := appendReadSeqPrefix(buf[:0], userID)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	seqs := make(map[uuid.UUID]uint64)
	for iter.First(); iter.Valid(); iter.Next() {
		var room uuid.UUID
		copy(room[:], iter.Key()[len(prefix):])
		seqs[room] = binary.BigEndian.Uint64(iter.Value())
	}
	return seqs, iter.Error()
}

// roomReadSeqs mirrors userReadSeqs over the room-major 'Q' index, keyed by user.
func (s *Store) roomReadSeqs(roomID uuid.UUID) (map[uuid.UUID]uint64, error) {
	var buf [17]byte
	prefix := appendRoomReadSeqPrefix(buf[:0], roomID)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	seqs := make(map[uuid.UUID]uint64)
	for iter.First(); iter.Valid(); iter.Next() {
		var user uuid.UUID
		copy(user[:], iter.Key()[len(prefix):])
		seqs[user] = binary.BigEndian.Uint64(iter.Value())
	}
	return seqs, iter.Error()
}

func (s *Store) Conversations(ctx context.Context, userID uuid.UUID) ([]ConversationInfo, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	rooms, err := s.RoomsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	readSeqs, err := s.userReadSeqs(userID)
	if err != nil {
		return nil, err
	}

	result := make([]ConversationInfo, 0, len(rooms))

	for i := range rooms {
		room := rooms[i].Room
		member := rooms[i].Member

		var unread uint64
		if readSeq := readSeqs[room.RoomID]; room.LastSeq > readSeq {
			unread = room.LastSeq - readSeq
		}

		info := ConversationInfo{
			Room:        room,
			Member:      member,
			UnreadCount: unread,
			LastMessage: room.LastMsg,
		}

		result = append(result, info)
	}

	slices.SortFunc(result, func(a, b ConversationInfo) int {
		if a.Member.IsPinned != b.Member.IsPinned {
			if a.Member.IsPinned {
				return -1
			}
			return 1
		}
		return cmp.Compare(lastActive(b), lastActive(a))
	})

	return result, nil
}

// lastActive is the sort key of a conversation: its newest activity time.
func lastActive(info ConversationInfo) int64 {
	t := info.Room.UpdatedAt.UnixMicro()
	if info.LastMessage != nil && info.LastMessage.ServerTime > t {
		return info.LastMessage.ServerTime
	}
	return t
}
