package engine

import (
	"context"
	"encoding/json/jsontext"
	"os"
	"path/filepath"
	"testing"
	"uuid"
)

// migrateOwnerFixture splits rooms by the parity of their last byte: even
// rooms stay on the source engine, odd rooms move to destination index 0.
func migrateOwnerFixture(room uuid.UUID) int {
	if room[15]%2 == 0 {
		return -1
	}
	return 0
}

// fixtureRoom returns a fresh room id with the wanted last-byte parity.
func fixtureRoom(t *testing.T, want int) uuid.UUID {
	t.Helper()
	for range 64 {
		room := uuid.NewV7()
		if int(room[15]%2) == want%2 {
			return room
		}
	}
	t.Fatal("no room id matched the wanted parity")
	return uuid.Nil()
}

// seedRooms writes an archived prefix plus a live tail per room, with a pin,
// a reaction and a read cursor on the first live message, so every key family
// has data to move. It returns the writer and the marked (pinned) seq.
func seedRooms(t *testing.T, e *Engine, rooms ...uuid.UUID) (uuid.UUID, uint64) {
	t.Helper()
	ctx := context.Background()
	sender := uuid.NewV7()
	for _, room := range rooms {
		for range 5 {
			mustWriteMessage(t, e, room, sender, "cold")
		}
	}
	if err := e.ArchiveOnce(ctx, instantArchive(10)); err != nil {
		t.Fatalf("archive: %v", err)
	}
	var marked uint64
	for _, room := range rooms {
		for i := range 3 {
			message := mustWriteMessage(t, e, room, sender, "warm")
			if i == 0 {
				marked = message.RoomSeq
				if err := e.PinMessage(ctx, room, message.RoomSeq, sender); err != nil {
					t.Fatalf("pin: %v", err)
				}
				if err := e.AddReaction(ctx, room, message.RoomSeq, sender, "👍"); err != nil {
					t.Fatalf("react: %v", err)
				}
				if err := e.MarkRead(ctx, room, sender, message.RoomSeq); err != nil {
					t.Fatalf("mark read: %v", err)
				}
			}
		}
	}
	return sender, marked
}

func verifyMigratedRoom(t *testing.T, e *Engine, room, sender uuid.UUID, marked uint64, label string) {
	t.Helper()
	ctx := context.Background()
	page, err := e.Messages(ctx, room, 0, 100)
	if err != nil {
		t.Fatalf("%s: read messages: %v", label, err)
	}
	if len(page.Messages) != 8 {
		t.Fatalf("%s: messages = %d, want 8 (archived prefix + live tail)", label, len(page.Messages))
	}
	if seq, err := e.LastSeq(ctx, room); err != nil || seq != 8 {
		t.Fatalf("%s: last seq = %d, %v; want 8", label, seq, err)
	}
	pins, err := e.PinnedMessages(ctx, room)
	if err != nil || len(pins) != 1 || pins[0].RoomSeq != marked {
		t.Fatalf("%s: pins = %v, %v; want the marked message", label, pins, err)
	}
	groups, err := e.Reactions(ctx, room, marked)
	if err != nil || len(groups) != 1 {
		t.Fatalf("%s: reactions = %v, %v; want one group", label, groups, err)
	}
	read, err := e.ReadSeq(ctx, room, sender)
	if err != nil || read != marked {
		t.Fatalf("%s: read cursor = %d, %v; want %d", label, read, err, marked)
	}
}

