package engine

import (
	"context"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"testing"
	"time"
	"uuid"
)

// idempotentMessage builds the message every subtest retries: same room,
// sender, and client id, with the payload as the only variable.
func idempotentMessage(room, sender, client uuid.UUID, payload string) Message {
	return Message{
		ClientMsgID: client,
		SenderID:    sender,
		RoomID:      room,
		MsgType:     MsgTypeText,
		Payload:     jsontext.Value(`{"text":"` + payload + `"}`),
	}
}

// v7WithMillis builds a v7 client id carrying the given unix millisecond
// timestamp and a distinct counter in the random bits, the same RFC 9562
// layout the dedup paths decode.
func v7WithMillis(ms int64, distinct uint16) uuid.UUID {
	var id uuid.UUID
	binary.BigEndian.PutUint64(id[:8], uint64(ms)<<16)
	id[6] = (id[6] & 0x0f) | 0x70 // version 7
	id[8] = (id[8] & 0x3f) | 0x80 // RFC variant
	binary.BigEndian.PutUint16(id[9:11], distinct)
	return id
}

// burstMessages builds n distinct same-millisecond v7 messages, ids numbered
// from first so an id and its payload stay paired across calls.
func burstMessages(room, sender uuid.UUID, ms int64, first, n int) []Message {
	messages := make([]Message, n)
	for i := range messages {
		distinct := uint16(first + i)
		messages[i] = idempotentMessage(
			room, sender, v7WithMillis(ms, distinct), fmt.Sprintf("burst-%d", distinct))
	}
	return messages
}

