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

func setupFakeUsers(ctx context.Context, n int) (*gateway.SessionManager, []*gateway.Client, *httptest.Server) {
	sm := gateway.NewSessionManager()
	var clients []*gateway.Client
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer c.CloseNow()
		for {
			_, _, err := c.Read(ctx)
			if err != nil {
				return
			}
		}
	}))
	for i := range n {
		userID := strconv.Itoa(i)
		session := sm.LoadOrCreate(userID, gateway.NewUserSession)
		wsURL := "ws" + srv.URL[4:]
		clientConn, resp, err := websocket.Dial(ctx, wsURL, nil)
		if err != nil {
			log.Panic().Err(err).Msg("failed to connect to websocket server")
		}
		if resp.StatusCode != http.StatusSwitchingProtocols {
			log.Panic().Msg("failed to switch protocols")
		}
		c := &gateway.Client{
			Conn: clientConn,
			Send: make(chan []byte, 1000),
		}
		session.Add(c)
		clients = append(clients, c)
		go func(cli *gateway.Client) {
			for msg := range cli.Send {
				err := cli.Conn.Write(ctx, websocket.MessageText, msg)
				if err != nil {
					log.Error().Err(err).Msg("failed to write message")
				}
			}
		}(c)
	}
	return sm, clients, srv
}

func BenchmarkSession(b *testing.B) {
	levels := []int{10, 100, 1000, 10000}
	payload := []byte("hello private message")

	for _, n := range levels {
		var totalCount atomic.Uint64
		var dropCount atomic.Uint64

		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			var lastCount uint64
			start := time.Now()

			for {
				<-ticker.C
				current := totalCount.Load()
				elapsed := time.Since(start)
				rate := float64(current-lastCount) / 1.0
				avgRate := float64(current) / elapsed.Seconds()
				lastCount = current
				log.Info().
					Int("users", n).
					Uint64("current", current).
					Uint64("dropped", dropCount.Load()).
					Float64("rate", rate).
					Float64("avg_rate", avgRate).
					Float64("elapsed", elapsed.Seconds()).
					Msg("benchmark progress")
			}
		}()
		b.Run(fmt.Sprintf("Users_%d", n), func(b *testing.B) {
			sm, clients, srv := setupFakeUsers(b.Context(), n)

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
			for b.Loop() {
				idx := totalCount.Load()
				userID := strconv.FormatUint(idx%uint64(n), 10)
				if session, ok := sm.Load(userID); ok {
					session.Broadcast(payload)
				}
				totalCount.Add(1)
			}
		})
	}
}