func TestMigrate(t *testing.T) {
	t.Run("dry run reports without writing", func(t *testing.T) {
		ctx := context.Background()
		src := newTestEngine(t, ArchiveDir(t.TempDir()+"/archive"))
		dst := newTestEngine(t, ArchiveDir(t.TempDir()+"/archive"))
		move := fixtureRoom(t, 1)
		sender, marked := seedRooms(t, src, move)

		stats, err := src.Migrate(ctx, []*Engine{dst}, MigrateParams{
			Owner: migrateOwnerFixture, DryRun: true,
		})
		if err != nil {
			t.Fatalf("dry-run migrate: %v", err)
		}
		if stats.Keys == 0 || stats.Files == 0 {
			t.Fatalf("dry-run stats = %+v, want moved keys and files", stats)
		}
		verifyMigratedRoom(t, src, move, sender, marked, "source after dry-run")
		if page, err := dst.Messages(ctx, move, 0, 100); err != nil || len(page.Messages) != 0 {
			t.Fatalf("destination changed by dry-run: %d messages, %v", len(page.Messages), err)
		}
	})

	t.Run("moves every family and archive file", func(t *testing.T) {
		ctx := context.Background()
		src := newTestEngine(t, ArchiveDir(t.TempDir()+"/archive"))
		dst := newTestEngine(t, ArchiveDir(t.TempDir()+"/archive"))
		keep := fixtureRoom(t, 0)
		move := fixtureRoom(t, 1)
		sender, marked := seedRooms(t, src, keep, move)

		stats, err := src.Migrate(ctx, []*Engine{dst}, MigrateParams{Owner: migrateOwnerFixture})
		if err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if stats.Keys == 0 || stats.Files != 1 {
			t.Fatalf("stats = %+v, want keys moved and exactly one archive file", stats)
		}

		verifyMigratedRoom(t, dst, move, sender, marked, "destination")
		verifyMigratedRoom(t, src, keep, sender, marked, "source kept room")

		if page, err := src.Messages(ctx, move, 0, 100); err != nil || len(page.Messages) != 0 {
			t.Fatalf("moved room still readable on source: %d messages, %v", len(page.Messages), err)
		}
		if _, err := os.Stat(filepath.Join(src.archiveDir, "rooms", move.String())); err == nil {
			t.Fatal("moved room's archive directory still exists on source")
		}
		if _, err := os.Stat(filepath.Join(src.archiveDir, "rooms", keep.String())); err != nil {
			t.Fatalf("kept room's archive directory missing on source: %v", err)
		}
	})

	t.Run("second pass moves nothing", func(t *testing.T) {
		ctx := context.Background()
		src := newTestEngine(t, ArchiveDir(t.TempDir()+"/archive"))
		dst := newTestEngine(t, ArchiveDir(t.TempDir()+"/archive"))
		seedRooms(t, src, fixtureRoom(t, 0), fixtureRoom(t, 1))

		if _, err := src.Migrate(ctx, []*Engine{dst}, MigrateParams{Owner: migrateOwnerFixture}); err != nil {
			t.Fatalf("first migrate: %v", err)
		}
		again, err := src.Migrate(ctx, []*Engine{dst}, MigrateParams{Owner: migrateOwnerFixture})
		if err != nil {
			t.Fatalf("second migrate: %v", err)
		}
		if again.Keys != 0 || again.Files != 0 {
			t.Fatalf("second migrate moved %+v, want nothing", again)
		}
	})

	t.Run("rooms larger than one batch stream through", func(t *testing.T) {
		ctx := context.Background()
		src := newTestEngine(t)
		dst := newTestEngine(t)
		room := fixtureRoom(t, 1)
		sender := uuid.NewV7()
		total := migrateBatchEntries + 128
		messages := make([]Message, total)
		for i := range messages {
			messages[i] = Message{
				ClientMsgID: uuid.NewV7(), SenderID: sender, RoomID: room,
				MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"batch"}`),
			}
		}
		results := src.WriteMessages(ctx, messages)
		for i := range results {
			if results[i].Err != nil {
				t.Fatalf("write message %d: %v", i, results[i].Err)
			}
		}

		if _, err := src.Migrate(ctx, []*Engine{dst}, MigrateParams{Owner: migrateOwnerFixture}); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if _, err := src.LastSeq(ctx, room); err == nil {
			t.Fatal("moved room still has a room head on the source")
		}
		if seq, err := dst.LastSeq(ctx, room); err != nil || seq != uint64(total) {
			t.Fatalf("destination last seq = %d, %v; want %d", seq, err, total)
		}
		page, err := dst.Messages(ctx, room, 0, 100)
		if err != nil || len(page.Messages) != 100 || !page.HasMore {
			t.Fatalf("destination page = %d messages, more=%v, %v", len(page.Messages), page.HasMore, err)
		}
	})

	t.Run("missing owner function rejected", func(t *testing.T) {
		src := newTestEngine(t)
		if _, err := src.Migrate(context.Background(), nil, MigrateParams{}); err == nil {
			t.Fatal("migrate without an owner function should fail")
		}
	})

	t.Run("keys without a room stay put", func(t *testing.T) {
		ctx := context.Background()
		src := newTestEngine(t)
		dst := newTestEngine(t)
		if err := src.db.Set([]byte("healthz:ping"), []byte("x"), nil); err != nil {
			t.Fatalf("seed foreign key: %v", err)
		}
		stats, err := src.Migrate(ctx, []*Engine{dst}, MigrateParams{Owner: migrateOwnerFixture})
		if err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if stats.Keys != 0 {
			t.Fatalf("foreign key was moved: %+v", stats)
		}
		value, closer, err := src.db.Get([]byte("healthz:ping"))
		if err != nil || string(value) != "x" {
			t.Fatalf("foreign key missing on source: %q, %v", value, err)
		}
		closer.Close()
	})
}
