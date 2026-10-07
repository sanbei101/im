package store

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"sync"
	"testing"
	"uuid"
)

// TestMessageJSONOmitsEmptyExt 回归：Ext 为空但非 nil（客户端带了空 ext）时，
// omitzero 仍会把它当成\"有值\"并输出空字符串，整条 JSON 直接非法。
func TestMessageJSONOmitsEmptyExt(t *testing.T) {
	for _, ext := range []jsontext.Value{nil, jsontext.Value(""), {}} {
		m := Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), RoomSeq: 1,
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"hi"}`), Ext: ext,
		}
		encoded, err := json.Marshal(m)
		if err != nil {
			t.Fatalf("marshal with ext=%q: %v", string(ext), err)
		}
		if bytes.Contains(encoded, []byte(`"ext"`)) {
			t.Fatalf("empty ext should be omitted, got %s", encoded)
		}
		var out map[string]any
		if err := json.Unmarshal(encoded, &out); err != nil {
			t.Fatalf("output is not valid JSON: %s: %v", encoded, err)
		}
	}
}

// TestMessageBinaryRoundTrip 覆盖 int8 MsgType 与可变长 ReplyToMsgID 的二进制布局，
// 以及截断记录必须报错而不是静默解码。
func TestMessageBinaryRoundTrip(t *testing.T) {
	for _, m := range []Message{
		{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 9, ServerTime: 123, MsgType: MsgTypeSystem, Payload: jsontext.Value(`{"s":1}`),
		},
		{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(), RoomID: uuid.NewV7(),
			RoomSeq: 10, ServerTime: 456, ReplyToMsgID: uuid.NewV7(),
			MsgType: MsgTypeFile, Payload: jsontext.Value(`{"u":"x"}`), Ext: jsontext.Value(`{"e":1}`),
		},
	} {
		got, err := decodeMessage(encodeMessage(m))
		if err != nil {
			t.Fatal(err)
		}
		if got.MsgID != m.MsgID || got.ClientMsgID != m.ClientMsgID || got.SenderID != m.SenderID ||
			got.RoomID != m.RoomID || got.RoomSeq != m.RoomSeq || got.ServerTime != m.ServerTime ||
			got.MsgType != m.MsgType || string(got.Payload) != string(m.Payload) ||
			string(got.Ext) != string(m.Ext) || got.ReplyToMsgID != m.ReplyToMsgID {
			t.Fatalf("round trip mismatch:\nwant %+v\ngot  %+v", m, got)
		}
	}
	// truncated records must be rejected, not silently decoded
	for _, bad := range [][]byte{
		{},
		make([]byte, 60),
	} {
		if _, err := decodeMessage(bad); err == nil {
			t.Fatalf("expected error for %d-byte record", len(bad))
		}
	}
}

func TestStoreMessageIdempotencyPaginationAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	storePath := t.TempDir() + "/store"
	data, err := Open(storePath)
	if err != nil {
		t.Fatal(err)
	}

	userID := uuid.NewV7()
	roomID := uuid.NewV7()
	if _, err := data.CreateUser(ctx, "alice", "password"); err != nil {
		t.Fatal(err)
	}
	if err := data.CreateRoom(
		ctx,
		Room{RoomID: roomID, ChatType: "single"},
		[]Member{{UserID: userID, Role: "member"}},
	); err != nil {
		t.Fatal(err)
	}

	clientID := uuid.NewV7()
	first, err := data.WriteMessage(ctx, Message{
		ClientMsgID: clientID,
		SenderID:    userID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     []byte(`{"text":"first"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	retry, err := data.WriteMessage(ctx, Message{
		ClientMsgID: clientID,
		SenderID:    userID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     []byte(`{"text":"first"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if retry.MsgID != first.MsgID || retry.RoomSeq != first.RoomSeq {
		t.Fatalf("idempotent retry changed result: first=%+v retry=%+v", first, retry)
	}
	_, err = data.WriteMessage(ctx, Message{
		ClientMsgID: clientID,
		SenderID:    userID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     []byte(`{"text":"changed"}`),
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("changed idempotent payload error = %v, want %v", err, ErrConflict)
	}

	for _, text := range []string{"second", "third"} {
		if _, err := data.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    userID,
			RoomID:      roomID,
			MsgType:     MsgTypeText,
			Payload:     []byte(`{"text":"` + text + `"}`),
		}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := data.Messages(ctx, roomID, 0, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 2 || !page.HasMore || string(page.Messages[0].Payload) != `{"text":"third"}` {
		t.Fatalf("unexpected latest page: %+v", page)
	}
	beforeFirst, err := data.Messages(ctx, roomID, first.RoomSeq+1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeFirst.Messages) != 1 || beforeFirst.Messages[0].RoomSeq != first.RoomSeq {
		t.Fatalf("unexpected before page: %+v", beforeFirst)
	}

	checkpointPath := t.TempDir() + "/checkpoint"
	if err := data.Checkpoint(ctx, checkpointPath); err != nil {
		t.Fatal(err)
	}
	if err := data.Close(); err != nil {
		t.Fatal(err)
	}
	checkpoint, err := Open(checkpointPath)
	if err != nil {
		t.Fatal(err)
	}
	defer checkpoint.Close()
	room, err := checkpoint.Room(ctx, roomID)
	if err != nil {
		t.Fatal(err)
	}
	if room.LastSeq != 3 {
		t.Fatalf("checkpoint room last sequence = %d, want 3", room.LastSeq)
	}
}

func TestStoreContextCancellation(t *testing.T) {
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := data.CreateUser(ctx, "cancelled", "password"); !errors.Is(err, context.Canceled) {
		t.Fatalf("CreateUser error = %v, want context.Canceled", err)
	}
	if _, err := data.Messages(ctx, uuid.NewV7(), 0, 20); !errors.Is(err, context.Canceled) {
		t.Fatalf("Messages error = %v, want context.Canceled", err)
	}
}

func TestStoreConcurrentMessageWrites(t *testing.T) {
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		t.Fatal(err)
	}

	const writers = 8
	const messagesPerWriter = 25
	var wg sync.WaitGroup
	wg.Add(writers)
	for range writers {
		go func() {
			defer wg.Done()
			for range messagesPerWriter {
				if _, err := data.WriteMessage(context.Background(), Message{
					ClientMsgID: uuid.NewV7(),
					SenderID:    senderID,
					RoomID:      roomID,
					MsgType:     MsgTypeText,
					Payload:     []byte("concurrent"),
				}); err != nil {
					t.Errorf("concurrent WriteMessage: %v", err)
				}
			}
		}()
	}
	wg.Wait()

	room, err := data.Room(context.Background(), roomID)
	if err != nil {
		t.Fatal(err)
	}
	want := uint64(writers * messagesPerWriter)
	if room.LastSeq != want {
		t.Fatalf("room last sequence = %d, want %d", room.LastSeq, want)
	}
	page, err := data.Messages(context.Background(), roomID, 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 100 || page.Messages[0].RoomSeq != want {
		t.Fatalf(
			"unexpected concurrent message page: len=%d first_seq=%d",
			len(page.Messages),
			page.Messages[0].RoomSeq,
		)
	}
}

func TestStoreBatchWriteDedupAndClose(t *testing.T) {
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		t.Fatal(err)
	}
	clientID := uuid.NewV7()
	results := data.WriteMessages(context.Background(), []Message{
		{ClientMsgID: clientID, SenderID: senderID, RoomID: roomID, MsgType: MsgTypeText, Payload: []byte("same")},
		{ClientMsgID: clientID, SenderID: senderID, RoomID: roomID, MsgType: MsgTypeText, Payload: []byte("same")},
		{ClientMsgID: uuid.NewV7(), SenderID: senderID, RoomID: roomID, MsgType: MsgTypeText, Payload: []byte("next")},
	})
	if len(results) != 3 || results[0].Err != nil || results[1].Err != nil || results[2].Err != nil {
		t.Fatalf("batch results = %+v", results)
	}
	if results[0].Message.MsgID != results[1].Message.MsgID ||
		results[0].Message.RoomSeq != results[1].Message.RoomSeq {
		t.Fatalf("duplicate batch result changed: %+v", results)
	}
	if results[2].Message.RoomSeq != 2 {
		t.Fatalf("next batch sequence = %d, want 2", results[2].Message.RoomSeq)
	}
	if err := data.Close(); err != nil {
		t.Fatal(err)
	}
	closed := data.WriteMessages(context.Background(), []Message{{RoomID: roomID}})
	if len(closed) != 1 || !errors.Is(closed[0].Err, ErrClosed) {
		t.Fatalf("closed write result = %+v", closed)
	}
}

func BenchmarkWriteMessagesBatch(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()
	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		messages := make([]Message, 32)
		for i := range messages {
			messages[i] = Message{
				ClientMsgID: uuid.NewV7(),
				SenderID:    senderID,
				RoomID:      roomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("batch"),
			}
		}
		results := data.WriteMessages(context.Background(), messages)
		for i := range results {
			if results[i].Err != nil {
				b.Fatal(results[i].Err)
			}
		}
	}
}

func BenchmarkWriteMessages100Batch(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()
	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		messages := make([]Message, 100)
		for i := range messages {
			messages[i] = Message{
				ClientMsgID: uuid.NewV7(),
				SenderID:    senderID,
				RoomID:      roomID,
				MsgType:     MsgTypeText,
				Payload:     []byte("batch"),
			}
		}
		results := data.WriteMessages(context.Background(), messages)
		for i := range results {
			if results[i].Err != nil {
				b.Fatal(results[i].Err)
			}
		}
	}
}

func BenchmarkWriteMessage(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}

	payload := []byte(`{"text":"benchmark payload"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if _, err := data.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    senderID,
			RoomID:      roomID,
			MsgType:     MsgTypeText,
			Payload:     payload,
		}); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadMessages(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		context.Background(),
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}

	for range 100 {
		if _, err := data.WriteMessage(context.Background(), Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    senderID,
			RoomID:      roomID,
			MsgType:     MsgTypeText,
			Payload:     []byte("payload"),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		page, err := data.Messages(context.Background(), roomID, 0, 20)
		if err != nil || len(page.Messages) != 20 {
			b.Fatalf("read messages: page=%+v err=%v", page, err)
		}
	}
}

func BenchmarkReadMessagesParallel(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		ctx,
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}

	for range 200 {
		if _, err := data.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    senderID,
			RoomID:      roomID,
			MsgType:     MsgTypeText,
			Payload:     []byte("payload"),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			page, err := data.Messages(ctx, roomID, 0, 20)
			if err != nil || len(page.Messages) != 20 {
				b.Fatalf("read messages: count=%d, err=%v", len(page.Messages), err)
			}
		}
	})
}

