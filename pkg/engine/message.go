package engine

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func (e *Engine) Messages(ctx context.Context, roomID uuid.UUID, before uint64, limit int) (MessagePage, error) {
	if err := contextErr(ctx); err != nil {
		return MessagePage{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if messages, hasMore, ok := e.cache.page(roomID, before, limit); ok {
		if err := e.populateReactions(ctx, roomID, messages); err != nil {
			return MessagePage{}, err
		}
		return MessagePage{Messages: messages, HasMore: hasMore}, nil
	}

	// The history window [lo, hi) spans at most limit+1 seqs. hi needs the
	// room's LastSeq only for the newest page; a missing room stays an empty
	// page, as the plain Pebble scan of a missing prefix would be.
	hi := before
	if before == 0 {
		lastSeq, err := e.LastSeq(ctx, roomID)
		if errors.Is(err, ErrNotFound) {
			return MessagePage{Messages: []Message{}}, nil
		}
		if err != nil {
			return MessagePage{}, err
		}
		hi = lastSeq + 1
	}
	lo := uint64(1)
	if hi > uint64(limit)+1 {
		lo = hi - uint64(limit+1)
	}

	ascending, err := e.readAscending(ctx, roomID, lo, hi)
	if err != nil {
		return MessagePage{}, err
	}

	total := len(ascending)
	if before == 0 && total > 0 {
		e.cache.put(roomID, ascending, hi-1)
	}
	// The page is the newest n messages, newest first: reverse that tail of
	// the ascending slice in place instead of allocating a second page.
	messages := ascending[max(total-limit, 0):]
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	if err := e.populateReactions(ctx, roomID, messages); err != nil {
		return MessagePage{}, err
	}
	return MessagePage{Messages: messages, HasMore: total > limit}, nil
}

// MessagesAfter returns messages with room_seq in (after, min(after+limit, LastSeq)] in ascending order.
func (e *Engine) MessagesAfter(ctx context.Context, roomID uuid.UUID, after uint64, limit int) (MessagePage, error) {
	if err := contextErr(ctx); err != nil {
		return MessagePage{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	lastSeq, err := e.LastSeq(ctx, roomID)
	if errors.Is(err, ErrNotFound) {
		return MessagePage{Messages: []Message{}}, nil
	}
	if err != nil {
		return MessagePage{}, err
	}

	if after >= lastSeq {
		return MessagePage{Messages: []Message{}}, nil
	}

	if cached, hasMore, ok := e.cache.after(roomID, after, limit); ok {
		if err := e.populateReactions(ctx, roomID, cached); err != nil {
			return MessagePage{}, err
		}
		return MessagePage{Messages: cached, HasMore: hasMore}, nil
	}

	lo := after + 1
	hi := min(lastSeq+1, lo+uint64(limit))

	ascending, err := e.readAscending(ctx, roomID, lo, hi)
	if err != nil {
		return MessagePage{}, err
	}
	if err := e.populateReactions(ctx, roomID, ascending); err != nil {
		return MessagePage{}, err
	}
	hasMore := lastSeq >= hi
	return MessagePage{Messages: ascending, HasMore: hasMore}, nil
}

// readAscending returns the messages with seq in [lo, hi) across both tiers
// in ascending order, cold rows first. An unconfigured archive directory
// turns an archived range into an error instead of silently lost history.
func (e *Engine) readAscending(ctx context.Context, room uuid.UUID, lo, hi uint64) ([]Message, error) {
	watermark, err := e.archiveWatermark(room)
	if err != nil {
		return nil, err
	}
	if e.archiveDir == "" && lo <= watermark {
		return nil, fmt.Errorf(
			"messages %d..%d are archived but no archive directory is configured",
			lo,
			min(hi, watermark+1),
		)
	}
	warmLo := lo
	if watermark >= warmLo {
		warmLo = watermark + 1
	}
	warm, err := e.scanWarm(room, warmLo, hi)
	if err != nil {
		return nil, err
	}
	cold, err := e.readArchivedRange(ctx, room, lo, min(hi, watermark+1))
	if err != nil {
		return nil, err
	}
	ascending := make([]Message, 0, len(warm)+len(cold))
	ascending = append(ascending, cold...)
	return append(ascending, warm...), nil
}

// scanWarm returns the Pebble messages with seq in [lo, hi) in ascending order.
func (e *Engine) scanWarm(roomID uuid.UUID, lo, hi uint64) ([]Message, error) {
	if hi <= lo {
		return nil, nil
	}
	var lowerBuf, upperBuf [messageKeyLen]byte
	iter, err := e.db.NewIter(&pebble.IterOptions{
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

func (e *Engine) SearchRoomMessages(
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

	var prefixBuf [messagePrefixLen]byte
	prefix := appendMessagePrefix(prefixBuf[:0], roomID)
	var upBuf [messageKeyLen + 1]byte
	upperBound := messageUpperBoundBuf(upBuf[:0], prefix, before)
	iter, err := e.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
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
	if err := e.populateReactions(ctx, roomID, messages); err != nil {
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

func (e *Engine) RecallMessage(
	ctx context.Context,
	roomID uuid.UUID,
	seq uint64,
	operatorID uuid.UUID,
	isOwnerOrAdmin bool,
) (Message, error) {
	if err := contextErr(ctx); err != nil {
		return Message{}, err
	}
	e.stateMu.RLock()
	closed := e.closed
	e.stateMu.RUnlock()
	if closed {
		return Message{}, ErrClosed
	}
	request := &recallRequest{
		ctx:            ctx,
		roomID:         roomID,
		seq:            seq,
		operatorID:     operatorID,
		isOwnerOrAdmin: isOwnerOrAdmin,
		result:         make(chan recallResult, 1),
	}
	select {
	case e.recallQueue <- request:
	case <-ctx.Done():
		return Message{}, ctx.Err()
	}
	result := <-request.result
	return result.message, result.err
}

// runRecall wraps execRecall with a panic guarantee: the caller is answered
// with an error instead of blocking forever.
func (e *Engine) runRecall(request *recallRequest) {
	defer func() {
		if r := recover(); r != nil {
			request.result <- recallResult{err: fmt.Errorf("recall failed: %v", r)}
		}
	}()
	e.execRecall(request)
}

// execRecall runs on the single writer goroutine: a recall also rewrites the
// room head, so it must not interleave with message batches.
func (e *Engine) execRecall(request *recallRequest) {
	result := recallResult{err: contextErr(request.ctx)}
	if result.err == nil {
		result.message, result.err = e.recall(request)
	}
	request.result <- result
}

func (e *Engine) recall(request *recallRequest) (Message, error) {
	roomID, seq, operatorID := request.roomID, request.seq, request.operatorID

	var msgKeyBuf [messageKeyLen]byte
	msgKey := appendMessageKey(msgKeyBuf[:0], roomID, seq)
	msg, err := e.getRecord(msgKey, decodeMessage)
	if errors.Is(err, ErrNotFound) {
		// The body is archived: it left the local tier and cannot be
		// recalled anymore, no matter who asks.
		if _, remoteErr := e.archivedMessage(request.ctx, roomID, seq); remoteErr == nil {
			return Message{}, ErrRecallTimeout
		}
		return Message{}, ErrNotFound
	}
	if err != nil {
		return Message{}, err
	}
	// The recalled message lands in the tail cache, so it
	// must not alias the pebble buffer.
	msg = ownedMessage(msg)

	if msg.MsgType == MsgTypeRecall {
		return msg, nil
	}

	if operatorID != msg.SenderID && !request.isOwnerOrAdmin {
		return Message{}, ErrForbidden
	}

	if !request.isOwnerOrAdmin && time.Since(time.UnixMicro(msg.ServerTime)) > e.recallWindow {
		return Message{}, ErrRecallTimeout
	}

	msg.MsgType = MsgTypeRecall
	msg.Payload = jsontext.Value(fmt.Sprintf(`{"recalled_by":"%s"}`, operatorID))

	batch := e.db.NewBatch()
	defer batch.Close()
	if err := e.setBytes(batch, msgKey, encodeMessage(msg)); err != nil {
		return Message{}, err
	}
	if err := commit(batch); err != nil {
		return Message{}, err
	}
	// A message inside the recall window is always in the tail cache; keep the
	// cached copy in sync so reads do not serve stale content.
	e.cache.replace(roomID, &msg)
	return msg, nil
}
