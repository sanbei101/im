package gateway

import (
	"testing"
	"uuid"

	"github.com/sanbei101/im/pkg/config"
)

func BenchmarkGateway(b *testing.B) {
	g := NewGateway(config.NewTest())

	b.Run("session-load-or-create", func(b *testing.B) {
		user := uuid.NewV7()
		g.UserSessionManager.LoadOrCreate(user, NewUserSession)

		b.ReportAllocs()
		for b.Loop() {
			if session := g.UserSessionManager.LoadOrCreate(user, NewUserSession); session == nil {
				b.Fatal("nil session")
			}
		}
	})
	b.Run("touch-room-user", func(b *testing.B) {
		room := "bench-room"
		user := uuid.NewV7()
		g.TouchRoomUser(room, user)

		b.ReportAllocs()
		for b.Loop() {
			g.TouchRoomUser(room, user)
		}
	})
	b.Run("pending-round-trip", func(b *testing.B) {
		client := newTestClient(uuid.NewV7(), 1)

		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			requestID := uuid.NewV7().String()
			g.StorePending(requestID, client)
			if got := g.LoadAndDeletePending(requestID); got == nil {
				b.Fatal("pending entry lost")
			}
		}
	})
	b.Run("broadcast-typing", func(b *testing.B) {
		room := "bench-typing"
		sender := uuid.NewV7()
		g.TouchRoomUser(room, sender)
		for range 50 {
			peer := uuid.NewV7()
			g.TouchRoomUser(room, peer)
			g.UserSessionManager.LoadOrCreate(peer, NewUserSession).Add(newTestClient(peer, 4096))
		}

		b.ReportAllocs()
		for b.Loop() {
			g.BroadcastTyping(sender, room)
		}
	})
}
