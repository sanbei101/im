package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeMessage(message Message) []byte {
	return appendMessage(nil, message)
}

// appendMessage encodes into dst so hot batch writes can reuse one buffer;
// Pebble copies the record on batch.Set.
func appendMessage(data []byte, message Message) []byte {
	var number [8]byte
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
	message := Message{
		MsgID:       d.uuid(),
		ClientMsgID: d.uuid(),
		SenderID:    d.uuid(),
		RoomID:      d.uuid(),
		RoomSeq:     d.u64(),
		ServerTime:  d.i64(),
	}
	if d.byte() == 1 {
		message.ReplyToMsgID = d.uuid()
	}
	message.MsgType = MsgType(d.byte())
	message.Payload = d.bytes()
	message.Ext = d.bytes()
	if err := d.check("message"); err != nil {
		return Message{}, err
	}
	return message, nil
}

// ownedMessage copies Message's variable-length fields out of a pebble-owned
// buffer. decodeMessage aliases the Get/iterator value, which is invalid once
// its closer is closed; callers that keep the Message beyond that (caches,
// responses) must hold an owned copy.
func ownedMessage(m Message) Message {
	m.Payload = jsontext.Value(bytes.Clone(m.Payload))
	m.Ext = jsontext.Value(bytes.Clone(m.Ext))
	return m
}

// appendDedup encodes into dst; the record is fixed-size, so batch writes
// pass a stack buffer and allocate nothing.
func appendDedup(data []byte, value Dedup) []byte {
	var number [8]byte
	data = putUUID(data, value.MsgID)
	putU64(number[:], value.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], value.ServerTime)
	data = append(data, number[:]...)
	return append(data, value.PayloadSum[:]...)
}

