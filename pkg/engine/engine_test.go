package engine

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/phuslu/log"
)

// openTestEngine opens an engine at path and registers its cleanup.
func openTestEngine(tb testing.TB, path string, opts ...Option) *Engine {
	tb.Helper()
	engine, err := Open(path, opts...)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := engine.Close(); err != nil {
			tb.Errorf("close engine: %v", err)
		}
	})
	return engine
}

// newTestEngine opens a throwaway engine and registers its cleanup.
func newTestEngine(tb testing.TB, opts ...Option) *Engine {
	tb.Helper()
	return openTestEngine(tb, tb.TempDir()+"/engine", opts...)
}

func mustWriteMessage(t *testing.T, e *Engine, roomID, senderID uuid.UUID, payload string) Message {
	t.Helper()
	message, err := e.WriteMessage(context.Background(), Message{
		ClientMsgID: uuid.NewV7(),
		SenderID:    senderID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     jsontext.Value(`{"text":"` + payload + `"}`),
	})
	if err != nil {
		t.Fatalf("write message: %v", err)
	}
	return message
}

// messagesEqual compares Messages field by field; the struct itself is not
// comparable because Payload and Ext are slices.
func messagesEqual(a, b Message) bool {
	return a.MsgID == b.MsgID && a.ClientMsgID == b.ClientMsgID && a.SenderID == b.SenderID &&
		a.RoomID == b.RoomID && a.RoomSeq == b.RoomSeq && a.ServerTime == b.ServerTime &&
		a.ReplyToMsgID == b.ReplyToMsgID && a.MsgType == b.MsgType &&
		string(a.Payload) == string(b.Payload) && string(a.Ext) == string(b.Ext)
}

func TestMessageCodec(t *testing.T) {
	t.Run("round trip full record", func(t *testing.T) {
		want := Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 10, ServerTime: 456, ReplyToMsgID: uuid.NewV7(),
			MsgType: MsgTypeFile, Payload: jsontext.Value(`{"u":"x"}`), Ext: jsontext.Value(`{"e":1}`),
		}
		got, err := decodeMessage(encodeMessage(want))
		if err != nil {
			t.Fatal(err)
		}
		if !messagesEqual(got, want) {
			t.Fatalf("round trip mismatch:\nwant %+v\ngot  %+v", want, got)
		}
	})
	t.Run("round trip minimal record", func(t *testing.T) {
		want := Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 1, ServerTime: -1, MsgType: MsgTypeSystem, Payload: jsontext.Value(`{}`),
		}
		got, err := decodeMessage(encodeMessage(want))
		if err != nil {
			t.Fatal(err)
		}
		if !messagesEqual(got, want) {
			t.Fatalf("round trip mismatch:\nwant %+v\ngot  %+v", want, got)
		}
	})
	t.Run("truncated records rejected", func(t *testing.T) {
		full := encodeMessage(Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 1, ServerTime: 1, MsgType: MsgTypeText, Payload: jsontext.Value(`{}`),
		})
		for _, broken := range [][]byte{{}, full[:len(full)/2], append(full, 0)} {
			if _, err := decodeMessage(broken); err == nil {
				t.Fatalf("expected error for %d-byte record", len(broken))
			}
		}
	})
	t.Run("dedup round trip", func(t *testing.T) {
		want := dedup{MsgID: uuid.NewV7(), RoomSeq: 7, ServerTime: 42, PayloadSum: [32]byte{1, 2, 3}}
		got, err := decodeDedup(appendDedup(nil, want))
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("dedup round trip mismatch: want %+v got %+v", want, got)
		}
		if _, err := decodeDedup(want.PayloadSum[:16]); err == nil {
			t.Fatal("expected error for short dedup record")
		}
	})
}

