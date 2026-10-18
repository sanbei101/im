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
		updated time.Time
		members []Member
	}{
		{roomA, time.Unix(100, 0), []Member{{UserID: alice.UserID, Role: RoleOwner}, {UserID: bob.UserID, Role: RoleMember}}},
		{roomB, time.Unix(300, 0), []Member{{UserID: alice.UserID, Role: RoleOwner}}},
		{roomC, time.Unix(200, 0), []Member{{UserID: alice.UserID, Role: RoleOwner}}},
	} {
		if err := db.CreateRoom(ctx, Room{
			RoomID: tc.room, ChatType: ChatTypeGroup, Name: "room",
			CreatedAt: tc.updated, UpdatedAt: tc.updated,
		}, tc.members); err != nil {
			t.Fatal(err)
		}
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
	if conversations[0].Room.RoomID != roomB || conversations[1].Room.RoomID != roomC ||
		conversations[2].Room.RoomID != roomA {
		t.Fatalf("conversation order = %s, %s, %s", conversations[0].Room.RoomID,
			conversations[1].Room.RoomID, conversations[2].Room.RoomID)
	}
	if conversations[0].ReadSeq != 5 || conversations[1].ReadSeq != 0 || conversations[2].ReadSeq != 0 {
		t.Fatalf("read cursors = %d, %d, %d", conversations[0].ReadSeq,
			conversations[1].ReadSeq, conversations[2].ReadSeq)
	}

	bobRooms, err := db.Conversations(ctx, bob.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bobRooms) != 1 || bobRooms[0].Room.RoomID != roomA || bobRooms[0].ReadSeq != 0 {
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
	for range rooms {
		if err := db.CreateRoom(ctx, Room{
			RoomID: uuid.NewV7(), ChatType: ChatTypeGroup, Name: "bench-room",
		}, []Member{{UserID: user.UserID, Role: RoleOwner}}); err != nil {
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

func BenchmarkDBConversationListParallel(b *testing.B) {
	db := newPGTestDB(b)
	ctx := context.Background()
	user, err := db.CreateUser(ctx, "bench-conversations-parallel", "hash")
	if err != nil {
		b.Fatal(err)
	}
	const rooms = 100
	for range rooms {
		if err := db.CreateRoom(ctx, Room{
			RoomID: uuid.NewV7(), ChatType: ChatTypeGroup, Name: "bench-room",
		}, []Member{{UserID: user.UserID, Role: RoleOwner}}); err != nil {
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
