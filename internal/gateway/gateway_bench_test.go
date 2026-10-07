package gateway

import (
	"testing"
	"uuid"

	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/proto/pb"
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
		peers := make([]*UserClient, 0, 50)
		for range 50 {
			peer := uuid.NewV7()
			g.TouchRoomUser(room, peer)
			client := newTestClient(peer, 1)
			g.UserSessionManager.LoadOrCreate(peer, NewUserSession).Add(client)
			peers = append(peers, client)
		}

		b.ReportAllocs()
		for b.Loop() {
			g.BroadcastTyping(sender, room)
			for _, peer := range peers {
				select {
				case <-peer.Send:
				default:
				}
			}
		}
	})
}

func BenchmarkGatewayHandlePush(b *testing.B) {
	g := NewGateway(config.NewTest())
	stream := newAPIStream(g, "bench", "127.0.0.1:1")

	const users = 50
	peers := make([]*UserClient, users)
	userIDs := make([]string, users)
	for i := range peers {
		id := uuid.NewV7()
		userIDs[i] = id.String()
		peers[i] = newTestClient(id, 1)
		g.UserSessionManager.LoadOrCreate(id, NewUserSession).Add(peers[i])
	}
	batch := &pb.PushBatch{Pushes: make([]*pb.Push, users)}
	for i := range batch.Pushes {
		batch.Pushes[i] = &pb.Push{
			UserId: userIDs[i], RoomId: "bench-room", RoomSeq: uint64(i + 1),
			MsgId: uuid.NewV7().String(), SenderId: uuid.NewV7().String(), MsgType: 1,
			Payload: []byte(`{"text":"push benchmark"}`), ServerTime: 1728000000000000,
			ClientMsgId: uuid.NewV7().String(),
		}
	}

	b.ReportAllocs()
	for b.Loop() {
		stream.handlePush(batch)
		for _, peer := range peers {
			select {
			case <-peer.Send:
			default:
			}
		}
	}
}