func TestMsgTypeCodec(t *testing.T) {
	for _, name := range []string{"text", "image", "video", "file", "system", "recall"} {
		parsed, err := ParseMsgType(name)
		if err != nil || !parsed.Valid() || parsed.String() != name {
			t.Fatalf("msg type %q: parsed=%v err=%v", name, parsed, err)
		}
	}
	if _, err := ParseMsgType("sticker"); err == nil {
		t.Fatal("expected error for unknown msg type")
	}
	// The wire form is the string, on every engine boundary.
	encoded, err := json.Marshal(MsgTypeImage)
	if err != nil || string(encoded) != `"image"` {
		t.Fatalf("marshal msg type: %s err=%v", encoded, err)
	}
	var decoded MsgType
	if err := json.Unmarshal([]byte(`"recall"`), &decoded); err != nil || decoded != MsgTypeRecall {
		t.Fatalf("unmarshal msg type: %v err=%v", decoded, err)
	}
}

// TestEngineOptions pins the guards: a zero or nil option value keeps the
// production default instead of degrading the engine.
func TestEngineOptions(t *testing.T) {
	e := newTestEngine(t, RecallWindow(0), Logger(nil))
	if e.recallWindow != defaultRecallWindow {
		t.Fatalf("recall window = %v, want default %v", e.recallWindow, defaultRecallWindow)
	}
	if e.logger != &log.DefaultLogger {
		t.Fatal("nil logger must keep the default logger")
	}
	custom := newTestEngine(t, RecallWindow(time.Hour))
	if custom.recallWindow != time.Hour {
		t.Fatalf("recall window = %v, want 1h", custom.recallWindow)
	}
}

