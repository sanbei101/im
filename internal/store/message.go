package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeMessage(message Message) []byte {
	var number [8]byte
	totalLen := 16*4 + 8 + 8 + 1 + 1 + 4 + len(message.Payload) + 4 + len(message.Ext)
	if message.ReplyToMsgID != uuid.Nil() {
		totalLen += 16
	}
	data := make([]byte, 0, totalLen)
	data = putUUID(data, message.MsgID)
	data = putUUID(data, message.ClientMsgID)
	data = putUUID(data, message.SenderID)
	data = putUUID(data, message.RoomID)
	putU64(number[:], message.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], message.ServerTime)
	data = append(data, number[:]...)
	if message.ReplyToMsgID != uuid.Nil() {
		data = append(data, 1)
		data = putUUID(data, message.ReplyToMsgID)
	} else {
		data = append(data, 0)
	}
	data = append(data, byte(message.MsgType))
	return appendBytes(appendBytes(data, message.Payload), message.Ext)
}

func decodeMessage(data []byte) (Message, error) {
	d := decoder{data: data}
	message := Message{}
	var err error
	if message.MsgID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.ClientMsgID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.SenderID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.RoomID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.RoomSeq, err = d.u64(); err != nil {
		return Message{}, err
	}
	if message.ServerTime, err = d.i64(); err != nil {
		return Message{}, err
	}
	if d.byte() == 1 {
		if message.ReplyToMsgID, err = d.uuid(); err != nil {
			return Message{}, err
		}
	}
	message.MsgType = MsgType(d.byte())
	if message.Payload, err = d.bytes(); err != nil {
		return Message{}, err
	}
	if message.Ext, err = d.bytes(); err != nil || !d.done() {
		return Message{}, errors.New("invalid message record")
	}
	return message, nil
}

func encodeDedup(value Dedup) []byte {
	var number [8]byte
	data := make([]byte, 0, 16+8+8+32)
	data = putUUID(data, value.MsgID)
	putU64(number[:], value.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], value.ServerTime)
	data = append(data, number[:]...)
	return append(data, value.PayloadSum[:]...)
}

func decodeDedup(data []byte) (Dedup, error) {
	if len(data) != 16+8+8+32 {
		return Dedup{}, errors.New("invalid dedup record")
	}
	d := decoder{data: data}
	value := Dedup{}
	var err error
	if value.MsgID, err = d.uuid(); err != nil {
		return Dedup{}, err
	}
	if value.RoomSeq, err = d.u64(); err != nil {
		return Dedup{}, err
	}
	if value.ServerTime, err = d.i64(); err != nil {
		return Dedup{}, err
	}
	copy(value.PayloadSum[:], d.data[d.pos:])
	return value, nil
}

func (s *Store) WriteMessage(ctx context.Context, message Message) (Message, error) {
	results := s.WriteMessages(ctx, []Message{message})
	if len(results) == 0 {
		return Message{}, ErrClosed
	}
	return results[0].Message, results[0].Err
}

type dedupLookupKey struct {
	room   uuid.UUID
	sender uuid.UUID
	client uuid.UUID
}

