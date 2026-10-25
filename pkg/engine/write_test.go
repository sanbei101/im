package engine

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"
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
