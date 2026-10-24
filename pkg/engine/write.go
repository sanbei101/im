package engine

import (
	"context"
	"fmt"
	"time"
	"uuid"
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
	msgID          uuid.UUID
	operatorID     uuid.UUID
	isOwnerOrAdmin bool
	result         chan recallResult
}

// recallResult carries the outcome of a recall request back to its caller.
type recallResult struct {
	message Message
	err     error
}