func TestEngineIdempotency(t *testing.T) {
	ctx := context.Background()

	t.Run("same-batch replay", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		results := e.WriteMessages(ctx, []Message{
			idempotentMessage(room, sender, client, "once"),
			idempotentMessage(room, sender, client, "once"),
		})
		if len(results) != 2 {
			t.Fatalf("results = %d, want 2", len(results))
		}
		if results[0].Err != nil || results[1].Err != nil {
			t.Fatalf("duplicate in one batch: errs = [%v %v]", results[0].Err, results[1].Err)
		}
		if results[1].Message.RoomSeq != results[0].Message.RoomSeq {
			t.Fatalf("second copy got its own seq: %d vs %d",
				results[1].Message.RoomSeq, results[0].Message.RoomSeq)
		}
	})

	t.Run("same-batch conflict", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		results := e.WriteMessages(ctx, []Message{
			idempotentMessage(room, sender, client, "first"),
			idempotentMessage(room, sender, client, "different"),
		})
		if results[0].Err != nil {
			t.Fatalf("original message rejected: %v", results[0].Err)
		}
		if !errors.Is(results[1].Err, ErrConflict) {
			t.Fatalf("changed retry err = %v, want %v", results[1].Err, ErrConflict)
		}
		if results[1].Message.RoomSeq != 0 {
			t.Fatalf("conflicting copy must not be assigned a seq, got %d", results[1].Message.RoomSeq)
		}
	})

	t.Run("cross-batch replay", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		original, err := e.WriteMessage(ctx, idempotentMessage(room, sender, client, "retry me"))
		if err != nil {
			t.Fatal(err)
		}
		// The first call has returned, so its commit and cache fill are done:
		// this retry must resolve through the dedup cache.
		replayed, err := e.WriteMessage(ctx, idempotentMessage(room, sender, client, "retry me"))
		if err != nil {
			t.Fatalf("retry: %v", err)
		}
		if replayed.RoomSeq != original.RoomSeq || replayed.ServerTime != original.ServerTime {
			t.Fatalf("retry got new identity: %+v, want %+v", replayed, original)
		}
	})

	t.Run("cross-batch conflict", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		if _, err := e.WriteMessage(ctx, idempotentMessage(room, sender, client, "original")); err != nil {
			t.Fatal(err)
		}
		_, err := e.WriteMessage(ctx, idempotentMessage(room, sender, client, "rewritten"))
		if !errors.Is(err, ErrConflict) {
			t.Fatalf("changed cross-batch retry err = %v, want %v", err, ErrConflict)
		}
	})

	t.Run("replay survives a restart", func(t *testing.T) {
		// The in-memory dedup cache is gone after a restart; the retry must
		// resolve through the Pebble dedup record.
		path := t.TempDir() + "/engine"
		e := openTestEngine(t, path)
		room, sender, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		original, err := e.WriteMessage(ctx, idempotentMessage(room, sender, client, "across restarts"))
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}

		reopened := openTestEngine(t, path)
		replayed, err := reopened.WriteMessage(ctx, idempotentMessage(room, sender, client, "across restarts"))
		if err != nil {
			t.Fatalf("retry after restart: %v", err)
		}
		if replayed.RoomSeq != original.RoomSeq || replayed.ServerTime != original.ServerTime {
			t.Fatalf("retry after restart got new identity: %+v, want %+v", replayed, original)
		}
	})

	t.Run("freshness watermark never skips older retries", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender := uuid.NewV7(), uuid.NewV7()
		oldID := v7WithMillis(time.Now().UnixMilli()-10_000, 1)
		original, err := e.WriteMessage(ctx, idempotentMessage(room, sender, oldID, "old"))
		if err != nil {
			t.Fatal(err)
		}
		// A newer fresh write advances the dedup watermark past the old id.
		fresh, err := e.WriteMessage(ctx, idempotentMessage(
			room, sender, v7WithMillis(time.Now().UnixMilli(), 2), "new"))
		if err != nil {
			t.Fatal(err)
		}
		if fresh.RoomSeq == original.RoomSeq {
			t.Fatal("fresh write replayed the old message")
		}
		// Drop the old entry from the dedup cache: only a Pebble probe can
		// still answer the retry, and the guard must let it through.
		e.dedupCache.Delete(dedupLookupKey{room: room, sender: sender, client: oldID})
		replayed, err := e.WriteMessage(ctx, idempotentMessage(room, sender, oldID, "old"))
		if err != nil {
			t.Fatalf("older retry: %v", err)
		}
		if replayed.RoomSeq != original.RoomSeq || replayed.ServerTime != original.ServerTime {
			t.Fatalf("older retry got new identity: %+v, want %+v", replayed, original)
		}
	})

	t.Run("restart watermark does not mask pre-restart retries", func(t *testing.T) {
		path := t.TempDir() + "/engine"
		e := openTestEngine(t, path)
		room, sender := uuid.NewV7(), uuid.NewV7()
		oldID := v7WithMillis(time.Now().UnixMilli()-10_000, 1)
		original, err := e.WriteMessage(ctx, idempotentMessage(room, sender, oldID, "persisted"))
		if err != nil {
			t.Fatal(err)
		}
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}

		reopened := openTestEngine(t, path)
		// A fresh write after the restart advances the watermark again; the
		// retry of the older pre-restart id must still resolve via Pebble.
		if _, err := reopened.WriteMessage(ctx, idempotentMessage(
			room, sender, v7WithMillis(time.Now().UnixMilli(), 2), "after restart")); err != nil {
			t.Fatal(err)
		}
		replayed, err := reopened.WriteMessage(ctx, idempotentMessage(room, sender, oldID, "persisted"))
		if err != nil {
			t.Fatalf("pre-restart retry: %v", err)
		}
		if replayed.RoomSeq != original.RoomSeq || replayed.ServerTime != original.ServerTime {
			t.Fatalf("pre-restart retry got new identity: %+v, want %+v", replayed, original)
		}
	})

	t.Run("same-millisecond burst stays fresh, saturation re-probes", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender := uuid.NewV7(), uuid.NewV7()
		ms := time.Now().UnixMilli()

		// Under the cache capacity: same-millisecond first sends are fresh.
		results := e.WriteMessages(ctx, burstMessages(room, sender, ms, 1, dedupCacheSize-100))
		for i, result := range results {
			if result.Err != nil {
				t.Fatalf("burst message %d: %v", i, result.Err)
			}
		}
		if !e.dedupFresh(v7WithMillis(ms, 60000), ms) {
			t.Fatal("same-millisecond id below the cache capacity should be fresh")
		}

		// Past the capacity the streak saturates and probes resume; the
		// first burst message, long evicted from the cache, must still
		// replay through Pebble.
		rest := e.WriteMessages(ctx, burstMessages(room, sender, ms, dedupCacheSize-99, 200))
		for i, result := range rest {
			if result.Err != nil {
				t.Fatalf("saturated burst message %d: %v", i, result.Err)
			}
		}
		if e.dedupFresh(v7WithMillis(ms, 60001), ms) {
			t.Fatal("saturated streak must fall back to probing")
		}
		replayed, err := e.WriteMessage(ctx, idempotentMessage(
			room, sender, v7WithMillis(ms, 1), "burst-1"))
		if err != nil {
			t.Fatalf("saturated retry: %v", err)
		}
		if replayed.RoomSeq != results[0].Message.RoomSeq {
			t.Fatalf("saturated retry got seq %d, want %d",
				replayed.RoomSeq, results[0].Message.RoomSeq)
		}
	})

	t.Run("tampered future ids cannot poison the watermark", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender := uuid.NewV7(), uuid.NewV7()
		now := time.Now().UnixMilli()

		// An id from beyond the skew window stages normally but must not
		// move the watermark.
		futureID := v7WithMillis(now+3600_000, 1)
		original, err := e.WriteMessage(ctx, idempotentMessage(room, sender, futureID, "future"))
		if err != nil {
			t.Fatal(err)
		}
		if e.dedupMarkDay != 0 {
			t.Fatal("tampered id advanced the watermark")
		}

		// Honest traffic keeps the fast path alive right after the tampered
		// write instead of probing for the rest of the day.
		if _, err := e.WriteMessage(ctx, idempotentMessage(
			room, sender, v7WithMillis(now, 2), "honest")); err != nil {
			t.Fatal(err)
		}
		if !e.dedupFresh(v7WithMillis(now+1, 3), now+1) {
			t.Fatal("guard should stay healthy after a tampered id")
		}

		// The tampered message's own retry must still replay: beyond the
		// skew window an id is never fresh, so the Pebble probe runs.
		e.dedupCache.Delete(dedupLookupKey{room: room, sender: sender, client: futureID})
		replayed, err := e.WriteMessage(ctx, idempotentMessage(room, sender, futureID, "future"))
		if err != nil {
			t.Fatalf("tampered retry: %v", err)
		}
		if replayed.RoomSeq != original.RoomSeq || replayed.ServerTime != original.ServerTime {
			t.Fatalf("tampered retry got new identity: %+v, want %+v", replayed, original)
		}
	})

	t.Run("same client id from another sender is a fresh message", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender, other, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
		first, err := e.WriteMessage(ctx, idempotentMessage(room, sender, client, "mine"))
		if err != nil {
			t.Fatal(err)
		}
		second, err := e.WriteMessage(ctx, idempotentMessage(room, other, client, "mine"))
		if err != nil {
			t.Fatalf("same client id for another sender: %v", err)
		}
		if second.RoomSeq != first.RoomSeq+1 {
			t.Fatalf("second sender's message seq = %d, want %d", second.RoomSeq, first.RoomSeq+1)
		}
	})

	t.Run("cancelled context rejects without writing", func(t *testing.T) {
		e := newTestEngine(t)
		room, sender := uuid.NewV7(), uuid.NewV7()
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := e.WriteMessage(
			cancelled,
			idempotentMessage(room, sender, uuid.NewV7(), "never"),
		); !errors.Is(
			err,
			context.Canceled,
		) {
			t.Fatalf("cancelled write err = %v, want %v", err, context.Canceled)
		}
		page, err := e.Messages(ctx, room, 0, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Messages) != 0 {
			t.Fatalf("cancelled write left %d messages behind", len(page.Messages))
		}
	})

	t.Run("closed engine fails every message", func(t *testing.T) {
		e := newTestEngine(t)
		if err := e.Close(); err != nil {
			t.Fatal(err)
		}
		results := e.WriteMessages(ctx, []Message{
			idempotentMessage(uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), "a"),
			idempotentMessage(uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), "b"),
		})
		for i, result := range results {
			if !errors.Is(result.Err, ErrClosed) {
				t.Fatalf("result %d err = %v, want %v", i, result.Err, ErrClosed)
			}
		}
	})

	t.Run("empty batch returns nothing", func(t *testing.T) {
		e := newTestEngine(t)
		if results := e.WriteMessages(ctx, nil); results != nil {
			t.Fatalf("empty batch = %+v, want nil", results)
		}
	})
}

