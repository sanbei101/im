package store

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
	"uuid"
)

// newTestStore opens a throwaway store and registers its cleanup.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := data.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})
	return data
}

func mustCreateUser(t *testing.T, _ *Store, name string) User {
	t.Helper()
	return User{UserID: uuid.NewV7(), Username: name, Nickname: name, CreatedAt: time.Now()}
}

func mustCreateRoom(t *testing.T, _ *Store, chatType string, _ ...Member) Room {
	t.Helper()
	room := Room{RoomID: uuid.NewV7(), ChatType: chatType}
	if chatType == ChatTypeGroup {
		room.Name = "test room"
	}
	return room
}

func mustWriteMessage(t *testing.T, s *Store, roomID, senderID uuid.UUID, payload string) Message {
	t.Helper()
	message, err := s.WriteMessage(context.Background(), Message{
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
		want := Dedup{MsgID: uuid.NewV7(), RoomSeq: 7, ServerTime: 42, PayloadSum: [32]byte{1, 2, 3}}
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

func TestStoreMessages(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	sender := mustCreateUser(t, s, "msg-sender")
	room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})

	t.Run("write assigns identity", func(t *testing.T) {
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "first")
		if msg.RoomSeq != 1 || msg.MsgID == uuid.Nil() || msg.ServerTime == 0 {
			t.Fatalf("server did not stamp message: %+v", msg)
		}
		got, err := s.Room(ctx, room.RoomID)
		if err != nil || got.LastSeq != 1 {
			t.Fatalf("room last seq: room=%+v err=%v", got, err)
		}
		// Work lazily creates its message head; room metadata and membership
		// live in PostgreSQL, so a missing Pebble room is not an error.
		freshRoom := uuid.NewV7()
		if _, err := s.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(), SenderID: sender.UserID, RoomID: freshRoom, MsgType: MsgTypeText,
		}); err != nil {
			t.Fatalf("write to fresh room: %v", err)
		}
	})

	t.Run("idempotent retry and conflict", func(t *testing.T) {
		clientID := uuid.NewV7()
		first, err := s.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender.UserID, RoomID: room.RoomID,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"same"}`),
		})
		if err != nil {
			t.Fatal(err)
		}
		retry, err := s.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender.UserID, RoomID: room.RoomID,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"same"}`),
		})
		if err != nil || retry.MsgID != first.MsgID || retry.RoomSeq != first.RoomSeq {
			t.Fatalf("retry changed result: first=%+v retry=%+v err=%v", first, retry, err)
		}
		if _, err := s.WriteMessage(ctx, Message{
			ClientMsgID: clientID, SenderID: sender.UserID, RoomID: room.RoomID,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"changed"}`),
		}); !errors.Is(err, ErrConflict) {
			t.Fatalf("changed payload error = %v, want %v", err, ErrConflict)
		}
	})

	t.Run("batch dedup within one commit", func(t *testing.T) {
		clientID := uuid.NewV7()
		results := s.WriteMessages(ctx, []Message{
			{
				ClientMsgID: clientID,
				SenderID:    sender.UserID,
				RoomID:      room.RoomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("dup"),
			},
			{
				ClientMsgID: clientID,
				SenderID:    sender.UserID,
				RoomID:      room.RoomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("dup"),
			},
			{
				ClientMsgID: uuid.NewV7(),
				SenderID:    sender.UserID,
				RoomID:      room.RoomID,
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
		paged := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		for i := range 7 {
			mustWriteMessage(t, s, paged.RoomID, sender.UserID, "m"+string(rune('a'+i)))
		}

		newest, err := s.Messages(ctx, paged.RoomID, 0, 3)
		if err != nil || len(newest.Messages) != 3 || !newest.HasMore {
			t.Fatalf("newest page: page=%+v err=%v", newest, err)
		}
		if newest.Messages[0].RoomSeq != 7 || newest.Messages[2].RoomSeq != 5 {
			t.Fatalf("newest page order wrong: %+v", newest.Messages)
		}

		next, err := s.Messages(ctx, paged.RoomID, newest.Messages[2].RoomSeq, 3)
		if err != nil || len(next.Messages) != 3 || next.Messages[0].RoomSeq != 4 {
			t.Fatalf("cursor page: page=%+v err=%v", next, err)
		}
		tail, err := s.Messages(ctx, paged.RoomID, next.Messages[2].RoomSeq, 3)
		if err != nil || len(tail.Messages) != 1 || tail.HasMore {
			t.Fatalf("tail page: page=%+v err=%v", tail, err)
		}

		whole, err := s.Messages(ctx, paged.RoomID, 0, 20)
		if err != nil || len(whole.Messages) != 7 || whole.HasMore {
			t.Fatalf("full page: page=%+v err=%v", whole, err)
		}
		oversized, err := s.Messages(ctx, paged.RoomID, 0, 500)
		if err != nil || len(oversized.Messages) != 7 {
			t.Fatalf("oversized limit must clamp: page=%+v err=%v", oversized, err)
		}
		if empty, err := s.Messages(ctx, uuid.NewV7(), 0, 20); err != nil || len(empty.Messages) != 0 {
			t.Fatalf("missing room page: page=%+v err=%v", empty, err)
		}

		afterPage, err := s.MessagesAfter(ctx, paged.RoomID, 4, 10)
		if err != nil || len(afterPage.Messages) != 3 {
			t.Fatalf("after page: page=%+v err=%v", afterPage, err)
		}
		if afterPage.Messages[0].RoomSeq != 5 || afterPage.Messages[1].RoomSeq != 6 ||
			afterPage.Messages[2].RoomSeq != 7 {
			t.Fatalf("after page ordering wrong: %+v", afterPage.Messages)
		}
	})

	t.Run("concurrent writes keep sequence dense", func(t *testing.T) {
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		const writers, perWriter = 4, 25
		var wg sync.WaitGroup
		wg.Add(writers)
		for range writers {
			go func() {
				defer wg.Done()
				for range perWriter {
					if _, err := s.WriteMessage(ctx, Message{
						ClientMsgID: uuid.NewV7(), SenderID: sender.UserID, RoomID: room.RoomID,
						MsgType: MsgTypeText, Payload: []byte("concurrent"),
					}); err != nil {
						t.Errorf("concurrent write: %v", err)
						return
					}
				}
			}()
		}
		wg.Wait()

		page, err := s.Messages(ctx, room.RoomID, 0, 100)
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
		room := mustCreateRoom(t, s, ChatTypeGroup, Member{UserID: sender.UserID, Role: RoleOwner})
		hit := mustWriteMessage(t, s, room.RoomID, sender.UserID, "Needle Here")
		mustWriteMessage(t, s, room.RoomID, sender.UserID, "nothing special")
		recalled := mustWriteMessage(t, s, room.RoomID, sender.UserID, "NEEDLE to remove")
		if _, err := s.RecallMessage(ctx, room.RoomID, recalled.MsgID, sender.UserID, false); err != nil {
			t.Fatal(err)
		}

		found, err := s.SearchRoomMessages(ctx, room.RoomID, "  NEEDLE  ", 0, 10)
		if err != nil || len(found) != 1 || found[0].MsgID != hit.MsgID {
			t.Fatalf("search: found=%+v err=%v", found, err)
		}
		if _, err := s.SearchRoomMessages(ctx, room.RoomID, "", 0, 10); err != nil || len(found) != 1 {
			t.Fatalf("empty keyword must return nothing, got err=%v", err)
		}
	})

	t.Run("message by id", func(t *testing.T) {
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "byid")
		got, err := s.MessageByID(ctx, room.RoomID, msg.MsgID)
		if err != nil || got.RoomSeq != msg.RoomSeq {
			t.Fatalf("message by id: msg=%+v err=%v", got, err)
		}
		if _, err := s.MessageByID(ctx, room.RoomID, uuid.NewV7()); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing message error = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("recall rules", func(t *testing.T) {
		room := mustCreateRoom(t, s, ChatTypeGroup,
			Member{UserID: sender.UserID, Role: RoleMember},
			Member{UserID: mustCreateUser(t, s, "recall-owner").UserID, Role: RoleOwner},
		)
		msg := mustWriteMessage(t, s, room.RoomID, sender.UserID, "recall me")
		outsider := uuid.NewV7()

		if _, err := s.RecallMessage(ctx, room.RoomID, msg.MsgID, outsider, false); !errors.Is(err, ErrForbidden) {
			t.Fatalf("outsider recall error = %v, want %v", err, ErrForbidden)
		}
		stale := mustWriteMessage(t, s, room.RoomID, sender.UserID, "too old to recall")
		if err := s.backdateMessage(stale, time.Hour); err != nil {
			t.Fatal(err)
		}
		if _, err := s.RecallMessage(
			ctx,
			room.RoomID,
			stale.MsgID,
			sender.UserID,
			false,
		); !errors.Is(
			err,
			ErrRecallTimeout,
		) {
			t.Fatalf("stale recall error = %v, want %v", err, ErrRecallTimeout)
		}
		recalled, err := s.RecallMessage(ctx, room.RoomID, stale.MsgID, outsider, true)
		if err != nil || recalled.MsgType != MsgTypeRecall {
			t.Fatalf("owner recall: msg=%+v err=%v", recalled, err)
		}
		again, err := s.RecallMessage(ctx, room.RoomID, msg.MsgID, sender.UserID, false)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.RecallMessage(
			ctx,
			room.RoomID,
			msg.MsgID,
			sender.UserID,
			false,
		); err != nil ||
			again.MsgType != MsgTypeRecall {
			t.Fatalf("repeat recall: msg=%+v err=%v", again, err)
		}

		page, err := s.Messages(ctx, room.RoomID, 0, 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range page.Messages {
			if m.MsgID == msg.MsgID && m.MsgType != MsgTypeRecall {
				t.Fatalf("recall not visible in tail cache: %+v", m)
			}
		}
	})

	t.Run("cancelled context fails fast", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := s.WriteMessage(cancelled, Message{RoomID: room.RoomID}); !errors.Is(err, context.Canceled) {
			t.Fatalf("write error = %v, want %v", err, context.Canceled)
		}
		if _, err := s.Messages(cancelled, room.RoomID, 0, 10); !errors.Is(err, context.Canceled) {
			t.Fatalf("read error = %v, want %v", err, context.Canceled)
		}
	})
}

// backdateMessage rewrites the stored message with an ancient ServerTime so
// recall-window logic can be exercised without sleeping.
func (s *Store) backdateMessage(message Message, age time.Duration) error {
	message.ServerTime = time.Now().Add(-age).UnixMicro()
	batch := s.db.NewBatch()
	defer batch.Close()
	var keyBuf [25]byte
	if err := s.setBytes(
		batch,
		appendMessageKey(keyBuf[:0], message.RoomID, message.RoomSeq),
		encodeMessage(message),
	); err != nil {
		return err
	}
	return commit(batch)
}

func newBenchStore(b *testing.B) *Store {
	b.Helper()
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		if err := data.Close(); err != nil {
			b.Errorf("close store: %v", err)
		}
	})
	return data
}

func benchRoom(b *testing.B, _ *Store) (uuid.UUID, uuid.UUID) {
	b.Helper()
	return uuid.NewV7(), uuid.NewV7()
}

func BenchmarkStoreMessageWrite(b *testing.B) {
	for _, batch := range []int{1, 32, 100} {
		b.Run(fmt.Sprintf("batch-%d", batch), func(b *testing.B) {
			s := newBenchStore(b)
			room, sender := benchRoom(b, s)
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
				results := s.WriteMessages(context.Background(), messages)
				for i := range results {
					if results[i].Err != nil {
						b.Fatal(results[i].Err)
					}
				}
			}
		})
	}
}

func BenchmarkStoreMessageWriteParallel(b *testing.B) {
	for name, rooms := range map[string]int{"single-room": 1, "multi-room": 64} {
		b.Run(name, func(b *testing.B) {
			s := newBenchStore(b)
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
					if _, err := s.WriteMessage(context.Background(), Message{
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

func BenchmarkStoreRead(b *testing.B) {
	b.Run("newest-page", func(b *testing.B) {
		s := newBenchStore(b)
		room, sender := benchRoom(b, s)
		benchWrite(b, s, room, sender, 200)

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			page, err := s.Messages(context.Background(), room, 0, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("read messages: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
	b.Run("sync-pull-after", func(b *testing.B) {
		s := newBenchStore(b)
		room, sender := benchRoom(b, s)
		benchWrite(b, s, room, sender, 50)

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			page, err := s.MessagesAfter(context.Background(), room, 30, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("sync pull after: count=%d err=%v", len(page.Messages), err)
			}
		}
	})
}

func benchWrite(b *testing.B, s *Store, room, sender uuid.UUID, count int) []Message {
	b.Helper()
	messages := make([]Message, count)
	for i := range count {
		message, err := s.WriteMessage(context.Background(), Message{
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

func BenchmarkStoreSearch(b *testing.B) {
	b.Run("recent-200", func(b *testing.B) {
		s := newBenchStore(b)
		room, sender := benchRoom(b, s)
		for i := range 200 {
			if _, err := s.WriteMessage(context.Background(), Message{
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
			found, err := s.SearchRoomMessages(context.Background(), room, "number", 0, 20)
			if err != nil || len(found) != 20 {
				b.Fatalf("search: count=%d err=%v", len(found), err)
			}
		}
	})
	b.Run("deep-scan-5000", func(b *testing.B) {
		s := newBenchStore(b)
		room, sender := benchRoom(b, s)
		for i := range 5000 {
			text := "filler"
			if i < 20 {
				text = "needle"
			}
			if _, err := s.WriteMessage(context.Background(), Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
				MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"` + text + `"}`),
			}); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			found, err := s.SearchRoomMessages(context.Background(), room, "needle", 0, 20)
			if err != nil || len(found) != 20 {
				b.Fatalf("search: count=%d err=%v", len(found), err)
			}
		}
	})
}

func BenchmarkStoreRecall(b *testing.B) {
	s := newBenchStore(b)
	room, sender := benchRoom(b, s)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		message, err := s.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
			MsgType: MsgTypeText, Payload: []byte("recall bench"),
		})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := s.RecallMessage(context.Background(), room, message.MsgID, sender, false); err != nil {
			b.Fatal(err)
		}
	}
}

func benchRoomRecord(b *testing.B) []byte {
	b.Helper()
	room := Room{
		RoomID: uuid.NewV7(), ChatType: ChatTypeGroup,
		Name: "一个长度可观的房间名称", AvatarURL: "room://0197a2b3-c4d5-e6f7-a8b9-c0d1e2f3a4b5",
		Notice: "群公告:请勿发送与主题无关的内容", LastSeq: 42,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
		LastMsg: &Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(),
			RoomID: uuid.UUID{}, RoomSeq: 42, ServerTime: time.Now().UnixMicro(),
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"hello world"}`),
		},
	}
	return encodeRoom(room)
}

func BenchmarkRoomDecode(b *testing.B) {
	data := benchRoomRecord(b)
	b.Run("full", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			if _, err := decodeRoom(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("head", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			if _, err := decodeRoomHead(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
