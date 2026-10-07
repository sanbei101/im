package store

import (
	"context"
	"fmt"
	"testing"
	"uuid"
)

func BenchmarkStoreUsersSearch(b *testing.B) {
	s := newBenchStore(b)
	ctx := context.Background()
	const users = 5000
	for i := range users {
		if _, err := s.CreateUser(ctx, fmt.Sprintf("user-%05d", i), "x"); err != nil {
			b.Fatal(err)
		}
	}
	// Matches a single user: both implementations must scan the whole table.
	b.Run("rare-keyword", func(b *testing.B) {
		b.Run("index", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := s.SearchUsers(ctx, "4999", 20); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
	// Matches thousands of users: both implementations stop at the limit.
	b.Run("common-keyword", func(b *testing.B) {
		b.Run("index", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := s.SearchUsers(ctx, "user", 20); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

func BenchmarkStoreUsersByIDs(b *testing.B) {
	s := newBenchStore(b)
	ctx := context.Background()
	const users = 500
	ids := make([]uuid.UUID, users)
	for i := range ids {
		user, err := s.CreateUser(ctx, fmt.Sprintf("user-%05d", i), "x")
		if err != nil {
			b.Fatal(err)
		}
		ids[i] = user.UserID
	}
	b.Run("per-id-get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, id := range ids {
				if _, err := s.UserByID(ctx, id); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.Run("batch-seek", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := s.UsersByIDs(ctx, ids); err != nil {
				b.Fatal(err)
			}
		}
	})
}