func BenchmarkMessageByID(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	roomID := uuid.NewV7()
	senderID := uuid.NewV7()
	if err := data.CreateRoom(
		ctx,
		Room{RoomID: roomID, ChatType: "group"},
		[]Member{{UserID: senderID, Role: "owner"}},
	); err != nil {
		b.Fatal(err)
	}

	var targetMsgID uuid.UUID
	for range 50 {
		msg, err := data.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    senderID,
			RoomID:      roomID,
			MsgType:     MsgTypeText,
			Payload:     []byte("payload"),
		})
		if err != nil {
			b.Fatal(err)
		}
		targetMsgID = msg.MsgID
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		msg, err := data.MessageByID(ctx, roomID, targetMsgID)
		if err != nil || msg.MsgID != targetMsgID {
			b.Fatalf("message by id: err=%v", err)
		}
	}
}

func BenchmarkConversations50Rooms(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	userID := uuid.NewV7()
	data.CreateUser(ctx, "bench_user", "pass")

	for i := range 50 {
		roomID := uuid.NewV7()
		if err := data.CreateRoom(ctx, Room{
			RoomID: roomID, ChatType: "group", Name: fmt.Sprintf("Room %d", i),
		}, []Member{{UserID: userID, Role: "member"}}); err != nil {
			b.Fatal(err)
		}
		_, err := data.WriteMessage(ctx, Message{
			ClientMsgID: uuid.NewV7(),
			SenderID:    userID,
			RoomID:      roomID,
			MsgType:     MsgTypeText,
			Payload:     []byte(`{"text":"hello"}`),
		})
		if err != nil {
			b.Fatal(err)
		}
		data.MarkRoomRead(ctx, userID, roomID, 1)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		convs, err := data.Conversations(ctx, userID)
		if err != nil || len(convs) != 50 {
			b.Fatalf("conversations: count=%d, err=%v", len(convs), err)
		}
	}
}

