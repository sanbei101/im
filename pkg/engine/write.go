package engine

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

type messageWriteRequest struct {
	ctx      context.Context
	messages []Message
	result   chan []MessageWriteResult
}

// MessageWriteResult pairs each submitted message with its committed form or
// the error that rejected it.
type MessageWriteResult struct {
	Message Message
	Err     error
}

// WriteMessage commits one message and returns its committed form. A retry
// with the same room, sender, and client_msg_id returns the original message;
// a changed payload returns ErrConflict.
func (e *Engine) WriteMessage(ctx context.Context, message Message) (Message, error) {
	results := e.WriteMessages(ctx, []Message{message})
	if len(results) == 0 {
		return Message{}, ErrClosed
	}
	return results[0].Message, results[0].Err
}

// WriteMessages submits messages to the writer goroutine. It returns one
// result per input in order; a closed engine fails every message with
// ErrClosed.
func (e *Engine) WriteMessages(ctx context.Context, messages []Message) []MessageWriteResult {
	if len(messages) == 0 {
		return nil
	}
	failed := func(err error) []MessageWriteResult {
		results := make([]MessageWriteResult, len(messages))
		for i := range results {
			results[i] = MessageWriteResult{Message: messages[i], Err: err}
		}
		return results
	}
	result := getResultChan()
	defer putResultChan(result)
	request := messageWriteRequest{ctx: ctx, messages: messages, result: result}
	e.stateMu.RLock()
	if e.closed {
		e.stateMu.RUnlock()
		return failed(ErrClosed)
	}
	select {
	case e.messageQueue <- request:
		e.stateMu.RUnlock()
	case <-ctx.Done():
		e.stateMu.RUnlock()
		return failed(ctx.Err())
	}
	return <-result
}

// runMessageWriter is the single writer goroutine: it drains both queues
// until Close, finishing every already-queued request before exiting.
func (e *Engine) runMessageWriter() {
	defer close(e.writerDone)
	for {
		select {
		case request := <-e.messageQueue:
			e.writeRequests(e.collectMessageRequests(request))
		case request := <-e.recallQueue:
			e.runRecall(request)
		case <-e.closeSignal:
			for {
				select {
				case request := <-e.messageQueue:
					e.writeRequests(e.collectMessageRequests(request))
				case request := <-e.recallQueue:
					e.runRecall(request)
				default:
					return
				}
			}
		}
	}
}

// writeRequests commits a collected batch. A panic in the commit path must
// not wedge in-flight callers forever, so every request still queued in the
// batch is answered with an error result.
func (e *Engine) writeRequests(requests []messageWriteRequest) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("message write failed: %v", r)
			for i := range requests {
				results := make([]MessageWriteResult, len(requests[i].messages))
				for j := range results {
					results[j] = MessageWriteResult{Message: requests[i].messages[j], Err: err}
				}
				requests[i].result <- results
			}
		}
	}()
	e.commitMessageRequests(requests)
}

// collectMessageRequests gathers queued requests into one write batch, bounded
// by messageBatchSize messages or messageBatchWindow of wait time, whichever
// comes first: many small random writes become one sequential batch commit.
func (e *Engine) collectMessageRequests(first messageWriteRequest) []messageWriteRequest {
	requests := []messageWriteRequest{first}
	messageCount := len(first.messages)
	window := time.NewTimer(messageBatchWindow)
	defer window.Stop()
	for messageCount < messageBatchSize {
		select {
		case request := <-e.messageQueue:
			requests = append(requests, request)
			messageCount += len(request.messages)
		case <-window.C:
			return requests
		}
	}
	return requests
}

