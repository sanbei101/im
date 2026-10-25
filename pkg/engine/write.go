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

// messageBatch accumulates one writer-goroutine write: it allocates room
// seqs, resolves idempotency, and stages Pebble writes, then commits the
// batch atomically and publishes the committed state to the caches. All
// buffers belong to the single writer goroutine and need no locking.
type messageBatch struct {
	engine   *Engine
	batch    *pebble.Batch
	messages []Message
	results  []MessageWriteResult

	day      uint16
	lastSeqs map[uuid.UUID]uint64
	digests  [][32]byte
	// pending maps an idempotency key to the first in-batch message index
	// that staged it, so later duplicates replay or conflict against it.
	pending      map[dedupLookupKey]int
	allocated    []int
	dedupEntries []dedupEntry
	failedRooms  map[uuid.UUID]struct{}

	dedupKeyBuf [dedupKeyLen]byte
	roomKeyBuf  [roomKeyLen]byte
	msgKeyBuf   [messageKeyLen]byte
	recBuf      []byte
	roomBuf     []byte
}

// writeMessageBatch stages, commits, and publishes one write batch.
func (e *Engine) writeMessageBatch(messages []Message, results []MessageWriteResult) {
	mb := e.newMessageBatch(messages, results)
	defer mb.batch.Close()
	mb.stageMessages()
	mb.stageRoomHeads()
	if !mb.commit() {
		return
	}
	mb.publish()
}

func (e *Engine) newMessageBatch(messages []Message, results []MessageWriteResult) *messageBatch {
	return &messageBatch{
		engine:      e,
		batch:       e.db.NewBatch(),
		messages:    messages,
		results:     results,
		day:         dedupDay(time.Now()),
		lastSeqs:    make(map[uuid.UUID]uint64),
		digests:     make([][32]byte, len(messages)),
		pending:     make(map[dedupLookupKey]int, len(messages)),
		failedRooms: make(map[uuid.UUID]struct{}),
		recBuf:      e.scratch.record,
		roomBuf:     e.scratch.room,
	}
}

// stageMessages preprocesses and stages every not-yet-failed message: resolve
// idempotency, allocate the next room_seq, and put the message record plus
// its dedup credential into the batch.
func (mb *messageBatch) stageMessages() {
	for index := range mb.messages {
		if mb.results[index].Err == nil {
			mb.stageMessage(index)
		}
	}
}

func (mb *messageBatch) stageMessage(index int) {
	message := &mb.messages[index]
	mb.results[index].Message = *message
	lastSeq, err := mb.engine.loadLastSeq(message.RoomID, mb.lastSeqs)
	if err != nil {
		mb.results[index].Err = err
		return
	}
	mb.digests[index] = mb.engine.messageDigest(message)

	lookup := dedupLookupKey{room: message.RoomID, sender: message.SenderID, client: message.ClientMsgID}
	if previous, ok := mb.pending[lookup]; ok {
		mb.replayInBatch(index, previous)
		return
	}
	mb.pending[lookup] = index

	if old, ok := mb.engine.dedupCache.Get(lookup); ok {
		replayDedup(&mb.results[index], old, mb.digests[index])
		return
	}
	if old, found, err := mb.engine.findDedup(mb.day, lookup); err != nil {
		mb.results[index].Err = err
		return
	} else if found {
		replayDedup(&mb.results[index], old, mb.digests[index])
		return
	}

	mb.allocate(index, message, lookup, lastSeq)
}

// replayInBatch answers a same-batch duplicate against the first staged
// message carrying the same idempotency key.
func (mb *messageBatch) replayInBatch(index, previous int) {
	switch {
	case mb.digests[previous] != mb.digests[index]:
		mb.results[index].Err = ErrConflict
	case mb.results[previous].Err != nil:
		mb.results[index].Err = mb.results[previous].Err
	default:
		mb.results[index].Message = mb.results[previous].Message
	}
}

// allocate takes the next room_seq for a fresh message and stages its record
// and dedup credential into the batch.
func (mb *messageBatch) allocate(index int, message *Message, lookup dedupLookupKey, lastSeq uint64) {
	lastSeq++
	result := &mb.results[index]
	result.Message.RoomSeq = lastSeq
	result.Message.ServerTime = time.Now().UnixMicro()
	mb.lastSeqs[message.RoomID] = lastSeq
	mb.allocated = append(mb.allocated, index)

	msgKey := appendMessageKey(mb.msgKeyBuf[:0], message.RoomID, lastSeq)
	mb.recBuf = appendMessage(mb.recBuf[:0], result.Message)
	if err := mb.engine.setBytes(mb.batch, msgKey, mb.recBuf); err != nil {
		result.Err = err
		return
	}
	record := dedup{
		RoomSeq:    lastSeq,
		ServerTime: result.Message.ServerTime,
		PayloadSum: mb.digests[index],
	}
	dedupKey := appendDedupKey(mb.dedupKeyBuf[:0], mb.day, message.RoomID, message.SenderID, message.ClientMsgID)
	if err := mb.engine.setBytes(mb.batch, dedupKey, appendDedup(mb.engine.scratch.dedup[:], record)); err != nil {
		result.Err = errors.Join(fmt.Errorf("stage dedup: %w", err), unstage(mb.batch, msgKey))
		return
	}
	mb.dedupEntries = append(mb.dedupEntries, dedupEntry{lookup: lookup, dedup: record})
}

// stageRoomHeads rewrites the head of every room touched in this batch, so a
// later loadLastSeq sees the newly allocated seqs once the batch commits.
func (mb *messageBatch) stageRoomHeads() {
	for roomID, seq := range mb.lastSeqs {
		roomKey := appendRoomKey(mb.roomKeyBuf[:0], roomID)
		mb.roomBuf = appendRoomHead(mb.roomBuf[:0], seq)
		if err := mb.engine.setBytes(mb.batch, roomKey, mb.roomBuf); err != nil {
			mb.failRoom(roomID, err)
		}
	}
}

// failRoom fails every staged message of one room after its head write could
// not be staged.
func (mb *messageBatch) failRoom(roomID uuid.UUID, err error) {
	mb.failedRooms[roomID] = struct{}{}
	for index := range mb.results {
		if mb.results[index].Message.RoomID == roomID && mb.results[index].Err == nil {
			mb.results[index].Err = err
		}
	}
}

// commit commits the batch durably; on failure every not-yet-failed message
// is answered with the commit error and false is returned.
func (mb *messageBatch) commit() bool {
	if err := commit(mb.batch); err != nil {
		err = fmt.Errorf("commit messages: %w", err)
		for index := range mb.results {
			if mb.results[index].Err == nil {
				mb.results[index].Err = err
			}
		}
		return false
	}
	return true
}

// publish refreshes the in-memory state from a committed batch. Caches are
// filled only here, so a failed commit must not leave in-memory state ahead
// of disk, and every cached message is covered by a successful ACK.
func (mb *messageBatch) publish() {
	mb.engine.scratch.record, mb.engine.scratch.room = mb.recBuf, mb.roomBuf
	for roomID, seq := range mb.lastSeqs {
		if _, ok := mb.failedRooms[roomID]; !ok {
			mb.engine.getRoomShard(roomID).rooms.Set(roomID, seq)
		}
	}
	for _, entry := range mb.dedupEntries {
		mb.engine.dedupCache.Set(entry.lookup, entry.dedup)
	}
	for _, index := range mb.allocated {
		if mb.results[index].Err == nil {
			mb.engine.cache.append(&mb.results[index].Message)
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