func BenchmarkRoomMembers500(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	roomID := uuid.NewV7()
	ownerID := uuid.NewV7()
	members := make([]Member, 500)
	members[0] = Member{UserID: ownerID, Role: "owner"}
	for i := 1; i < 500; i++ {
		members[i] = Member{UserID: uuid.NewV7(), Role: "member"}
	}
	if err := data.CreateRoom(ctx, Room{RoomID: roomID, ChatType: "group"}, members); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		list, err := data.Members(ctx, roomID)
		if err != nil || len(list) != 500 {
			b.Fatalf("members: count=%d, err=%v", len(list), err)
		}
	}
}

func BenchmarkRoomMembersParallel(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	roomID := uuid.NewV7()
	ownerID := uuid.NewV7()
	members := make([]Member, 200)
	members[0] = Member{UserID: ownerID, Role: "owner"}
	for i := 1; i < 200; i++ {
		members[i] = Member{UserID: uuid.NewV7(), Role: "member"}
	}
	if err := data.CreateRoom(ctx, Room{RoomID: roomID, ChatType: "group"}, members); err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			list, err := data.Members(ctx, roomID)
			if err != nil || len(list) != 200 {
				b.Fatalf("members: count=%d, err=%v", len(list), err)
			}
		}
	})
}

func BenchmarkMarkRoomRead(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	userID := uuid.NewV7()
	roomID := uuid.NewV7()

	b.ReportAllocs()
	b.ResetTimer()
	var seq uint64
	for b.Loop() {
		seq++
		if err := data.MarkRoomRead(ctx, userID, roomID, seq); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReactionsAggregation(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	roomID := uuid.NewV7()
	msgID := uuid.NewV7()
	emojis := []string{"👍", "❤️", "😂", "🎉", "🔥"}

	for i := range 100 {
		user := uuid.NewV7()
		emoji := emojis[i%len(emojis)]
		if err := data.AddReaction(ctx, roomID, msgID, user, emoji); err != nil {
			b.Fatal(err)
		}
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		reactions, err := data.Reactions(ctx, roomID, msgID)
		if err != nil || len(reactions) != len(emojis) {
			b.Fatalf("reactions: count=%d, err=%v", len(reactions), err)
		}
	}
}

func BenchmarkReadUsers100Members(b *testing.B) {
	data, err := Open(b.TempDir() + "/store")
	if err != nil {
		b.Fatal(err)
	}
	defer data.Close()

	ctx := context.Background()
	roomID := uuid.NewV7()
	ownerID := uuid.NewV7()
	members := make([]Member, 100)
	members[0] = Member{UserID: ownerID, Role: "owner"}
	for i := 1; i < 100; i++ {
		members[i] = Member{UserID: uuid.NewV7(), Role: "member"}
	}
	if err := data.CreateRoom(ctx, Room{RoomID: roomID, ChatType: "group"}, members); err != nil {
		b.Fatal(err)
	}

	msg, err := data.WriteMessage(ctx, Message{
		ClientMsgID: uuid.NewV7(),
		SenderID:    ownerID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     []byte("test"),
	})
	if err != nil {
		b.Fatal(err)
	}

	for i := range 80 {
		data.MarkRoomRead(ctx, members[i].UserID, roomID, msg.RoomSeq)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		users, err := data.ReadUsers(ctx, roomID, msg.MsgID)
		if err != nil || len(users) != 80 {
			b.Fatalf("read users: count=%d, err=%v", len(users), err)
		}
	}
}

func TestFriendAndBlacklist(t *testing.T) {
	ctx := context.Background()
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	u1, err := data.CreateUser(ctx, "alice", "pass123")
	if err != nil {
		t.Fatal(err)
	}
	u2, err := data.CreateUser(ctx, "bob", "pass456")
	if err != nil {
		t.Fatal(err)
	}

	// Update profile & Search
	updated, err := data.UpdateUserProfile(ctx, u1.UserID, "AliceInWonderland", "https://avatar.com/1")
	if err != nil || updated.Nickname != "AliceInWonderland" {
		t.Fatalf("update profile failed: %+v err: %v", updated, err)
	}
	searched, err := data.SearchUsers(ctx, "wonder", 10)
	if err != nil || len(searched) != 1 || searched[0].UserID != u1.UserID {
		t.Fatalf("search users failed: %+v err: %v", searched, err)
	}

	// Apply friend
	if err := data.ApplyFriend(ctx, u1.UserID, u2.UserID, "Hello, add me"); err != nil {
		t.Fatal(err)
	}
	apps, err := data.Applications(ctx, u2.UserID)
	if err != nil || len(apps) != 1 || apps[0].Greeting != "Hello, add me" {
		t.Fatalf("applications failed: %+v err: %v", apps, err)
	}

	// Audit friend (accept)
	if err := data.AuditFriend(ctx, u2.UserID, u1.UserID, true); err != nil {
		t.Fatal(err)
	}
	f1, err := data.Friends(ctx, u1.UserID)
	if err != nil || len(f1) != 1 || f1[0].FriendID != u2.UserID {
		t.Fatalf("u1 friends failed: %+v err: %v", f1, err)
	}
	f2, err := data.Friends(ctx, u2.UserID)
	if err != nil || len(f2) != 1 || f2[0].FriendID != u1.UserID {
		t.Fatalf("u2 friends failed: %+v err: %v", f2, err)
	}

	// Update remark
	if err := data.UpdateFriendRemark(ctx, u1.UserID, u2.UserID, "Bobby"); err != nil {
		t.Fatal(err)
	}
	f1, _ = data.Friends(ctx, u1.UserID)
	if f1[0].Remark != "Bobby" {
		t.Fatalf("expected remark Bobby, got %s", f1[0].Remark)
	}

	// Blacklist
	if err := data.AddBlacklist(ctx, u1.UserID, u2.UserID); err != nil {
		t.Fatal(err)
	}
	isBlk, err := data.IsBlacklisted(ctx, u1.UserID, u2.UserID)
	if err != nil || !isBlk {
		t.Fatalf("expected blacklisted, got %v err: %v", isBlk, err)
	}
	if err := data.RemoveBlacklist(ctx, u1.UserID, u2.UserID); err != nil {
		t.Fatal(err)
	}
	isBlk, _ = data.IsBlacklisted(ctx, u1.UserID, u2.UserID)
	if isBlk {
		t.Fatal("expected not blacklisted")
	}

	// Delete friend
	if err := data.DeleteFriend(ctx, u1.UserID, u2.UserID); err != nil {
		t.Fatal(err)
	}
	f1, _ = data.Friends(ctx, u1.UserID)
	if len(f1) != 0 {
		t.Fatalf("expected 0 friends after delete, got %d", len(f1))
	}
}

func TestGroupManagementAndConversations(t *testing.T) {
	ctx := context.Background()
	data, err := Open(t.TempDir() + "/store")
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	u1, _ := data.CreateUser(ctx, "user1", "pass")
	u2, _ := data.CreateUser(ctx, "user2", "pass")
	u3, _ := data.CreateUser(ctx, "user3", "pass")

	roomID := uuid.NewV7()
	if err := data.CreateRoom(ctx, Room{
		RoomID: roomID, ChatType: "group", Name: "Team Alpha",
	}, []Member{
		{UserID: u1.UserID, Role: "owner"},
		{UserID: u2.UserID, Role: "member"},
	}); err != nil {
		t.Fatal(err)
	}

	// Add member
	if err := data.AddMembers(ctx, roomID, []Member{{UserID: u3.UserID, Role: "member"}}); err != nil {
		t.Fatal(err)
	}
	members, err := data.Members(ctx, roomID)
	if err != nil || len(members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(members))
	}

	// Update room
	updatedRoom, err := data.UpdateRoom(ctx, roomID, "Team Beta", "https://avatar", "No spam")
	if err != nil || updatedRoom.Name != "Team Beta" || updatedRoom.Notice != "No spam" {
		t.Fatalf("update room failed: %+v err: %v", updatedRoom, err)
	}

	// Send message
	msg, err := data.WriteMessage(ctx, Message{
		ClientMsgID: uuid.NewV7(),
		SenderID:    u1.UserID,
		RoomID:      roomID,
		MsgType:     MsgTypeText,
		Payload:     []byte(`"Hello world"`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Conversations for u2
	convs, err := data.Conversations(ctx, u2.UserID)
	if err != nil || len(convs) != 1 {
		t.Fatalf("expected 1 conversation, got %d err: %v", len(convs), err)
	}
	if convs[0].UnreadCount != 1 || convs[0].LastMessage == nil || convs[0].LastMessage.MsgID != msg.MsgID {
		t.Fatalf("unexpected conversation: %+v", convs[0])
	}

	// Mark read
	if err := data.MarkRoomRead(ctx, u2.UserID, roomID, 1); err != nil {
		t.Fatal(err)
	}
	convs, _ = data.Conversations(ctx, u2.UserID)
	if convs[0].UnreadCount != 0 {
		t.Fatalf("expected 0 unread, got %d", convs[0].UnreadCount)
	}

	// Pin conversation
	pinned := true
	if err := data.UpdateMemberSettings(ctx, roomID, u2.UserID, &pinned, nil); err != nil {
		t.Fatal(err)
	}
	convs, _ = data.Conversations(ctx, u2.UserID)
	if !convs[0].Member.IsPinned {
		t.Fatal("expected conversation to be pinned")
	}

	// Remove member
	if err := data.RemoveMember(ctx, roomID, u3.UserID); err != nil {
		t.Fatal(err)
	}
	members, _ = data.Members(ctx, roomID)
	if len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	// Dissolve room
	if err := data.DissolveRoom(ctx, roomID); err != nil {
		t.Fatal(err)
	}
	convs, _ = data.Conversations(ctx, u1.UserID)
	if len(convs) != 0 {
		t.Fatalf("expected 0 conversations after dissolve, got %d", len(convs))
	}
}

func TestP1StoreFeatures(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	data, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()

	u1, err := data.CreateUser(ctx, "user1", "pwd")
	if err != nil {
		t.Fatal(err)
	}
	u2, err := data.CreateUser(ctx, "user2", "pwd")
	if err != nil {
		t.Fatal(err)
	}
	u3, err := data.CreateUser(ctx, "user3", "pwd")
	if err != nil {
		t.Fatal(err)
	}

	roomID := uuid.NewV7()
	if err := data.CreateRoom(ctx, Room{
		RoomID: roomID, ChatType: "group", Name: "Test Room",
	}, []Member{
		{UserID: u1.UserID, Role: "owner"},
		{UserID: u2.UserID, Role: "member"},
		{UserID: u3.UserID, Role: "member"},
	}); err != nil {
		t.Fatal(err)
	}

	// 1. MessageByID and Recall
	msg, err := data.WriteMessage(ctx, Message{
		RoomID:      roomID,
		SenderID:    u2.UserID,
		ClientMsgID: uuid.NewV7(),
		MsgType:     MsgTypeText,
		Payload:     jsontext.Value(`{"text":"hello"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	found, err := data.MessageByID(ctx, roomID, msg.MsgID)
	if err != nil {
		t.Fatalf("MessageByID failed: %v", err)
	}
	if found.RoomSeq != msg.RoomSeq {
		t.Fatalf("expected seq %d, got %d", msg.RoomSeq, found.RoomSeq)
	}

	// Unauthorized recall
	_, err = data.RecallMessage(ctx, roomID, msg.MsgID, u3.UserID, false)
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	// Sender recall within timeout
	recalled, err := data.RecallMessage(ctx, roomID, msg.MsgID, u2.UserID, false)
	if err != nil {
		t.Fatalf("RecallMessage failed: %v", err)
	}
	if recalled.MsgType != MsgTypeRecall {
		t.Fatalf("expected MsgTypeRecall, got %v", recalled.MsgType)
	}

	// Test recall timeout
	oldMsg, err := data.WriteMessage(ctx, Message{
		RoomID:      roomID,
		SenderID:    u2.UserID,
		ClientMsgID: uuid.NewV7(),
		MsgType:     MsgTypeText,
		Payload:     jsontext.Value(`{"text":"old"}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Manually simulate old timestamp by overwriting message record
	oldMsg.ServerTime = 100 // ancient microsecond timestamp
	var keyBuf [25]byte
	msgKey := appendMessageKey(keyBuf[:0], roomID, oldMsg.RoomSeq)
	batch := data.db.NewBatch()
	_ = data.setBytes(batch, msgKey, encodeMessage(oldMsg))
	_ = commit(batch)
	batch.Close()

	// Sender tries recall ancient message -> ErrRecallTimeout
	_, err = data.RecallMessage(ctx, roomID, oldMsg.MsgID, u2.UserID, false)
	if !errors.Is(err, ErrRecallTimeout) {
		t.Fatalf("expected ErrRecallTimeout, got %v", err)
	}

	// Group owner recalls ancient message -> success
	recalledByOwner, err := data.RecallMessage(ctx, roomID, oldMsg.MsgID, u1.UserID, true)
	if err != nil {
		t.Fatalf("owner recall failed: %v", err)
	}
	if recalledByOwner.MsgType != MsgTypeRecall {
		t.Fatalf("expected MsgTypeRecall, got %v", recalledByOwner.MsgType)
	}

	// 2. Reactions
	if err := data.AddReaction(ctx, roomID, msg.MsgID, u1.UserID, "👍"); err != nil {
		t.Fatal(err)
	}
	if err := data.AddReaction(ctx, roomID, msg.MsgID, u2.UserID, "👍"); err != nil {
		t.Fatal(err)
	}
	if err := data.AddReaction(ctx, roomID, msg.MsgID, u1.UserID, "❤️"); err != nil {
		t.Fatal(err)
	}

	reactions, err := data.Reactions(ctx, roomID, msg.MsgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(reactions) != 2 {
		t.Fatalf("expected 2 reaction groups, got %d", len(reactions))
	}
	for _, r := range reactions {
		if r.Emoji == "👍" && r.Count != 2 {
			t.Fatalf("expected 2 thumbs up, got %d", r.Count)
		}
	}

	// Remove reaction
	if err := data.RemoveReaction(ctx, roomID, msg.MsgID, u2.UserID, "👍"); err != nil {
		t.Fatal(err)
	}
	reactions, _ = data.Reactions(ctx, roomID, msg.MsgID)
	for _, r := range reactions {
		if r.Emoji == "👍" && r.Count != 1 {
			t.Fatalf("expected 1 thumbs up after removal, got %d", r.Count)
		}
	}

	// 3. ReadUsers
	// msg seq is 1, oldMsg seq is 2
	if err := data.MarkRoomRead(ctx, u1.UserID, roomID, 2); err != nil {
		t.Fatal(err)
	}
	if err := data.MarkRoomRead(ctx, u3.UserID, roomID, 1); err != nil {
		t.Fatal(err)
	}
	// msg seq 1 should be read by u1 (read 2) and u3 (read 1)
	readUsers1, err := data.ReadUsers(ctx, roomID, msg.MsgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(readUsers1) != 2 {
		t.Fatalf("expected 2 read users for msg 1, got %d", len(readUsers1))
	}

	// oldMsg seq 2 should only be read by u1 (read 2)
	readUsers2, err := data.ReadUsers(ctx, roomID, oldMsg.MsgID)
	if err != nil {
		t.Fatal(err)
	}
	if len(readUsers2) != 1 || readUsers2[0] != u1.UserID {
		t.Fatalf("expected u1 read msg 2, got %v", readUsers2)
	}

	// 4. PinMessage / UnpinMessage / PinnedMessages
	if err := data.PinMessage(ctx, roomID, msg.MsgID, u1.UserID); err != nil {
		t.Fatal(err)
	}
	pins, err := data.PinnedMessages(ctx, roomID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pins) != 1 || pins[0].MsgID != msg.MsgID {
		t.Fatalf("expected pinned message, got %v", pins)
	}
	if err := data.UnpinMessage(ctx, roomID, msg.MsgID); err != nil {
		t.Fatal(err)
	}
	pins, _ = data.PinnedMessages(ctx, roomID)
	if len(pins) != 0 {
		t.Fatalf("expected 0 pinned messages after unpin, got %d", len(pins))
	}

	// 5. DeviceToken
	if err := data.SaveDeviceToken(ctx, u1.UserID, DeviceInfo{
		Token:    "apns-token-xyz",
		Platform: "ios",
	}); err != nil {
		t.Fatal(err)
	}
	dev, err := data.DeviceToken(ctx, u1.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if dev.Token != "apns-token-xyz" || dev.Platform != "ios" {
		t.Fatalf("device info mismatch: %+v", dev)
	}
	if err := data.DeleteDeviceToken(ctx, u1.UserID); err != nil {
		t.Fatal(err)
	}
	_, err = data.DeviceToken(ctx, u1.UserID)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deleting device token, got %v", err)
	}
}