func (e *Engine) commitMessageRequests(requests []messageWriteRequest) {
	total := 0
	for _, request := range requests {
		total += len(request.messages)
	}
	if total == 0 {
		return
	}
	// Single request: reuse the caller's slice, no copy.
	if len(requests) == 1 {
		request := &requests[0]
		results := make([]MessageWriteResult, len(request.messages))
		for index := range request.messages {
			results[index].Message = request.messages[index]
			if err := contextErr(request.ctx); err != nil {
				results[index].Err = err
			}
		}
		e.writeMessageBatch(request.messages, results)
		request.result <- results
		return
	}
	results := make([]MessageWriteResult, total)
	messages := make([]Message, total)
	offset := 0
	for i := range requests {
		request := &requests[i]
		for j := range request.messages {
			// Contexts that already died fail locally and leave a zero Message slot;
			// writeMessageBatch skips pre-errored entries, keeping the two slices aligned.
			if err := contextErr(request.ctx); err != nil {
				results[offset+j] = MessageWriteResult{Message: request.messages[j], Err: err}
				continue
			}
			messages[offset+j] = request.messages[j]
		}
		offset += len(request.messages)
	}
	e.writeMessageBatch(messages, results)
	offset = 0
	for i := range requests {
		requests[i].result <- results[offset : offset+len(requests[i].messages)]
		offset += len(requests[i].messages)
	}
}

// freeResultChans recycles per-request reply channels. Channels are put back
// only after their result has been received, so every handed-out channel is
// empty; a bounded free list avoids both the allocation and the type assertion.
var freeResultChans = make(chan chan []MessageWriteResult, 64)

func getResultChan() chan []MessageWriteResult {
	select {
	case channel := <-freeResultChans:
		return channel
	default:
		return make(chan []MessageWriteResult, 1)
	}
}

func putResultChan(channel chan []MessageWriteResult) {
	select {
	case freeResultChans <- channel:
	default: // free list full: let the collector reclaim the channel
	}
}

// recallRequest funnels RecallMessage through the single writer goroutine: a
// recall rewrites the room head, so it must serialize with message batches
// that allocate seqs and rewrite the same record.
type recallRequest struct {
	ctx            context.Context
	roomID         uuid.UUID
	seq            uint64
	operatorID     uuid.UUID
	isOwnerOrAdmin bool
	result         chan recallResult
}

// recallResult carries the outcome of a recall request back to its caller.
type recallResult struct {
	message Message
	err     error
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
	dedup  dedup
}

// writeScratch holds the writer goroutine's reusable encode buffers. Only the
// single writer goroutine touches it, so it needs no locking.
type writeScratch struct {
	record []byte
	room   []byte
	digest []byte
	dedup  [dedupRecordLen]byte
}

