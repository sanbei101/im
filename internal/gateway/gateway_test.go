package gateway

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/coder/websocket"

	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
)

func newTestGateway(t *testing.T) *Gateway {
	t.Helper()
	return NewGateway(config.LoadTest())
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

		client := newTestClient(user, 1)
		first.Add(client)
		if manager.RemoveClient(user, newTestClient(user, 1)) {
			t.Fatal("removing a non-member client must not delete the session")
		}
		if _, ok := manager.Load(user); !ok {
			t.Fatal("session deleted while it still has clients")
		}
		if !manager.RemoveClient(user, client) {
			t.Fatal("removing the last client must report an empty session")
		}
		if _, ok := manager.Load(user); ok {
			t.Fatal("session survived removal of its last client")
		}
		if ids := manager.All(); len(ids) != 0 {
			t.Fatalf("all after removal: %v", ids)
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
			typing, err := render.NewFrameReader(bytes.NewReader(frame)).ReadFrame[render.TypingFrame]()
			if err != nil {
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

		stream := newAPIStream(g, "127.0.0.1:8801")
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
			notif, err := render.NewFrameReader(bytes.NewReader(frameBytes)).ReadFrame[render.NotificationFrame]()
			if err != nil {
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

func TestUserClientPingPump(t *testing.T) {
	t.Run("keeps the connection alive while the peer answers", func(t *testing.T) {
		serverCtx, stopServer := context.WithCancel(t.Context())
		defer stopServer()

		started := make(chan struct{})
		stopped := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				OriginPatterns: []string{"*"},
			})
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")

			// Ping requires a concurrent reader to consume the pong.
			go func() {
				for {
					if _, _, err := conn.Read(serverCtx); err != nil {
						return
					}
				}
			}()

			client := &UserClient{Conn: conn, UserID: uuid.NewV7()}
			close(started)
			client.pingPump(serverCtx, 10*time.Millisecond, 50*time.Millisecond)
			close(stopped)
		}))
		defer server.Close()

		dialCtx, cancelDial := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancelDial()
		conn, _, err := websocket.Dial( //nolint:bodyclose // Dial clears the body on success and closes it on error.
			dialCtx,
			server.URL,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		defer conn.Close(websocket.StatusNormalClosure, "")

		// The client read loop is what lets the WebSocket stack answer pings.
		readCtx, stopRead := context.WithCancel(t.Context())
		defer stopRead()
		go func() {
			for {
				if _, _, err := conn.Read(readCtx); err != nil {
					return
				}
			}
		}()

		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("server did not accept the connection")
		}

		// Several ping intervals pass; a working peer must keep the connection up.
		time.Sleep(60 * time.Millisecond)
		select {
		case <-stopped:
			t.Fatal("pingPump stopped while the peer was answering")
		default:
		}

		stopServer()
		select {
		case <-stopped:
		case <-time.After(time.Second):
			t.Fatal("pingPump did not stop after context cancellation")
		}
	})

	t.Run("times out for a silent peer", func(t *testing.T) {
		serverCtx, stopServer := context.WithCancel(t.Context())
		defer stopServer()

		stopped := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
				OriginPatterns: []string{"*"},
			})
			if err != nil {
				return
			}
			defer conn.Close(websocket.StatusNormalClosure, "")

			go func() {
				for {
					if _, _, err := conn.Read(serverCtx); err != nil {
						return
					}
				}
			}()

			client := &UserClient{Conn: conn, UserID: uuid.NewV7()}
			client.pingPump(serverCtx, 10*time.Millisecond, 30*time.Millisecond)
			close(stopped)
		}))
		defer server.Close()

		dialCtx, cancelDial := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancelDial()
		conn, _, err := websocket.Dial( //nolint:bodyclose // Dial clears the body on success and closes it on error.
			dialCtx,
			server.URL,
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		// Deliberately never read: the pong is not processed, so pingPump must
		// time out and close the connection.
		defer func() {
			if err := conn.CloseNow(); err != nil {
				t.Logf("close websocket: %v", err)
			}
		}()

		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("pingPump did not time out for a silent peer")
		}
	})
}

func BenchmarkGateway(b *testing.B) {
	g := NewGateway(config.LoadTest())

	b.Run("session-load-or-create", func(b *testing.B) {
		user := uuid.NewV7()
		g.UserSessionManager.LoadOrCreate(user, NewUserSession)

		b.ResetTimer()
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

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			g.TouchRoomUser(room, user)
		}
	})
	b.Run("pending-round-trip", func(b *testing.B) {
		client := newTestClient(uuid.NewV7(), 1)

		b.ResetTimer()
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

		b.ResetTimer()
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
	b.Run("room-push-fanout", func(b *testing.B) {
		stream := newAPIStream(g, "127.0.0.1:1")

		const users = 50
		peers := make([]*UserClient, users)
		userIDs := make([]string, users)
		for i := range peers {
			id := uuid.NewV7()
			userIDs[i] = id.String()
			peers[i] = newTestClient(id, 1)
			g.UserSessionManager.LoadOrCreate(id, NewUserSession).Add(peers[i])
		}

		roomBatch := &pb.RoomPushBatch{
			RoomPushes: []*pb.RoomPush{{
				RoomId:     "bench-room",
				RoomSeq:    1,
				SenderId:   uuid.NewV7().String(),
				MsgType:    1,
				Payload:    []byte(`{"text":"push benchmark"}`),
				ServerTime: 1728000000000000,
				UserIds:    userIDs,
			}},
		}

		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			stream.handleRoomPush(roomBatch)
			for _, peer := range peers {
				select {
				case <-peer.Send:
				default:
				}
			}
		}
	})
}
