package store

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
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