// TestEngineLifecycle covers the two operational endpoints: Ping as the
// health probe and Checkpoint as the backup primitive — a checkpoint of a
// live engine is a complete, readable store.
func TestEngineLifecycle(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()
	if err := e.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	room, sender := uuid.NewV7(), uuid.NewV7()
	mustWriteMessage(t, e, room, sender, "checkpointed")

	snapshot := t.TempDir() + "/snapshot"
	if err := e.Checkpoint(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	restored := openTestEngine(t, snapshot)
	defer restored.Close()
	if err := restored.Ping(ctx); err != nil {
		t.Fatal(err)
	}
	page, err := restored.Messages(ctx, room, 0, 20)
	if err != nil || len(page.Messages) != 1 || page.Messages[0].RoomSeq != 1 {
		t.Fatalf("restored store: page=%+v err=%v", page, err)
	}
	// The restored store continues the sequence from the snapshot.
	next := mustWriteMessage(t, restored, room, sender, "after restore")
	if next.RoomSeq != 2 {
		t.Fatalf("seq after restore = %d, want 2", next.RoomSeq)
	}

	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if err := e.Ping(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("ping after close = %v, want %v", err, ErrClosed)
	}
}

func TestEngineMessages(t *testing.T) {
	e := newTestEngine(t)
	ctx := context.Background()
	sender := uuid.NewV7()
	room := uuid.NewV7()

	t.Run("write assigns identity", func(t *testing.T) {
		msg := mustWriteMessage(t, e, room, sender, "first")
		if msg.RoomSeq != 1 || msg.MsgID == uuid.Nil() || msg.ServerTime == 0 {
			t.Fatalf("server did not stamp message: %+v", msg)
		}
		lastSeq, err := e.LastSeq(ctx, room)
		if err != nil || lastSeq != 1 {
			t.Fatalf("room last seq = %d, want 1; err=%v", lastSeq, err)
		}
		// The head is created lazily by the first committed message: writing
		// to a room Pebble has never seen simply starts it at seq 1.
		freshRoom := uuid.NewV7()
		if _, err := e.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: freshRoom, MsgType: MsgTypeText,
		}); err != nil {
			t.Fatalf("write to fresh room: %v", err)
		}
	})

	t.Run("idempotent retry and conflict", func(t *testing.T) {
		clientID := uuid.NewV7()
		first, err := e.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender, RoomID: room,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"same"}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		retry, err := e.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender, RoomID: room,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"same"}`),
		})
		if err != nil || retry.MsgID != first.MsgID || retry.RoomSeq != first.RoomSeq {
			t.Fatalf("retry changed result: first=%+v retry=%+v err=%v", first, retry, err)
		}
		if _, err := e.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender, RoomID: room,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"changed"}`),
		}); !errors.Is(err, ErrConflict) {
			t.Fatalf("changed payload error = %v, want %v", err, ErrConflict)
		}
	})

	t.Run("batch dedup within one commit", func(t *testing.T) {
		clientID := uuid.NewV7()
		results := e.WriteMessages(ctx, []Message{
			{
				ClientMsgID: clientID,
				SenderID:    sender,
				RoomID:      room,
				MsgType:     MsgTypeText,
				Payload:     []byte("dup"),
			},
			{
				ClientMsgID: clientID,
				SenderID:    sender,
				RoomID:      room,
				MsgType:     MsgTypeText,
				Payload:     []byte("dup"),
			},
			{
				ClientMsgID: uuid.NewV7(),
				SenderID:    sender,
				RoomID:      room,
				MsgType:     MsgTypeText,
				Payload:     []byte("other"),
			},
		})
		if len(results) != 3 {
			t.Fatalf("batch returned %d results", len(results))
		}
		for i := range results {
			if results[i].Err != nil {
				t.Fatalf("batch result %d: %v", i, results[i].Err)
			}
		}
		if results[0].Message.MsgID != results[1].Message.MsgID {
			t.Fatalf("duplicate batch entries diverged: %+v vs %+v", results[0], results[1])
		}
		if results[2].Message.RoomSeq != results[0].Message.RoomSeq+1 {
			t.Fatalf("distinct batch message shares seq with duplicate: %+v", results[2])
		}
	})

	t.Run("pagination newest first with cursor", func(t *testing.T) {
		paged := uuid.NewV7()
		for i := range 7 {
			mustWriteMessage(t, e, paged, sender, "m"+string(rune('a'+i)))
		}

		newest, err := e.Messages(ctx, paged, 0, 3)
		if err != nil || len(newest.Messages) != 3 || !newest.HasMore {
			t.Fatalf("newest page: page=%+v err=%v", newest, err)
		}
		if newest.Messages[0].RoomSeq != 7 || newest.Messages[2].RoomSeq != 5 {
			t.Fatalf("newest page order wrong: %+v", newest.Messages)
		}

		next, err := e.Messages(ctx, paged, newest.Messages[2].RoomSeq, 3)
		if err != nil || len(next.Messages) != 3 || next.Messages[0].RoomSeq != 4 {
			t.Fatalf("cursor page: page=%+v err=%v", next, err)
		}
		tail, err := e.Messages(ctx, paged, next.Messages[2].RoomSeq, 3)
		if err != nil || len(tail.Messages) != 1 || tail.HasMore {
			t.Fatalf("tail page: page=%+v err=%v", tail, err)
		}

		whole, err := e.Messages(ctx, paged, 0, 20)
		if err != nil || len(whole.Messages) != 7 || whole.HasMore {
			t.Fatalf("full page: page=%+v err=%v", whole, err)
		}
		oversized, err := e.Messages(ctx, paged, 0, 500)
		if err != nil || len(oversized.Messages) != 7 {
			t.Fatalf("oversized limit must clamp: page=%+v err=%v", oversized, err)
		}
		if empty, err := e.Messages(ctx, uuid.NewV7(), 0, 20); err != nil || len(empty.Messages) != 0 {
			t.Fatalf("missing room page: page=%+v err=%v", empty, err)
		}

		afterPage, err := e.MessagesAfter(ctx, paged, 4, 10)
		if err != nil || len(afterPage.Messages) != 3 {
			t.Fatalf("after page: page=%+v err=%v", afterPage, err)
		}
		if afterPage.Messages[0].RoomSeq != 5 || afterPage.Messages[1].RoomSeq != 6 ||
			afterPage.Messages[2].RoomSeq != 7 {
			t.Fatalf("after page ordering wrong: %+v", afterPage.Messages)
		}
	})

	t.Run("concurrent writes keep sequence dense", func(t *testing.T) {
		room := uuid.NewV7()
		const writers, perWriter = 4, 25
		var wg sync.WaitGroup
		wg.Add(writers)
		for range writers {
			go func() {
				defer wg.Done()
				for range perWriter {
					if _, err := e.WriteMessage(ctx, Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
						MsgType: MsgTypeText, Payload: []byte("concurrent"),
					}); err != nil {
						t.Errorf("concurrent write: %v", err)
						return
					}
				}
			}()
		}
		wg.Wait()

		page, err := e.Messages(ctx, room, 0, 100)
		if err != nil {
			t.Fatal(err)
		}
		if len(page.Messages) != writers*perWriter {
			t.Fatalf("page holds %d messages, want %d", len(page.Messages), writers*perWriter)
		}
		for i, msg := range page.Messages {
			if want := uint64(writers * perWriter); msg.RoomSeq != want-uint64(i) {
				t.Fatalf("sequence not dense at %d: seq=%d", i, msg.RoomSeq)
			}
		}
	})

	t.Run("search skips recalled and honours cursor", func(t *testing.T) {
		room := uuid.NewV7()
		hit := mustWriteMessage(t, e, room, sender, "Needle Here")
		mustWriteMessage(t, e, room, sender, "nothing special")
		recalled := mustWriteMessage(t, e, room, sender, "NEEDLE to remove")
		if _, err := e.RecallMessage(ctx, room, recalled.MsgID, sender, false); err != nil {
			t.Fatal(err)
		}

		found, err := e.SearchRoomMessages(ctx, room, "  NEEDLE  ", 0, 10)
		if err != nil || len(found) != 1 || found[0].MsgID != hit.MsgID {
			t.Fatalf("search: found=%+v err=%v", found, err)
		}
		if _, err := e.SearchRoomMessages(ctx, room, "", 0, 10); err != nil || len(found) != 1 {
			t.Fatalf("empty keyword must return nothing, got err=%v", err)
		}
	})

	t.Run("message by id", func(t *testing.T) {
		msg := mustWriteMessage(t, e, room, sender, "byid")
		got, err := e.MessageByID(ctx, room, msg.MsgID)
		if err != nil || got.RoomSeq != msg.RoomSeq {
			t.Fatalf("message by id: msg=%+v err=%v", got, err)
		}
		if _, err := e.MessageByID(ctx, room, uuid.NewV7()); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing message error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("recall rules", func(t *testing.T) {
		room := uuid.NewV7()
		msg := mustWriteMessage(t, e, room, sender, "recall me")
		outsider := uuid.NewV7()

		if _, err := e.RecallMessage(ctx, room, msg.MsgID, outsider, false); !errors.Is(err, ErrForbidden) {
			t.Fatalf("outsider recall error = %v, want %v", err, ErrForbidden)
		}
		stale := mustWriteMessage(t, e, room, sender, "too old to recall")
		if err := e.backdateMessage(stale, time.Hour); err != nil {
			t.Fatal(err)
		}
		if _, err := e.RecallMessage(
			ctx,
			room,
			stale.MsgID,
			sender,
			false,
		); !errors.Is(
			err,
			ErrRecallTimeout,
		) {
			t.Fatalf("stale recall error = %v, want %v", err, ErrRecallTimeout)
		}
		recalled, err := e.RecallMessage(ctx, room, stale.MsgID, outsider, true)
		if err != nil || recalled.MsgType != MsgTypeRecall {
			t.Fatalf("owner recall: msg=%+v err=%v", recalled, err)
		}
		again, err := e.RecallMessage(ctx, room, msg.MsgID, sender, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := e.RecallMessage(
			ctx,
			room,
			msg.MsgID,
			sender,
			false,
		); err != nil ||
			again.MsgType != MsgTypeRecall {
			t.Fatalf("repeat recall: msg=%+v err=%v", again, err)
		}

		page, err := e.Messages(ctx, room, 0, 10)
		if err != nil {
			t.Fatal(err)
		}
		for i := range page.Messages {
			if page.Messages[i].MsgID == msg.MsgID && page.Messages[i].MsgType != MsgTypeRecall {
				t.Fatalf("recall not visible in tail cache: %+v", page.Messages[i])
			}
		}
	})

	t.Run("cancelled context fails fast", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := e.WriteMessage(cancelled, Message{RoomID: room}); !errors.Is(err, context.Canceled) {
			t.Fatalf("write error = %v, want %v", err, context.Canceled)
		}
		if _, err := e.Messages(cancelled, room, 0, 10); !errors.Is(err, context.Canceled) {
			t.Fatalf("read error = %v, want %v", err, context.Canceled)
		}
	})
}

// backdateMessage rewrites the stored message with an ancient ServerTime so
// recall-window logic can be exercised without sleeping.
func (e *Engine) backdateMessage(message Message, age time.Duration) error {
	message.ServerTime = time.Now().Add(-age).UnixMicro()
	batch := e.db.NewBatch()
	defer batch.Close()
	var keyBuf [25]byte
	if err := e.setBytes(
		batch,
		appendMessageKey(keyBuf[:0], message.RoomID, message.RoomSeq),
		encodeMessage(message),
	); err != nil {
		return err
	}
	return commit(batch)
}

func benchIDs() (uuid.UUID, uuid.UUID) {
	return uuid.NewV7(), uuid.NewV7()
}

func BenchmarkEngineMessageWrite(b *testing.B) {
	for _, batch := range []int{1, 32, 100} {
		b.Run(fmt.Sprintf("batch-%d", batch), func(b *testing.B) {
			e := newTestEngine(b)
			room, sender := benchIDs()
			payload := []byte(`{"text":"benchmark payload"}`)

			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				messages := make([]Message, batch)
				for i := range messages {
					messages[i] = Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
						MsgType: MsgTypeText, Payload: payload,
					}
				}
				results := e.WriteMessages(context.Background(), messages)
				for i := range results {
					if results[i].Err != nil {
						b.Fatal(results[i].Err)
					}
				}
			}
		})
	}
}