func (s *Store) writeMessageBatch(messages []Message, results []MessageWriteResult) {
	batch := s.db.NewBatch()
	defer batch.Close()
	rooms := make(map[uuid.UUID]Room)
	digests := make([][32]byte, len(messages))
	pending := make(map[dedupLookupKey]int, len(messages))
	var allocated []int

	var keyBuf [64]byte
	var msgKeyBuf [25]byte
	var digestBuf [256]byte

	for index := range messages {
		if results[index].Err != nil {
			continue
		}
		message := &messages[index]
		results[index].Message = *message
		room, ok := rooms[message.RoomID]
		if !ok {
			shard := s.getRoomShard(message.RoomID)
			shard.mu.RLock()
			room, ok = shard.rooms[message.RoomID]
			shard.mu.RUnlock()
		}
		if !ok {
			var err error
			if room, err = s.getRecord(appendRoomKey(keyBuf[:0], message.RoomID), decodeRoom); err != nil {
				results[index].Err = err
				continue
			}
		}
		rooms[message.RoomID] = room

		digestInput := digestBuf[:0]
		needLen := len(message.Payload) + len(message.Ext) + 19
		if needLen > len(digestBuf) {
			digestInput = make([]byte, 0, needLen)
		}
		digestInput = append(digestInput, byte(message.MsgType))
		digestInput = append(digestInput, message.Payload...)
		digestInput = append(digestInput, 0)
		digestInput = append(digestInput, message.Ext...)
		if message.ReplyToMsgID != uuid.Nil() {
			digestInput = append(digestInput, message.ReplyToMsgID[:]...)
		}
		digest := sha256.Sum256(digestInput)
		digests[index] = digest

		lookup := dedupLookupKey{room: message.RoomID, sender: message.SenderID, client: message.ClientMsgID}
		if previous, ok := pending[lookup]; ok {
			switch {
			case digests[previous] != digests[index]:
				results[index].Err = ErrConflict
			case results[previous].Err != nil:
				results[index].Err = results[previous].Err
			default:
				results[index].Message = results[previous].Message
			}
			continue
		}

		dedupKeyBytes := appendDedupKey(keyBuf[:0], message.RoomID, message.SenderID, message.ClientMsgID)
		pending[lookup] = index
		if old, err := s.getRecord(dedupKeyBytes, decodeDedup); err == nil {
			if old.PayloadSum != digests[index] {
				results[index].Err = ErrConflict
			} else {
				results[index].Message.MsgID, results[index].Message.RoomSeq, results[index].Message.ServerTime = old.MsgID, old.RoomSeq, old.ServerTime
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			results[index].Err = err
			continue
		}

		room.LastSeq++
		results[index].Message.RoomSeq = room.LastSeq
		results[index].Message.MsgID = uuid.NewV7()
		results[index].Message.ServerTime = time.Now().UnixMicro()
		room.UpdatedAt = time.Now()
		rooms[message.RoomID] = room
		allocated = append(allocated, index)

		msgKey := appendMessageKey(msgKeyBuf[:0], message.RoomID, results[index].Message.RoomSeq)
		if err := s.setBytes(batch, msgKey, encodeMessage(results[index].Message)); err != nil {
			results[index].Err = err
			continue
		}
		var indexKeyBuf [33]byte
		indexKey := appendMsgIDIndexKey(indexKeyBuf[:0], message.RoomID, results[index].Message.MsgID)
		var seqBytes [8]byte
		binary.BigEndian.PutUint64(seqBytes[:], results[index].Message.RoomSeq)
		if err := s.setBytes(batch, indexKey, seqBytes[:]); err != nil {
			results[index].Err = err
			continue
		}
		if err := s.setBytes(batch, dedupKeyBytes, encodeDedup(Dedup{
			MsgID: results[index].Message.MsgID, RoomSeq: results[index].Message.RoomSeq,
			ServerTime: results[index].Message.ServerTime, PayloadSum: digests[index],
		})); err != nil {
			results[index].Err = err
		}
	}
	for roomID := range rooms {
		room := rooms[roomID]
		shard := s.getRoomShard(roomID)
		shard.mu.Lock()
		shard.rooms[roomID] = room
		shard.mu.Unlock()
		roomKeyBytes := appendRoomKey(keyBuf[:0], roomID)
		if err := s.setBytes(batch, roomKeyBytes, encodeRoom(room)); err != nil {
			for index := range results {
				if results[index].Message.RoomID == roomID && results[index].Err == nil {
					results[index].Err = err
				}
			}
		}
	}
	if err := commit(batch); err != nil {
		for index := range results {
			if results[index].Err == nil {
				results[index].Err = fmt.Errorf("commit messages: %w", err)
			}
		}
		return
	}
	// The cache is filled only after the commit succeeded, so a message in the
	// tail cache is always covered by a successful ACK.
	for _, index := range allocated {
		if results[index].Err == nil {
			s.cache.append(&results[index].Message)
		}
	}
}

func (s *Store) Messages(ctx context.Context, roomID uuid.UUID, before uint64, limit int) (MessagePage, error) {
	if err := contextErr(ctx); err != nil {
		return MessagePage{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if messages, hasMore, ok := s.cache.page(roomID, before, limit); ok {
		return MessagePage{Messages: messages, HasMore: hasMore}, nil
	}

	// The history window [lo, hi) spans at most limit+1 seqs. hi needs the
	// room's LastSeq only for the newest page; a missing room stays an empty
	// page, as the plain Pebble scan of a missing prefix would be.
	hi := before
	if before == 0 {
		room, err := s.Room(ctx, roomID)
		if errors.Is(err, ErrNotFound) {
			return MessagePage{Messages: []Message{}}, nil
		}
		if err != nil {
			return MessagePage{}, err
		}
		hi = room.LastSeq + 1
	}
	lo := uint64(1)
	if hi > uint64(limit)+1 {
		lo = hi - uint64(limit+1)
	}

	watermark, err := s.archiveWatermark(roomID)
	if err != nil {
		return MessagePage{}, err
	}
	warmLo := lo
	if watermark >= warmLo {
		warmLo = watermark + 1
	}
	warm, err := s.scanWarm(roomID, warmLo, hi)
	if err != nil {
		return MessagePage{}, err
	}
	cold, err := s.readArchivedRange(ctx, roomID, lo, min64(hi, watermark+1))
	if err != nil {
		return MessagePage{}, err
	}

	ascending := make([]Message, 0, len(warm)+len(cold))
	ascending = append(ascending, cold...)
	ascending = append(ascending, warm...)
	messages := make([]Message, len(ascending))
	for i := range ascending {
		messages[len(ascending)-1-i] = ascending[i]
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	s.cache.put(roomID, ascending)
	return MessagePage{Messages: messages, HasMore: hasMore}, nil
}

// scanWarm returns the Pebble messages with seq in [lo, hi) in ascending order.
func (s *Store) scanWarm(roomID uuid.UUID, lo, hi uint64) ([]Message, error) {
	if hi <= lo {
		return nil, nil
	}
	var lowerBuf, upperBuf [25]byte
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: appendMessageKey(lowerBuf[:0], roomID, lo),
		UpperBound: appendMessageKey(upperBuf[:0], roomID, hi),
	})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var messages []Message
	for iter.First(); iter.Valid(); iter.Next() {
		value := append([]byte(nil), iter.Value()...)
		message, err := decodeMessage(value)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, iter.Error()
}

func min64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func messageUpperBound(prefix []byte, before uint64) []byte {
	if before == 0 {
		upperBound := append([]byte(nil), prefix...)
		for i := len(upperBound) - 1; i >= 0; i-- {
			if upperBound[i] < 0xff {
				upperBound[i]++
				return upperBound[:i+1]
			}
		}
		return nil
	}
	upperBound := make([]byte, len(prefix)+8)
	copy(upperBound, prefix)
	binary.BigEndian.PutUint64(upperBound[len(prefix):], before)
	return upperBound
}

func (s *Store) SearchRoomMessages(
	ctx context.Context,
	roomID uuid.UUID,
	keyword string,
	before uint64,
	limit int,
) ([]Message, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if keyword == "" {
		return nil, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var prefixBuf [32]byte
	prefix := appendMessagePrefix(prefixBuf[:0], roomID)
	upperBound := messageUpperBound(prefix, before)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	kwBytes := []byte(keyword)
	messages := make([]Message, 0, limit)
	for iter.Last(); iter.Valid() && len(messages) < limit; iter.Prev() {
		if err := contextErr(ctx); err != nil {
			return nil, err
		}
		value := append([]byte(nil), iter.Value()...)
		msg, err := decodeMessage(value)
		if err != nil {
			continue
		}
		if msg.MsgType == MsgTypeRecall {
			continue
		}
		if bytes.Contains(bytes.ToLower(msg.Payload), kwBytes) {
			messages = append(messages, msg)
		}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Store) MessageByID(ctx context.Context, roomID, msgID uuid.UUID) (Message, error) {
	if err := contextErr(ctx); err != nil {
		return Message{}, err
	}

	var indexKeyBuf [33]byte
	indexKey := appendMsgIDIndexKey(indexKeyBuf[:0], roomID, msgID)
	data, closer, err := s.db.Get(indexKey)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	defer closer.Close()
	if len(data) != 8 {
		return Message{}, errors.New("invalid message index record")
	}
	seq := binary.BigEndian.Uint64(data)

	var msgKeyBuf [25]byte
	msgKey := appendMessageKey(msgKeyBuf[:0], roomID, seq)
	message, err := s.getRecord(msgKey, decodeMessage)
	if errors.Is(err, ErrNotFound) {
		// The body may already be archived; the 'i' index survives archival
		// precisely so cold messages stay resolvable by id.
		return s.remoteMessage(ctx, roomID, seq)
	}
	return message, err
}

func (s *Store) RecallMessage(
	ctx context.Context,
	roomID, msgID, operatorID uuid.UUID,
	isOwnerOrAdmin bool,
) (Message, error) {
	if err := contextErr(ctx); err != nil {
		return Message{}, err
	}

	var indexKeyBuf [33]byte
	indexKey := appendMsgIDIndexKey(indexKeyBuf[:0], roomID, msgID)
	data, closer, err := s.db.Get(indexKey)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return Message{}, ErrNotFound
		}
		return Message{}, err
	}
	if len(data) != 8 {
		closer.Close()
		return Message{}, errors.New("invalid message index record")
	}
	seq := binary.BigEndian.Uint64(data)
	closer.Close()

	var msgKeyBuf [25]byte
	msgKey := appendMessageKey(msgKeyBuf[:0], roomID, seq)
	msg, err := s.getRecord(msgKey, decodeMessage)
	if errors.Is(err, ErrNotFound) {
		// The body is archived: it left the local store and cannot be
		// recalled anymore, no matter who asks.
		if _, remoteErr := s.remoteMessage(ctx, roomID, seq); remoteErr == nil {
			return Message{}, ErrRecallTimeout
		}
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, err
	}

	if msg.MsgType == MsgTypeRecall {
		return msg, nil
	}

	if operatorID != msg.SenderID && !isOwnerOrAdmin {
		return Message{}, ErrForbidden
	}

	now := time.Now().UnixMicro()
	if !isOwnerOrAdmin && now-msg.ServerTime > 120*1000*1000 {
		return Message{}, ErrRecallTimeout
	}

	msg.MsgType = MsgTypeRecall
	msg.Payload = jsontext.Value(fmt.Sprintf(`{"recalled_by":"%s"}`, operatorID))

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, msgKey, encodeMessage(msg)); err != nil {
		return Message{}, err
	}
	if err := commit(batch); err != nil {
		return Message{}, err
	}
	// A message inside the recall window is always in the tail cache; keep the
	// cached copy in sync so reads do not serve stale content.
	s.cache.replace(roomID, &msg)
	return msg, nil
}

func (s *Store) AddReaction(ctx context.Context, roomID, msgID, userID uuid.UUID, emoji string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if emoji == "" {
		return errors.New("emoji cannot be empty")
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	key := appendReactionKey(nil, roomID, msgID, userID, emoji)
	if err := s.setBytes(batch, key, []byte{}); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) RemoveReaction(ctx context.Context, roomID, msgID, userID uuid.UUID, emoji string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	key := appendReactionKey(nil, roomID, msgID, userID, emoji)
	if err := batch.Delete(key, nil); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) Reactions(ctx context.Context, roomID, msgID uuid.UUID) ([]ReactionGroup, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	prefix := appendReactionPrefix(nil, roomID, msgID)
	upperBound := prefixUpperBound(prefix)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	prefixLen := len(prefix)
	groups := make(map[string][]uuid.UUID)
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) <= prefixLen+16 {
			continue
		}
		var user uuid.UUID
		copy(user[:], key[prefixLen:prefixLen+16])
		emoji := string(key[prefixLen+16:])
		groups[emoji] = append(groups[emoji], user)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	result := make([]ReactionGroup, 0, len(groups))
	for emoji, users := range groups {
		result = append(result, ReactionGroup{
			Emoji:   emoji,
			Count:   len(users),
			UserIDs: users,
		})
	}
	return result, nil
}

func (s *Store) ReadUsers(ctx context.Context, roomID, msgID uuid.UUID) ([]uuid.UUID, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	msg, err := s.MessageByID(ctx, roomID, msgID)
	if err != nil {
		return nil, err
	}

	members, err := s.Members(ctx, roomID)
	if err != nil {
		return nil, err
	}

	var readUsers []uuid.UUID
	var keyBuf [33]byte
	for _, m := range members {
		key := appendReadSeqKey(keyBuf[:0], m.UserID, roomID)
		val, closer, err := s.db.Get(key)
		if err != nil {
			if errors.Is(err, pebble.ErrNotFound) {
				continue
			}
			return nil, err
		}
		readSeq := binary.BigEndian.Uint64(val)
		closer.Close()
		if readSeq >= msg.RoomSeq {
			readUsers = append(readUsers, m.UserID)
		}
	}
	return readUsers, nil
}

func (s *Store) PinMessage(ctx context.Context, roomID, msgID, operatorID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if _, err := s.MessageByID(ctx, roomID, msgID); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	key := appendPinKey(nil, roomID, msgID)
	var val [24]byte
	copy(val[:16], operatorID[:])
	binary.BigEndian.PutUint64(val[16:], uint64(time.Now().Unix()))

	if err := s.setBytes(batch, key, val[:]); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) UnpinMessage(ctx context.Context, roomID, msgID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	key := appendPinKey(nil, roomID, msgID)
	if err := batch.Delete(key, nil); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) PinnedMessages(ctx context.Context, roomID uuid.UUID) ([]Message, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	prefix := appendPinPrefix(nil, roomID)
	upperBound := prefixUpperBound(prefix)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var msgIDs []uuid.UUID
	prefixLen := len(prefix)
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) != prefixLen+16 {
			continue
		}
		var msgID uuid.UUID
		copy(msgID[:], key[prefixLen:])
		msgIDs = append(msgIDs, msgID)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(msgIDs))
	var indexKeyBuf [33]byte
	var msgKeyBuf [25]byte
	for _, id := range msgIDs {
		indexKey := appendMsgIDIndexKey(indexKeyBuf[:0], roomID, id)
		data, closer, err := s.db.Get(indexKey)
		if err != nil {
			continue
		}
		if len(data) != 8 {
			closer.Close()
			continue
		}
		seq := binary.BigEndian.Uint64(data)
		closer.Close()

		msgKey := appendMessageKey(msgKeyBuf[:0], roomID, seq)
		msg, err := s.getRecord(msgKey, decodeMessage)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

func (s *Store) SaveDeviceToken(ctx context.Context, userID uuid.UUID, info DeviceInfo) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	key := appendDeviceKey(nil, userID)
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("marshal device info: %w", err)
	}
	if err := s.setBytes(batch, key, data); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) DeleteDeviceToken(ctx context.Context, userID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	key := appendDeviceKey(nil, userID)
	if err := batch.Delete(key, nil); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) DeviceToken(ctx context.Context, userID uuid.UUID) (DeviceInfo, error) {
	if err := contextErr(ctx); err != nil {
		return DeviceInfo{}, err
	}

	key := appendDeviceKey(nil, userID)
	data, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return DeviceInfo{}, ErrNotFound
		}
		return DeviceInfo{}, err
	}
	defer closer.Close()

	var info DeviceInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return DeviceInfo{}, fmt.Errorf("unmarshal device info: %w", err)
	}
	return info, nil
}
