package store

import (
	"context"
	"errors"
	"testing"
	"uuid"
)

func TestStoreReadCursor(t *testing.T) {
	ctx := context.Background()

	t.Run("advances monotonically", func(t *testing.T) {
		s := newTestStore(t)
		roomID, userID := uuid.NewV7(), uuid.NewV7()

		if _, err := s.ReadSeq(ctx, roomID, userID); !errors.Is(err, ErrNotFound) {
			t.Fatalf("missing cursor error = %v, want %v", err, ErrNotFound)
		}
		if err := s.MarkRead(ctx, roomID, userID, 10); err != nil {
			t.Fatal(err)
		}
		if err := s.MarkRead(ctx, roomID, userID, 5); err != nil {
			t.Fatal(err)
		}
		seq, err := s.ReadSeq(ctx, roomID, userID)
		if err != nil || seq != 10 {
			t.Fatalf("read seq = %d, want 10; err=%v", seq, err)
		}
		if err := s.MarkRead(ctx, roomID, userID, 20); err != nil {
			t.Fatal(err)
		}
		seq, err = s.ReadSeq(ctx, roomID, userID)
		if err != nil || seq != 20 {
			t.Fatalf("read seq = %d, want 20; err=%v", seq, err)
		}
	})

	t.Run("read users follow member order", func(t *testing.T) {
		s := newTestStore(t)
		roomID := uuid.NewV7()
		first, second, outsider := Member{UserID: uuid.NewV7()}, Member{UserID: uuid.NewV7()}, uuid.NewV7()

		for userID, seq := range map[uuid.UUID]uint64{
			first.UserID: 5, second.UserID: 3, outsider: 9,
		} {
			if err := s.MarkRead(ctx, roomID, userID, seq); err != nil {
				t.Fatal(err)
			}
		}
		users, err := s.ReadUsers(ctx, roomID, 4, []Member{second, first})
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0] != first.UserID {
			t.Fatalf("read users = %+v, want [%s]", users, first.UserID)
		}
	})

	t.Run("deletes one cursor and the whole room", func(t *testing.T) {
		s := newTestStore(t)
		roomID := uuid.NewV7()
		first, second := uuid.NewV7(), uuid.NewV7()
		for _, userID := range []uuid.UUID{first, second} {
			if err := s.MarkRead(ctx, roomID, userID, 1); err != nil {
				t.Fatal(err)
			}
		}

		if err := s.DeleteRead(ctx, roomID, first); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ReadSeq(ctx, roomID, first); !errors.Is(err, ErrNotFound) {
			t.Fatalf("deleted cursor error = %v, want %v", err, ErrNotFound)
		}
		if err := s.DeleteRoomReads(ctx, roomID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.ReadSeq(ctx, roomID, second); !errors.Is(err, ErrNotFound) {
			t.Fatalf("room cursors survived deletion: %v", err)
		}
	})
}

func BenchmarkStoreReadCursor(b *testing.B) {
	b.Run("mark-read", func(b *testing.B) {
		s := newTestStore(b)
		ctx := context.Background()
		roomID, userID := uuid.NewV7(), uuid.NewV7()

		b.ResetTimer()
		b.ReportAllocs()
		for i := 1; b.Loop(); i++ {
			if err := s.MarkRead(ctx, roomID, userID, uint64(i)); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("read-seq", func(b *testing.B) {
		s := newTestStore(b)
		ctx := context.Background()
		roomID, userID := uuid.NewV7(), uuid.NewV7()
		if err := s.MarkRead(ctx, roomID, userID, 1); err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			if _, err := s.ReadSeq(ctx, roomID, userID); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("read-users-1000", func(b *testing.B) {
		s := newTestStore(b)
		ctx := context.Background()
		roomID := uuid.NewV7()
		members := make([]Member, 1000)
		for i := range members {
			members[i].UserID = uuid.NewV7()
			if err := s.MarkRead(ctx, roomID, members[i].UserID, uint64(i%500)); err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			users, err := s.ReadUsers(ctx, roomID, 250, members)
			if err != nil || len(users) != 500 {
				b.Fatalf("read users: count=%d err=%v", len(users), err)
			}
		}
	})
}
