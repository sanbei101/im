package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/phuslu/log"
	"github.com/sanbei101/im/internal/gateway"
)

func setupFakeUsers(n int) (*gateway.SessionManager, []*gateway.Client, *httptest.Server) {
	sm := gateway.NewSessionManager()
	var clients []*gateway.Client
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		ctx := context.Background()
		for {
			c.Read(ctx)
		}
	}))
	ctx := context.Background()
	for i := range n {
		userID := strconv.Itoa(i)
		session := sm.LoadOrCreate(userID, gateway.NewUserSession)
		wsURL := "ws" + srv.URL[4:]
		dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		clientConn, resp, err := websocket.Dial(dialCtx, wsURL, nil)
		cancel()
		if err != nil {
			log.Panic().Err(err).Msg("failed to connect to websocket server")
		}
		if resp.StatusCode != http.StatusSwitchingProtocols {
			log.Panic().Msg("failed to switch protocols")
		}
		c := &gateway.Client{
			Conn: clientConn,
			Send: make(chan []byte, 100),
		}
		session.Add(c)
		clients = append(clients, c)
		go func(cli *gateway.Client) {
			for msg := range cli.Send {
				cli.Conn.Write(ctx, websocket.MessageText, msg)
			}
		}(c)
	}
	return sm, clients, srv
}

func BenchmarkSession(b *testing.B) {
	levels := []int{10, 100, 1000, 10000}
	payload := []byte("hello private message")

	for _, n := range levels {
		b.Run(fmt.Sprintf("Users_%d", n), func(b *testing.B) {
			sm, clients, srv := setupFakeUsers(n)

			b.Cleanup(func() {
				for _, c := range clients {
					if c.Conn != nil {
						c.Conn.CloseNow()
					}
					close(c.Send)
				}
				srv.Close()
			})

			b.ResetTimer()
			b.ReportAllocs()

			var counter uint64

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					idx := atomic.AddUint64(&counter, 1) % uint64(n)
					userID := strconv.FormatUint(idx, 10)
					if session, ok := sm.Load(userID); ok {
						session.Broadcast(payload)
					}
				}
			})
		})
	}
}