func BenchmarkEngineMessageWriteParallel(b *testing.B) {
	for name, rooms := range map[string]int{"single-room": 1, "multi-room": 64} {
		b.Run(name, func(b *testing.B) {
			e := newTestEngine(b)
			sender := uuid.NewV7()
			roomIDs := make([]uuid.UUID, rooms)
			for i := range roomIDs {
				roomIDs[i] = uuid.NewV7()
			}
			payload := []byte(`{"text":"benchmark payload"}`)

			b.ResetTimer()
			b.ReportAllocs()
			b.RunParallel(func(pb *testing.PB) {
				for i := 0; pb.Next(); i++ {
					if _, err := e.WriteMessage(context.Background(), Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: roomIDs[i%rooms],
						MsgType: MsgTypeText, Payload: payload,
					}); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkEngineRead(b *testing.B) {
	b.Run("newest-page", func(b *testing.B) {
		e := newTestEngine(b)
		room, sender := benchIDs()
		benchWrite(b, e, room, sender, 200)

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			page, err := e.Messages(context.Background(), room, 0, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("read messages: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
	// A page older than the memory tail: this is the Pebble scan-and-decode
	// path every deeper history request takes.
	b.Run("deep-page", func(b *testing.B) {
		e := newTestEngine(b)
		room, sender := benchIDs()
		benchWrite(b, e, room, sender, 500)

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			page, err := e.Messages(context.Background(), room, 350, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("deep page: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
	b.Run("sync-pull-after", func(b *testing.B) {
		e := newTestEngine(b)
		room, sender := benchIDs()
		benchWrite(b, e, room, sender, 50)

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			page, err := e.MessagesAfter(context.Background(), room, 30, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("sync pull after: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
}

func benchWrite(b *testing.B, e *Engine, room, sender uuid.UUID, count int) []Message {
	b.Helper()
	messages := make([]Message, count)
	for i := range count {
		message, err := e.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
			MsgType: MsgTypeText, Payload: []byte("payload"),
		})
		if err != nil {
			b.Fatal(err)
		}
		messages[i] = message
	}
	return messages
}

func BenchmarkEngineSearch(b *testing.B) {
	b.Run("recent-200", func(b *testing.B) {
		e := newTestEngine(b)
		room, sender := benchIDs()
		for i := range 200 {
			if _, err := e.WriteMessage(context.Background(), Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
				MsgType: MsgTypeText,
				Payload: jsontext.Value(`{"text":"message number ` + string(rune('a'+i%26)) + `"}`),
			}); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			found, err := e.SearchRoomMessages(context.Background(), room, "number", 0, 20)
			if err != nil || len(found) != 20 {
				b.Fatalf("search: count=%d err=%v", len(found), err)
			}
		}
	})
	b.Run("deep-scan-5000", func(b *testing.B) {
		e := newTestEngine(b)
		room, sender := benchIDs()
		for i := range 5000 {
			text := "filler"
			if i < 20 {
				text = "needle"
			}
			if _, err := e.WriteMessage(context.Background(), Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
				MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"` + text + `"}`),
			}); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			found, err := e.SearchRoomMessages(context.Background(), room, "needle", 0, 20)
			if err != nil || len(found) != 20 {
				b.Fatalf("search: count=%d err=%v", len(found), err)
			}
		}
	})
}

func BenchmarkEngineRecall(b *testing.B) {
	b.Run("write-and-recall", func(b *testing.B) {
		e := newTestEngine(b)
		room, sender := benchIDs()

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			message, err := e.WriteMessage(context.Background(), Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
				MsgType: MsgTypeText, Payload: []byte("recall bench"),
			})
			if err != nil {
				b.Fatal(err)
			}
			if _, err := e.RecallMessage(context.Background(), room, message.MsgID, sender, false); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkRoomHead(b *testing.B) {
	data := appendRoomHead(nil, 42)

	b.Run("encode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			appendRoomHead(nil, 42)
		}
	})
	b.Run("decode", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := decodeLastSeq(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkEngineReadWithReactions measures a newest-page history read while
// the room carries reactions across its lifetime: the page read must cost
// what the page shows, not what the room accumulated.
func BenchmarkEngineReadWithReactions(b *testing.B) {
	for _, tc := range []struct {
		name        string
		messages    int
		everyNth    int // every n-th seeded message carries reactions
		usersPerMsg int
	}{
		{"reactions-on-old-messages", 1000, 10, 4},
		{"reactions-everywhere", 500, 1, 4},
	} {
		b.Run(tc.name, func(b *testing.B) {
			e := newTestEngine(b)
			room, sender := benchIDs()
			ctx := context.Background()
			users := make([]uuid.UUID, tc.usersPerMsg)
			for i := range users {
				users[i] = uuid.NewV7()
			}
			for i := range tc.messages {
				message, err := e.WriteMessage(ctx, Message{
					ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
					MsgType: MsgTypeText, Payload: []byte("payload"),
				})
				if err != nil {
					b.Fatal(err)
				}
				if i%tc.everyNth == 0 {
					for _, user := range users {
						if err := e.AddReaction(ctx, room, message.MsgID, user, "👍"); err != nil {
							b.Fatal(err)
						}
					}
				}
			}

			page, err := e.Messages(ctx, room, 0, 20)
			if err != nil {
				b.Fatal(err)
			}
			visible := 0
			for i := range page.Messages {
				visible += len(page.Messages[i].Reactions)
			}
			if visible == 0 {
				b.Fatal("page carries no reactions; benchmark would measure nothing")
			}

			b.ResetTimer()
			b.ReportAllocs()
			for b.Loop() {
				page, err := e.Messages(ctx, room, 0, 20)
				if err != nil || len(page.Messages) != 20 {
					b.Fatalf("read messages: count=%d err=%v", len(page.Messages), err)
				}
			}
		})
	}
}
