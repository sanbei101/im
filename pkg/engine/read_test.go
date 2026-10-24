package engine

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

func TestEngineReadCursor(t *testing.T) {
	ctx := context.Background()

	t.Run("advances monotonically", func(t *testing.T) {
		e := newTestEngine(t)
		roomID, userID := uuid.NewV7(), uuid.NewV7()

		if _, err := e.ReadSeq(ctx, roomID, userID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing cursor error = %v, want %v", err, ErrNotFound)
		}
		if err := e.MarkRead(ctx, roomID, userID, 10); err != nil {
			t.Fatal(err)
		}
		if err := e.MarkRead(ctx, roomID, userID, 5); err != nil {
			t.Fatal(err)
		}
		seq, err := e.ReadSeq(ctx, roomID, userID)
		if err != nil || seq != 10 {
			t.Fatalf("read seq = %d, want 10; err=%v", seq, err)
		}
		if err := e.MarkRead(ctx, roomID, userID, 20); err != nil {
			t.Fatal(err)
		}
		seq, err = e.ReadSeq(ctx, roomID, userID)
		if err != nil || seq != 20 {
			t.Fatalf("read seq = %d, want 20; err=%v", seq, err)
		}
	})

	t.Run("read users follow input order", func(t *testing.T) {
		e := newTestEngine(t)
		roomID := uuid.NewV7()
		first, second, outsider := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()

		for userID, seq := range map[uuid.UUID]uint64{
			first: 5, second: 3, outsider: 9,
		} {
			if err := e.MarkRead(ctx, roomID, userID, seq); err != nil {
				t.Fatal(err)
			}
		}
		users, err := e.ReadUsers(ctx, roomID, 4, []uuid.UUID{second, first})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0] != first {
			t.Fatalf("read users = %+v, want [%s]", users, first)
		}
	})

	t.Run("deletes one cursor and the whole room", func(t *testing.T) {
		e := newTestEngine(t)
		roomID := uuid.NewV7()
		first, second := uuid.NewV7(), uuid.NewV7()
		for _, userID := range []uuid.UUID{first, second} {
			if err := e.MarkRead(ctx, roomID, userID, 1); err != nil {
				t.Fatal(err)
			}
		}

		if err := e.DeleteRead(ctx, roomID, first); err != nil {
			t.Fatal(err)
		}
		if _, err := e.ReadSeq(ctx, roomID, first); !errors.Is(err, ErrNotFound) {
			t.Fatalf("deleted cursor error = %v, want %v", err, ErrNotFound)
		}
		if err := e.DeleteRoomReads(ctx, roomID); err != nil {
			t.Fatal(err)
		}
		if _, err := e.ReadSeq(ctx, roomID, second); !errors.Is(err, ErrNotFound) {
			t.Fatalf("room cursors survived deletion: %v", err)
		}
	})
}

func BenchmarkEngineReadCursor(b *testing.B) {
	b.Run("mark-read", func(b *testing.B) {
		e := newTestEngine(b)
		ctx := context.Background()
		roomID, userID := uuid.NewV7(), uuid.NewV7()

		b.ResetTimer()
		b.ReportAllocs()
		for i := 1; b.Loop(); i++ {
			if err := e.MarkRead(ctx, roomID, userID, uint64(i)); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("read-seq", func(b *testing.B) {
		e := newTestEngine(b)
		ctx := context.Background()
		roomID, userID := uuid.NewV7(), uuid.NewV7()
		if err := e.MarkRead(ctx, roomID, userID, 1); err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			if _, err := e.ReadSeq(ctx, roomID, userID); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("read-users-1000", func(b *testing.B) {
		e := newTestEngine(b)
		ctx := context.Background()
		roomID := uuid.NewV7()
		userIDs := make([]uuid.UUID, 1000)
		for i := range userIDs {
			userIDs[i] = uuid.NewV7()
			if err := e.MarkRead(ctx, roomID, userIDs[i], uint64(i%500)); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			users, err := e.ReadUsers(ctx, roomID, 250, userIDs)
			if err != nil || len(users) != 500 {
				b.Fatalf("read users: count=%d err=%v", len(users), err)
			}
		}
	})
}
