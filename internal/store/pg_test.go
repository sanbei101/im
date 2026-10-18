package store

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
	"uuid"
)

func pgTestURL() string {
	if value := os.Getenv("TEST_POSTGRES_URL"); value != "" {
		return value
	}
	return "postgres://127.0.0.1:5432/im_test?sslmode=disable"
}

// newPGTestDB opens the shared test database and clears control-plane tables.
// PostgreSQL tests and benchmarks are serialized by the Makefile.
func newPGTestDB(tb testing.TB) *DB {
	tb.Helper()
	ctx := context.Background()
	db, err := OpenDB(ctx, pgTestURL())
	if err != nil {
		tb.Fatalf("open postgres: %v (run `make test` first)", err)
	}
	tb.Cleanup(db.Close)
	if _, err := db.pool.Exec(ctx, `TRUNCATE users, rooms CASCADE`); err != nil {
		tb.Fatalf("truncate postgres: %v", err)
	}
	return db
}

func TestDBConversations(t *testing.T) {
	db := newPGTestDB(t)
	ctx := context.Background()

	alice, err := db.CreateUser(ctx, "alice", "hash")
	if err != nil {
		t.Fatal(err)
	}
	bob, err := db.CreateUser(ctx, "bob", "hash")
	if err != nil {
		t.Fatal(err)
	}
	carol, err := db.CreateUser(ctx, "carol", "hash")
	if err != nil {
		t.Fatal(err)
	}

	roomA := uuid.NewV7()
	roomB := uuid.NewV7()
	roomC := uuid.NewV7()
	for _, tc := range []struct {
		room    uuid.UUID
		members []Member
	}{
		{roomA, []Member{{UserID: alice.UserID, Role: RoleOwner}, {UserID: bob.UserID, Role: RoleMember}}},
		{roomB, []Member{{UserID: alice.UserID, Role: RoleOwner}}},
		{roomC, []Member{{UserID: alice.UserID, Role: RoleOwner}}},
	} {
		if err := db.CreateRoom(
			ctx,
			Room{RoomID: tc.room, ChatType: ChatTypeGroup, Name: "room"},
			tc.members,
		); err != nil {
			t.Fatal(err)
		}
	}

	newest := &Message{
		MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: alice.UserID,
		RoomID: roomA, RoomSeq: 42, ServerTime: 200, MsgType: MsgTypeText,
		Payload: []byte(`{"text":"newest"}`),
	}
	if err := db.UpdateRoomHead(ctx, roomA, 42, newest); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateRoomHead(ctx, roomB, 5, &Message{
		MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: alice.UserID,
		RoomID: roomB, RoomSeq: 5, ServerTime: 100, MsgType: MsgTypeText,
		Payload: []byte(`{"text":"older"}`),
	}); err != nil {
		t.Fatal(err)
	}
	// A lower sequence must not roll the projection back.
	if err := db.UpdateRoomHead(ctx, roomA, 41, &Message{
		MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: alice.UserID,
		RoomID: roomA, RoomSeq: 41, ServerTime: 999, MsgType: MsgTypeText,
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkRead(ctx, alice.UserID, roomA, 40); err != nil {
		t.Fatal(err)
	}
	if err := db.MarkRead(ctx, alice.UserID, roomB, 5); err != nil {
		t.Fatal(err)
	}

	conversations, err := db.Conversations(ctx, alice.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(conversations) != 3 {
		t.Fatalf("conversations = %d, want 3", len(conversations))
	}
	if conversations[0].Room.RoomID != roomA || conversations[1].Room.RoomID != roomB {
		t.Fatalf("conversation order = %s, %s; want %s, %s",
			conversations[0].Room.RoomID, conversations[1].Room.RoomID, roomA, roomB)
	}
	if conversations[0].UnreadCount != 2 || conversations[0].LastMessage == nil ||
		conversations[0].LastMessage.MsgID != newest.MsgID {
		t.Fatalf("newest conversation = %+v", conversations[0])
	}
	if conversations[1].UnreadCount != 0 || conversations[1].LastMessage == nil ||
		conversations[1].LastMessage.RoomSeq != 5 {
		t.Fatalf("read conversation = %+v", conversations[1])
	}
	if conversations[2].Room.RoomID != roomC || conversations[2].LastMessage != nil ||
		conversations[2].UnreadCount != 0 {
		t.Fatalf("headless conversation = %+v", conversations[2])
	}

	bobRooms, err := db.Conversations(ctx, bob.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bobRooms) != 1 || bobRooms[0].Room.RoomID != roomA || bobRooms[0].UnreadCount != 42 {
		t.Fatalf("bob conversations = %+v", bobRooms)
	}
	carolRooms, err := db.Conversations(ctx, carol.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(carolRooms) != 0 {
		t.Fatalf("carol conversations = %d, want 0", len(carolRooms))
	}
}

func BenchmarkDBConversationList(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-conversations", "hash")
	if err != nil {
		b.Fatal(err)
	}
	const rooms = 100
	for i := range rooms {
		roomID := uuid.NewV7()
		if err := db.CreateRoom(ctx, Room{
			RoomID: roomID, ChatType: ChatTypeGroup, Name: "bench-room",
		}, []Member{{UserID: user.UserID, Role: RoleOwner}}); err != nil {
			b.Fatal(err)
		}
		if err := db.UpdateRoomHead(ctx, roomID, uint64(i+1), &Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: user.UserID,
			RoomID: roomID, RoomSeq: uint64(i + 1), ServerTime: int64(time.Now().UnixMicro()),
			MsgType: MsgTypeText, Payload: []byte(`{"text":"benchmark"}`),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		conversations, err := db.Conversations(ctx, user.UserID)
		if err != nil || len(conversations) != rooms {
			b.Fatalf("conversations=%d err=%v", len(conversations), err)
		}
	}
}

func BenchmarkDBRoomHead(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-head", "hash")
	if err != nil {
		b.Fatal(err)
	}
	roomID := uuid.NewV7()
	if err := db.CreateRoom(ctx, Room{RoomID: roomID, ChatType: ChatTypeGroup}, []Member{{
		UserID: user.UserID, Role: RoleOwner,
	}}); err != nil {
		b.Fatal(err)
	}
	if err := db.UpdateRoomHead(ctx, roomID, 1, &Message{
		MsgID: uuid.NewV7(), RoomID: roomID, RoomSeq: 1, ServerTime: 1, MsgType: MsgTypeText,
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := db.RoomHead(ctx, roomID); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBMarkRead(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-read", "hash")
	if err != nil {
		b.Fatal(err)
	}
	roomID := uuid.NewV7()
	if err := db.CreateRoom(ctx, Room{RoomID: roomID, ChatType: ChatTypeGroup}, []Member{{
		UserID: user.UserID, Role: RoleOwner,
	}}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	for seq := uint64(1); b.Loop(); seq++ {
		if err := db.MarkRead(ctx, user.UserID, roomID, seq); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBSearchUsers(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	const users = 1000
	for i := range users {
		if _, err := db.CreateUser(ctx, "search-user-"+strconv.Itoa(i), "hash"); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		found, err := db.SearchUsers(ctx, "user-5", 20)
		if err != nil || len(found) == 0 {
			b.Fatalf("search users: found=%d err=%v", len(found), err)
		}
	}
}

func BenchmarkDBUsersByIDs(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	ids := make([]uuid.UUID, 200)
	for i := range ids {
		user, err := db.CreateUser(ctx, "lookup-user-"+strconv.Itoa(i), "hash")
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = user.UserID
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		users, err := db.UsersByIDs(ctx, ids)
		if err != nil || len(users) != len(ids) {
			b.Fatalf("load users: count=%d err=%v", len(users), err)
		}
	}
}

func BenchmarkDBConversationListParallel(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-conversations-parallel", "hash")
	if err != nil {
		b.Fatal(err)
	}
	const rooms = 100
	for i := range rooms {
		roomID := uuid.NewV7()
		if err := db.CreateRoom(ctx, Room{
			RoomID: roomID, ChatType: ChatTypeGroup, Name: "bench-room",
		}, []Member{{UserID: user.UserID, Role: RoleOwner}}); err != nil {
			b.Fatal(err)
		}
		if err := db.UpdateRoomHead(ctx, roomID, uint64(i+1), &Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: user.UserID,
			RoomID: roomID, RoomSeq: uint64(i + 1), ServerTime: int64(i + 1),
			MsgType: MsgTypeText, Payload: []byte(`{"text":"benchmark"}`),
		}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			conversations, err := db.Conversations(ctx, user.UserID)
			if err != nil || len(conversations) != rooms {
				b.Errorf("conversations=%d err=%v", len(conversations), err)
				return
			}
		}
	})
}

func BenchmarkDBMarkReadParallel(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	const users = 64
	type target struct {
		user uuid.UUID
		room uuid.UUID
	}
	targets := make([]target, users)
	for i := range targets {
		user, err := db.CreateUser(ctx, "bench-read-parallel-"+strconv.Itoa(i), "hash")
		if err != nil {
			b.Fatal(err)
		}
		roomID := uuid.NewV7()
		if err := db.CreateRoom(ctx, Room{RoomID: roomID, ChatType: ChatTypeGroup}, []Member{{
			UserID: user.UserID, Role: RoleOwner,
		}}); err != nil {
			b.Fatal(err)
		}
		targets[i] = target{user: user.UserID, room: roomID}
	}

	var next atomic.Uint64
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			target := targets[next.Add(1)%users]
			if err := db.MarkRead(ctx, target.user, target.room, 1); err != nil {
				b.Errorf("mark read: %v", err)
				return
			}
		}
	})
}

func BenchmarkDBUpdateRoomHead(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-head-write", "hash")
	if err != nil {
		b.Fatal(err)
	}
	roomID := uuid.NewV7()
	if err := db.CreateRoom(ctx, Room{RoomID: roomID, ChatType: ChatTypeGroup}, []Member{{
		UserID: user.UserID, Role: RoleOwner,
	}}); err != nil {
		b.Fatal(err)
	}
	message := &Message{
		MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: user.UserID,
		RoomID: roomID, ServerTime: 1, MsgType: MsgTypeText,
		Payload: []byte(`{"text":"benchmark"}`),
	}

	b.ResetTimer()
	b.ReportAllocs()
	for seq := uint64(1); b.Loop(); seq++ {
		message.RoomSeq = seq
		if err := db.UpdateRoomHead(ctx, roomID, seq, message); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDBUpdateRoomHeadsBatch(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-head-batch", "hash")
	if err != nil {
		b.Fatal(err)
	}
	const rooms = 32
	updates := make([]RoomHeadUpdate, rooms)
	for i := range updates {
		roomID := uuid.NewV7()
		if err := db.CreateRoom(ctx, Room{RoomID: roomID, ChatType: ChatTypeGroup}, []Member{{
			UserID: user.UserID, Role: RoleOwner,
		}}); err != nil {
			b.Fatal(err)
		}
		updates[i] = RoomHeadUpdate{
			RoomID: roomID, Seq: 1,
			Message: &Message{
				MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: user.UserID,
				RoomID: roomID, RoomSeq: 1, ServerTime: 1, MsgType: MsgTypeText,
				Payload: []byte(`{"text":"benchmark"}`),
			},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		if err := db.UpdateRoomHeads(ctx, updates); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStoreMessageWriteWithProjection(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	owner, err := db.CreateUser(ctx, "bench-projected-owner", "hash")
	if err != nil {
		b.Fatal(err)
	}
	roomID := uuid.NewV7()
	if err := db.CreateRoom(ctx, Room{RoomID: roomID, ChatType: ChatTypeGroup}, []Member{{
		UserID: owner.UserID, Role: RoleOwner,
	}}); err != nil {
		b.Fatal(err)
	}
	data := newBenchStore(b)
	data.AttachDB(db)
	sender := uuid.NewV7()
	payload := []byte(`{"text":"benchmark payload"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		messages := make([]Message, 100)
		for i := range messages {
			messages[i] = Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: roomID,
				MsgType: MsgTypeText, Payload: payload,
			}
		}
		results := data.WriteMessages(ctx, messages)
		for i := range results {
			if results[i].Err != nil {
				b.Fatal(results[i].Err)
			}
		}
	}
}

func TestUpdateRoomHeadsSkipsDeletedRoom(t *testing.T) {
	db := newPGTestDB(t)
	ctx := context.Background()
	owner, err := db.CreateUser(ctx, "head-deleted-owner", "hash")
	if err != nil {
		t.Fatal(err)
	}
	liveRoom := uuid.NewV7()
	if err := db.CreateRoom(ctx, Room{RoomID: liveRoom, ChatType: ChatTypeGroup}, []Member{{
		UserID: owner.UserID, Role: RoleOwner,
	}}); err != nil {
		t.Fatal(err)
	}
	message := &Message{
		MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: owner.UserID,
		RoomID: liveRoom, RoomSeq: 1, ServerTime: 1, MsgType: MsgTypeText,
	}
	err = db.UpdateRoomHeads(ctx, []RoomHeadUpdate{
		{RoomID: uuid.NewV7(), Seq: 1, Message: message},
		{RoomID: liveRoom, Seq: 1, Message: message},
	})
	if err != nil {
		t.Fatalf("batch with deleted room failed: %v", err)
	}
	head, err := db.RoomHead(ctx, liveRoom)
	if err != nil || head.LastSeq != 1 {
		t.Fatalf("live room head = %+v, err = %v", head, err)
	}
}

func BenchmarkStoreMessageWriteWithProjectionManyRooms(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	owner, err := db.CreateUser(ctx, "bench-projected-many", "hash")
	if err != nil {
		b.Fatal(err)
	}
	const rooms = 100
	roomIDs := make([]uuid.UUID, rooms)
	for i := range roomIDs {
		roomIDs[i] = uuid.NewV7()
		if err := db.CreateRoom(ctx, Room{RoomID: roomIDs[i], ChatType: ChatTypeGroup}, []Member{{
			UserID: owner.UserID, Role: RoleOwner,
		}}); err != nil {
			b.Fatal(err)
		}
	}
	data := newBenchStore(b)
	data.AttachDB(db)
	sender := uuid.NewV7()
	payload := []byte(`{"text":"benchmark payload"}`)

	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		messages := make([]Message, rooms)
		for i := range messages {
			messages[i] = Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: roomIDs[i],
				MsgType: MsgTypeText, Payload: payload,
			}
		}
		results := data.WriteMessages(ctx, messages)
		for i := range results {
			if results[i].Err != nil {
				b.Fatal(results[i].Err)
			}
		}
	}
}