func decodeDedup(data []byte) (Dedup, error) {
	d := decoder{data: data}
	value := Dedup{
		MsgID:      d.uuid(),
		RoomSeq:    d.u64(),
		ServerTime: d.i64(),
	}
	copy(value.PayloadSum[:], d.fixed(len(value.PayloadSum)))
	if err := d.check("dedup"); err != nil {
		return Dedup{}, err
	}
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

// dedupEntry is a dedup record staged for the in-memory cache; entries are
// cached only after the batch commit succeeds, so a cached entry is always
// covered by a successful ACK.
type dedupEntry struct {
	lookup dedupLookupKey
	dedup  Dedup
}

func (s *Store) writeMessageBatch(
	messages []Message,
	results []MessageWriteResult,
) {
	batch := s.db.NewBatch()
	defer batch.Close()
	rooms := make(map[uuid.UUID]Room)
	digests := make([][32]byte, len(messages))
	pending := make(map[dedupLookupKey]int, len(messages))
	var allocated []int
	var dedupEntries []dedupEntry
	failedRooms := make(map[uuid.UUID]struct{})

	var keyBuf [64]byte
	var msgKeyBuf [25]byte
	recBuf := s.recordScratch
	roomBuf := s.roomScratch
	var digestInput []byte

	for index := range messages {
		if results[index].Err != nil {
			continue
		}
		message := &messages[index]
		results[index].Message = *message
		room, ok := rooms[message.RoomID]
		if !ok {
			room, ok = s.getRoomShard(message.RoomID).rooms.Get(message.RoomID)
		}
		if !ok {
			var err error
			room, err = s.getRecord(appendRoomKey(keyBuf[:0], message.RoomID), decodeRoom)
			if errors.Is(err, ErrNotFound) {
				// Room metadata lives in PostgreSQL. The store only keeps the
				// message head, created lazily on the first committed message.
				room = Room{RoomID: message.RoomID}
			} else if err != nil {
				results[index].Err = err
				continue
			}
			room.RoomID = message.RoomID
		}
		rooms[message.RoomID] = room

		digestInput = append(digestInput[:0], byte(message.MsgType))
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

		if old, ok := s.dedupCache.Get(lookup); ok {
			pending[lookup] = index
			if old.PayloadSum != digests[index] {
				results[index].Err = ErrConflict
			} else {
				results[index].Message.MsgID, results[index].Message.RoomSeq, results[index].Message.ServerTime = old.MsgID, old.RoomSeq, old.ServerTime
			}
			continue
		}

		dedupKeyBytes := appendDedupKey(keyBuf[:0], message.RoomID, message.SenderID, message.ClientMsgID)
		pending[lookup] = index
		if old, err := s.getRecord(dedupKeyBytes, decodeDedup); err == nil {
			s.dedupCache.Set(lookup, old)
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
		recBuf = appendMessage(recBuf[:0], results[index].Message)
		if err := s.setBytes(batch, msgKey, recBuf); err != nil {
			results[index].Err = err
			continue
		}
		var indexKeyBuf [33]byte
		indexKey := appendMsgIDIndexKey(indexKeyBuf[:0], message.RoomID, results[index].Message.MsgID)
		var seqBytes [8]byte
		binary.BigEndian.PutUint64(seqBytes[:], results[index].Message.RoomSeq)
		if err := s.setBytes(batch, indexKey, seqBytes[:]); err != nil {
			results[index].Err = err
			if delErr := batch.Delete(msgKey, nil); delErr != nil {
				results[index].Err = errors.New("stage message index: " + err.Error() + "; rollback: " + delErr.Error())
			}
			continue
		}
		dedup := Dedup{
			MsgID: results[index].Message.MsgID, RoomSeq: results[index].Message.RoomSeq,
			ServerTime: results[index].Message.ServerTime, PayloadSum: digests[index],
		}
		if err := s.setBytes(batch, dedupKeyBytes, appendDedup(s.dedupScratch[:0], dedup)); err != nil {
			results[index].Err = err
			if delErr := batch.Delete(msgKey, nil); delErr != nil {
				results[index].Err = errors.New("stage dedup: " + err.Error() + "; rollback: " + delErr.Error())
				continue
			}
			if delErr := batch.Delete(indexKey, nil); delErr != nil {
				results[index].Err = errors.New("stage dedup: " + err.Error() + "; rollback: " + delErr.Error())
			}
			continue
		}
		dedupEntries = append(dedupEntries, dedupEntry{lookup: lookup, dedup: dedup})
	}
	for roomID := range rooms {
		room := rooms[roomID]
		roomKeyBytes := appendRoomKey(keyBuf[:0], roomID)
		roomBuf = appendRoomHead(roomBuf[:0], room.LastSeq)
		if err := s.setBytes(batch, roomKeyBytes, roomBuf); err != nil {
			failedRooms[roomID] = struct{}{}
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
	s.recordScratch, s.roomScratch, s.digestScratch = recBuf, roomBuf, digestInput
	// Caches are filled only after the commit succeeded: a failed commit must
	// not leave in-memory state ahead of disk.
	for roomID := range rooms {
		if _, ok := failedRooms[roomID]; ok {
			continue
		}
		room := rooms[roomID]
		s.getRoomShard(roomID).rooms.Set(roomID, room)
	}
	for _, entry := range dedupEntries {
		s.dedupCache.Set(entry.lookup, entry.dedup)
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
		if err := s.populateReactions(ctx, roomID, messages); err != nil {
			return MessagePage{}, err
		}
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
	if s.remote == nil && lo <= watermark {
		return MessagePage{}, fmt.Errorf(
			"messages %d..%d are archived but no remote object store is configured",
			lo,
			min(hi, watermark+1),
		)
	}
	warmLo := lo
	if watermark >= warmLo {
		warmLo = watermark + 1
	}
	warm, err := s.scanWarm(roomID, warmLo, hi)
	if err != nil {
		return MessagePage{}, err
	}
	cold, err := s.readArchivedRange(ctx, roomID, lo, min(hi, watermark+1))
	if err != nil {
		return MessagePage{}, err
	}

	total := len(warm) + len(cold)
	hasMore := total > limit
	n := total
	if hasMore {
		n = limit
	}
	messages := make([]Message, n)
	idx := 0
	for i := len(warm) - 1; i >= 0 && idx < n; i-- {
		messages[idx] = warm[i]
		idx++
	}
	for i := len(cold) - 1; i >= 0 && idx < n; i-- {
		messages[idx] = cold[i]
		idx++
	}

	if before == 0 && total > 0 {
		ascending := make([]Message, 0, total)
		ascending = append(ascending, cold...)
		ascending = append(ascending, warm...)
		s.cache.put(roomID, ascending, hi-1)
	}
	if err := s.populateReactions(ctx, roomID, messages); err != nil {
		return MessagePage{}, err
	}
	return MessagePage{Messages: messages, HasMore: hasMore}, nil
}

// MessagesAfter returns messages with room_seq in (after, min(after+limit, LastSeq)] in ascending order.
func (s *Store) MessagesAfter(ctx context.Context, roomID uuid.UUID, after uint64, limit int) (MessagePage, error) {
	if err := contextErr(ctx); err != nil {
		return MessagePage{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	room, err := s.Room(ctx, roomID)
	if errors.Is(err, ErrNotFound) {
		return MessagePage{Messages: []Message{}}, nil
	}
	if err != nil {
		return MessagePage{}, err
	}

	if after >= room.LastSeq {
		return MessagePage{Messages: []Message{}}, nil
	}

	if cached, hasMore, ok := s.cache.after(roomID, after, limit); ok {
		if err := s.populateReactions(ctx, roomID, cached); err != nil {
			return MessagePage{}, err
		}
		return MessagePage{Messages: cached, HasMore: hasMore}, nil
	}

	lo := after + 1
	hi := min(room.LastSeq+1, lo+uint64(limit))

	watermark, err := s.archiveWatermark(roomID)
	if err != nil {
		return MessagePage{}, err
	}
	if s.remote == nil && lo <= watermark {
		return MessagePage{}, fmt.Errorf(
			"messages %d..%d are archived but no remote object store is configured",
			lo,
			min(hi, watermark+1),
		)
	}
	warmLo := lo
	if watermark >= warmLo {
		warmLo = watermark + 1
	}
	warm, err := s.scanWarm(roomID, warmLo, hi)
	if err != nil {
		return MessagePage{}, err
	}
	cold, err := s.readArchivedRange(ctx, roomID, lo, min(hi, watermark+1))
	if err != nil {
		return MessagePage{}, err
	}

	ascending := make([]Message, 0, len(warm)+len(cold))
	ascending = append(ascending, cold...)
	ascending = append(ascending, warm...)

	if err := s.populateReactions(ctx, roomID, ascending); err != nil {
		return MessagePage{}, err
	}

	hasMore := room.LastSeq >= hi
	return MessagePage{Messages: ascending, HasMore: hasMore}, nil
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

	capacity := min(int(hi-lo), 128)
	messages := make([]Message, 0, capacity)
	for iter.First(); iter.Valid(); iter.Next() {
		value := bytes.Clone(iter.Value())
		message, err := decodeMessage(value)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, iter.Error()
}

func messageUpperBoundBuf(buf, prefix []byte, before uint64) []byte {
	if before == 0 {
		return prefixUpperBoundBuf(buf, prefix)
	}
	buf = append(buf[:0], prefix...)
	var seq [8]byte
	binary.BigEndian.PutUint64(seq[:], before)
	return append(buf, seq[:]...)
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
	var upBuf [40]byte
	upperBound := messageUpperBoundBuf(upBuf[:0], prefix, before)
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
		msg, err := decodeMessage(iter.Value())
		if err != nil || msg.MsgType == MsgTypeRecall {
			continue
		}
		if !containsFold(msg.Payload, kwBytes) {
			continue
		}
		// Prev invalidates the iterator buffer: only matched messages pay
		// for copying their variable-length fields out.
		msg.Payload = append(msg.Payload[:0:0], msg.Payload...)
		msg.Ext = append(msg.Ext[:0:0], msg.Ext...)
		messages = append(messages, msg)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}
	if err := s.populateReactions(ctx, roomID, messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// containsFold reports whether needle occurs in haystack, comparing ASCII
// letters case-insensitively (multi-byte runes such as CJK text compare
// byte-exactly, which is what a substring match needs) and allocating nothing.
func containsFold(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	first := asciiLower(needle[0])
	for i, end := 0, len(haystack)-len(needle); i <= end; i++ {
		if asciiLower(haystack[i]) != first {
			continue
		}
		if bytes.EqualFold(haystack[i:i+len(needle)], needle) {
			return true
		}
	}
	return false
}

func asciiLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

func (s *Store) MessageByID(ctx context.Context, roomID, msgID uuid.UUID) (Message, error) {
	if err := contextErr(ctx); err != nil {
		return Message{}, err
	}
	seq, err := s.seqByMsgID(roomID, msgID)
	if err != nil {
		return Message{}, err
	}
	message, err := s.messageBySeq(ctx, roomID, seq)
	if err != nil {
		return Message{}, err
	}
	reactions, err := s.Reactions(ctx, roomID, msgID)
	if err != nil {
		return Message{}, err
	}
	message.Reactions = reactions
	return message, nil
}

func (s *Store) RecallMessage(
	ctx context.Context,
	roomID, msgID, operatorID uuid.UUID,
	isOwnerOrAdmin bool,
) (Message, error) {
	if err := contextErr(ctx); err != nil {
		return Message{}, err
	}
	s.stateMu.RLock()
	closed := s.closed
	s.stateMu.RUnlock()
	if closed {
		return Message{}, ErrClosed
	}
	request := &recallRequest{
		ctx:            ctx,
		roomID:         roomID,
		msgID:          msgID,
		operatorID:     operatorID,
		isOwnerOrAdmin: isOwnerOrAdmin,
		result:         make(chan RecallResult, 1),
	}
	select {
	case s.recallQueue <- request:
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
	result := <-request.result
	return result.Message, result.Err
}

// execRecall runs on the single writer goroutine: rewriting a recall also
// rewrites the room record, so it must not interleave with message batches.
func (s *Store) execRecall(request *recallRequest) {
	result := RecallResult{Err: contextErr(request.ctx)}
	if result.Err == nil {
		result.Message, result.Err = s.recall(request)
	}
	request.result <- result
}

func (s *Store) forgetDedup(roomID uuid.UUID, messages []Message) {
	for i := range messages {
		s.dedupCache.Delete(dedupLookupKey{room: roomID, sender: messages[i].SenderID, client: messages[i].ClientMsgID})
	}
}

// recallWindow bounds how long after ServerTime a non-admin may recall a message.
const recallWindow = 2 * time.Minute

func (s *Store) recall(request *recallRequest) (Message, error) {
	roomID, msgID, operatorID := request.roomID, request.msgID, request.operatorID

	seq, err := s.seqByMsgID(roomID, msgID)
	if err != nil {
		return Message{}, err
	}

	var msgKeyBuf [25]byte
	msgKey := appendMessageKey(msgKeyBuf[:0], roomID, seq)
	msg, err := s.getRecord(msgKey, decodeMessage)
	if errors.Is(err, ErrNotFound) {
		// The body is archived: it left the local store and cannot be
		// recalled anymore, no matter who asks.
		if _, remoteErr := s.remoteMessage(request.ctx, roomID, seq); remoteErr == nil {
			return Message{}, ErrRecallTimeout
		}
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, err
	}
	// The recalled message lands in the rooms LRU and the tail cache, so it
	// must not alias the pebble buffer.
	msg = ownedMessage(msg)

	if msg.MsgType == MsgTypeRecall {
		return msg, nil
	}

	if operatorID != msg.SenderID && !request.isOwnerOrAdmin {
		return Message{}, ErrForbidden
	}

	if !request.isOwnerOrAdmin && time.Since(time.UnixMicro(msg.ServerTime)) > recallWindow {
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