func BenchmarkEngineIdempotentReplay(b *testing.B) {
	e := newTestEngine(b)
	ctx := context.Background()
	room, sender, client := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	retry := idempotentMessage(room, sender, client, "retry me")
	original, err := e.WriteMessage(ctx, retry)
	if err != nil {
		b.Fatal(err)
	}

	b.Run("fresh-write", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			fresh := retry
			fresh.ClientMsgID = uuid.NewV7()
			if _, err := e.WriteMessage(ctx, fresh); err != nil {
				b.Fatal(err)
			}
		}
	})

	// The dedup-cache hit a network retry takes: no Pebble read, no new seq.
	b.Run("cache-hit-replay", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			replayed, err := e.WriteMessage(ctx, retry)
			if err != nil || replayed.RoomSeq != original.RoomSeq {
				b.Fatalf("replayed=%+v err=%v", replayed, err)
			}
		}
	})
}

// BenchmarkEngineBatchWrite moves the same 32 messages per iteration through
// the two caller shapes: 32 single-message calls versus one batched call.
// The ratio is what the writer's coalescing window buys.
func BenchmarkEngineBatchWrite(b *testing.B) {
	e := newTestEngine(b)
	ctx := context.Background()
	room, sender := benchIDs()
	const perOp = 32
	payload := jsontext.Value(`{"text":"batched"}`)

	b.Run("single-message-calls", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			for range perOp {
				_, err := e.WriteMessage(ctx, Message{
					ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
					MsgType: MsgTypeText, Payload: payload,
				})
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("one-batched-call", func(b *testing.B) {
		messages := make([]Message, perOp)
		for i := range messages {
			messages[i] = Message{SenderID: sender, RoomID: room, MsgType: MsgTypeText, Payload: payload}
		}
		b.ReportAllocs()
		for b.Loop() {
			for i := range messages {
				messages[i].ClientMsgID = uuid.NewV7()
			}
			results := e.WriteMessages(ctx, messages)
			if len(results) != perOp {
				b.Fatalf("results = %d, want %d", len(results), perOp)
			}
			for i := range results {
				if results[i].Err != nil {
					b.Fatal(results[i].Err)
				}
			}
		}
	})
}
