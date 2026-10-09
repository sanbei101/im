package gateway

import (
	"bytes"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
)

func newTestGateway(t *testing.T) *Gateway {
	t.Helper()
	return NewGateway(config.NewTest())
}

func newTestClient(userID uuid.UUID, sendBuffer int) *UserClient {
	return &UserClient{
		UserID: userID,
		Send:   make(chan []byte, sendBuffer),
		frames: render.NewFrameWriter(),
	}
}

func TestGateway(t *testing.T) {
	t.Run("pending request registry", func(t *testing.T) {
		g := newTestGateway(t)
		client := newTestClient(uuid.NewV7(), 1)

		g.StorePending("req-1", client)
		if got := g.LoadAndDeletePending("req-1"); got != client {
			t.Fatalf("pending lookup returned %v", got)
		}
		if got := g.LoadAndDeletePending("req-1"); got != nil {
			t.Fatalf("pending lookup must consume the entry, got %v", got)
		}

		g.StorePending("req-2", client)
		g.DeletePending("req-2")
		if got := g.LoadAndDeletePending("req-2"); got != nil {
			t.Fatalf("deleted pending entry came back: %v", got)
		}
	})

	t.Run("session manager creates once per user", func(t *testing.T) {
		manager := NewSessionManager()
		user := uuid.NewV7()
		created := 0
		create := func() *UserSession {
			created++
			return NewUserSession()
		}

		first := manager.LoadOrCreate(user, create)
		second := manager.LoadOrCreate(user, create)
		if first != second || created != 1 {
			t.Fatalf("session recreated: created=%d same=%v", created, first == second)
		}

		found, ok := manager.Load(user)
		if !ok || found != first {
			t.Fatalf("load after create: ok=%v session=%v", ok, found)
		}
		manager.Delete(user)
		if _, ok := manager.Load(user); ok {
			t.Fatal("session survived delete")
		}
		if ids := manager.All(); len(ids) != 0 {
			t.Fatalf("all after delete: %v", ids)
		}
	})

	t.Run("concurrent session load or create", func(t *testing.T) {
		manager := NewSessionManager()
		user := uuid.NewV7()
		const goroutines = 16
		sessions := make([]*UserSession, goroutines)
		var wg sync.WaitGroup
		wg.Add(goroutines)
		for i := range goroutines {
			go func() {
				defer wg.Done()
				sessions[i] = manager.LoadOrCreate(user, NewUserSession)
			}()
		}
		wg.Wait()
		for _, session := range sessions {
			if session != sessions[0] {
				t.Fatal("concurrent LoadOrCreate produced multiple sessions")
			}
		}
	})

	t.Run("client send buffer guard", func(t *testing.T) {
		client := newTestClient(uuid.NewV7(), 1)
		if err := client.sendFrame([]byte("frame-1")); err != nil {
			t.Fatalf("send into free buffer: %v", err)
		}
		if err := client.sendFrame([]byte("frame-2")); err == nil {
			t.Fatal("send into full buffer must fail")
		}
		client.closeSend()
		if err := client.sendFrame([]byte("frame-3")); err == nil {
			t.Fatal("send on closed client must fail")
		}
		client.closeSend()
	})

	t.Run("typing broadcast reaches room peers only", func(t *testing.T) {
		g := newTestGateway(t)
		sender := uuid.NewV7()
		peer := uuid.NewV7()
		outsider := uuid.NewV7()
		peerClient := newTestClient(peer, 4)
		peerSession := g.UserSessionManager.LoadOrCreate(peer, NewUserSession)
		peerSession.Add(peerClient)
		outsiderClient := newTestClient(outsider, 4)

		g.TouchRoomUser("room-1", sender)
		g.TouchRoomUser("room-1", peer)
		g.TouchRoomUser("room-2", outsider)

		g.BroadcastTyping(sender, "room-1")
		select {
		case frame := <-peerClient.Send:
			var typing render.TypingFrame
			if err := render.NewFrameReader(bytes.NewReader(frame)).ReadFrame(&typing); err != nil {
				t.Fatalf("typing frame decode: %v", err)
			}
			if typing.Type != "typing" || typing.RoomID != "room-1" || typing.UserID != sender.String() {
				t.Fatalf("typing frame wrong: %+v", typing)
			}
		case <-time.After(time.Second):
			t.Fatal("peer did not receive the typing frame")
		}
		select {
		case frame := <-outsiderClient.Send:
			t.Fatalf("user in another room received the frame: %s", frame)
		default:
		}
		// The sender itself and unknown rooms are silently ignored.
		g.BroadcastTyping(sender, "room-1")
		g.BroadcastTyping(sender, "room-unknown")
	})

	t.Run("room routing is deterministic per shard topology", func(t *testing.T) {
		g := newTestGateway(t)
		rooms := make([]uuid.UUID, 100)
		for i := range rooms {
			rooms[i] = uuid.NewV7()
		}
		for _, roomID := range rooms {
			stream := g.streamForRoom(roomID)
			if stream == nil {
				t.Fatal("no stream for room")
			}
			if again := g.streamForRoom(roomID); again != stream {
				t.Fatalf("routing not deterministic for room %s", roomID)
			}
		}
	})

	t.Run("notification dispatch to user sessions", func(t *testing.T) {
		g := newTestGateway(t)
		targetUser := uuid.NewV7()
		client := newTestClient(targetUser, 2)
		session := g.UserSessionManager.LoadOrCreate(targetUser, NewUserSession)
		session.Add(client)

		stream := newAPIStream(g, "test-stream", "127.0.0.1:8801")
		stream.handleNotification(&pb.NotificationBatch{
			Notifications: []*pb.Notification{
				{
					UserId: targetUser.String(),
					Event:  "reaction",
					Data:   []byte(`{"room_id":"r1","msg_id":"m1"}`),
				},
			},
		})

		select {
		case frameBytes := <-client.Send:
			var notif render.NotificationFrame
			if err := render.NewFrameReader(bytes.NewReader(frameBytes)).ReadFrame(&notif); err != nil {
				t.Fatalf("notification frame decode: %v", err)
			}
			if notif.Type != "notification" || notif.Event != "reaction" {
				t.Fatalf("unexpected notification frame: %+v", notif)
			}
		case <-time.After(time.Second):
			t.Fatal("target client did not receive notification frame")
		}
	})
}
