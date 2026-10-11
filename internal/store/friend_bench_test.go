package store

import (
	"context"
	"testing"
	"uuid"
)

func BenchmarkStoreFriendOps(b *testing.B) {
	s := newBenchStore(b)
	ctx := context.Background()

	u1 := uuid.NewV7()
	u2 := uuid.NewV7()

	b.Run("apply-and-audit", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			from := uuid.NewV7()
			to := uuid.NewV7()
			if err := s.ApplyFriend(ctx, from, to, "hello"); err != nil {
				b.Fatal(err)
			}
			if err := s.AuditFriend(ctx, to, from, true); err != nil {
				b.Fatal(err)
			}
		}
	})

	if err := s.ApplyFriend(ctx, u1, u2, "bench"); err != nil {
		b.Fatal(err)
	}
	if err := s.AuditFriend(ctx, u2, u1, true); err != nil {
		b.Fatal(err)
	}

	b.Run("list-friends", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := s.Friends(ctx, u1); err != nil {
				b.Fatal(err)
			}
		}
	})
}