func (e *Engine) writeMessageBatch(
	messages []Message,
	results []MessageWriteResult,
) {
	batch := e.db.NewBatch()
	defer batch.Close()
	lastSeqs := make(map[uuid.UUID]uint64)
	digests := make([][32]byte, len(messages))
	pending := make(map[dedupLookupKey]int, len(messages))
	var allocated []int
	var dedupEntries []dedupEntry
	failedRooms := make(map[uuid.UUID]struct{})

	var dedupKeyBuf [dedupKeyLen]byte
	var roomKeyBuf [roomKeyLen]byte
	var msgKeyBuf [messageKeyLen]byte
	recBuf := e.scratch.record
	roomBuf := e.scratch.room
	day := dedupDay(time.Now())

	for index := range messages {
		if results[index].Err != nil {
			continue
		}
		message := &messages[index]
		results[index].Message = *message
		lastSeq, err := e.loadLastSeq(message.RoomID, lastSeqs)
		if err != nil {
			results[index].Err = err
			continue
		}
		digests[index] = e.messageDigest(message)

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
		pending[lookup] = index

		if old, ok := e.dedupCache.Get(lookup); ok {
			replayDedup(&results[index], old, digests[index])
			continue
		}
		if old, found, err := e.findDedup(day, lookup); err != nil {
			results[index].Err = err
			continue
		} else if found {
			replayDedup(&results[index], old, digests[index])
			continue
		}

		lastSeq++
		results[index].Message.RoomSeq = lastSeq
		results[index].Message.ServerTime = time.Now().UnixMicro()
		lastSeqs[message.RoomID] = lastSeq
		allocated = append(allocated, index)

		msgKey := appendMessageKey(msgKeyBuf[:0], message.RoomID, results[index].Message.RoomSeq)
		recBuf = appendMessage(recBuf[:0], results[index].Message)
		if err := e.setBytes(batch, msgKey, recBuf); err != nil {
			results[index].Err = err
			continue
		}
		record := dedup{
			RoomSeq:    results[index].Message.RoomSeq,
			ServerTime: results[index].Message.ServerTime, PayloadSum: digests[index],
		}
		dedupKeyBytes := appendDedupKey(dedupKeyBuf[:0], day, message.RoomID, message.SenderID, message.ClientMsgID)
		if err := e.setBytes(batch, dedupKeyBytes, appendDedup(e.scratch.dedup[:], record)); err != nil {
			results[index].Err = errors.Join(fmt.Errorf("stage dedup: %w", err), unstage(batch, msgKey))
			continue
		}
		dedupEntries = append(dedupEntries, dedupEntry{lookup: lookup, dedup: record})
	}
	for roomID := range lastSeqs {
		roomKeyBytes := appendRoomKey(roomKeyBuf[:0], roomID)
		roomBuf = appendRoomHead(roomBuf[:0], lastSeqs[roomID])
		if err := e.setBytes(batch, roomKeyBytes, roomBuf); err != nil {
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
	e.scratch.record, e.scratch.room = recBuf, roomBuf
	// Caches are filled only after the commit succeeded: a failed commit must
	// not leave in-memory state ahead of disk.
	for roomID := range lastSeqs {
		if _, ok := failedRooms[roomID]; ok {
			continue
		}
		e.getRoomShard(roomID).rooms.Set(roomID, lastSeqs[roomID])
	}
	for _, entry := range dedupEntries {
		e.dedupCache.Set(entry.lookup, entry.dedup)
	}
	// The cache is filled only after the commit succeeded, so a message in the
	// tail cache is always covered by a successful ACK.
	for _, index := range allocated {
		if results[index].Err == nil {
			e.cache.append(&results[index].Message)
		}
	}
}

// loadLastSeq resolves the room's last committed seq through the batch-local
// map, the shard cache, and Pebble, in that order. A missing head is a fresh
// room whose first message takes seq 1.
func (e *Engine) loadLastSeq(room uuid.UUID, lastSeqs map[uuid.UUID]uint64) (uint64, error) {
	if seq, ok := lastSeqs[room]; ok {
		return seq, nil
	}
	if seq, ok := e.getRoomShard(room).rooms.Get(room); ok {
		return seq, nil
	}
	var keyBuf [roomKeyLen]byte
	seq, err := e.getRecord(appendRoomKey(keyBuf[:0], room), decodeLastSeq)
	if errors.Is(err, ErrNotFound) {
		return 0, nil
	}
	return seq, err
}

// findDedup probes the dedup buckets a retry can live in; a hit refreshes
// the LRU cache.
func (e *Engine) findDedup(today uint16, lookup dedupLookupKey) (dedup, bool, error) {
	days, count := dedupProbeDays(lookup.client, today)
	var keyBuf [dedupKeyLen]byte
	for _, probe := range days[:count] {
		key := appendDedupKey(keyBuf[:0], probe, lookup.room, lookup.sender, lookup.client)
		record, err := e.getRecord(key, decodeDedup)
		if err == nil {
			e.dedupCache.Set(lookup, record)
			return record, true, nil
		}
		if !errors.Is(err, ErrNotFound) {
			return dedup{}, false, err
		}
	}
	return dedup{}, false, nil
}

// messageDigest digests the idempotency-relevant content: a retry must match
// on type, payload, ext, and reply target to replay. Runs on the single
// writer goroutine, so the scratch buffer needs no locking.
func (e *Engine) messageDigest(message *Message) [32]byte {
	input := e.scratch.digest[:0]
	input = append(input, byte(message.MsgType))
	input = append(input, message.Payload...)
	input = append(input, 0)
	input = append(input, message.Ext...)
	if message.ReplyToSeq != 0 {
		var seqBytes [8]byte
		putU64(seqBytes[:], message.ReplyToSeq)
		input = append(input, seqBytes[:]...)
	}
	e.scratch.digest = input
	return sha256.Sum256(input)
}

// replayDedup answers a retry: identical content replays the original
// committed identity, changed content is a conflict.
func replayDedup(result *MessageWriteResult, old dedup, digest [32]byte) {
	if old.PayloadSum != digest {
		result.Err = ErrConflict
		return
	}
	result.Message.RoomSeq = old.RoomSeq
	result.Message.ServerTime = old.ServerTime
}

// unstage removes keys already staged in this batch after a staging failure,
// so a half-written message cannot commit.
func unstage(batch *pebble.Batch, keys ...[]byte) error {
	var errs []error
	for _, key := range keys {
		if err := batch.Delete(key, nil); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
