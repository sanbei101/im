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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
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
			Send: make(chan []byte, 1000),
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
		var totalCount atomic.Uint64
		var dropCount atomic.Uint64
		done := make(chan struct{})

		go func() {
			ticker := time.NewTicker(1 * time.Second)
			defer ticker.Stop()

			var lastCount uint64
			start := time.Now()

			for {
				select {
				case <-ticker.C:
					current := totalCount.Load()
					elapsed := time.Since(start)
					rate := float64(current-lastCount) / 1.0
					avgRate := float64(current) / elapsed.Seconds()

					fmt.Printf("[%s] Users:%d | Current: %d/s | Avg: %.0f/s | Total: %d | Dropped: %d | Elapsed: %.1fs\n",
						time.Now().Format("15:04:05"),
						n,
						int(rate),
						avgRate,
						current,
						dropCount.Load(),
						elapsed.Seconds(),
					)
					lastCount = current

				case <-done:
					current := totalCount.Load()
					elapsed := time.Since(start)
					fmt.Printf("\n📊 Final Stats [Users:%d]:\n", n)
					fmt.Printf("  Total Messages: %d\n", current)
					fmt.Printf("  Dropped: %d (%.2f%%)\n",
						dropCount.Load(),
						float64(dropCount.Load())/float64(current)*100)
					fmt.Printf("  Duration: %.2fs\n", elapsed.Seconds())
					fmt.Printf("  Avg Rate: %.0f msg/s\n", float64(current)/elapsed.Seconds())
					return
				}
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

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					idx := totalCount.Load()
					userID := strconv.FormatUint(idx%uint64(n), 10)
					if session, ok := sm.Load(userID); ok {
						session.Broadcast(payload)
					}
					totalCount.Add(1)
				}
			})
		})
	}
}
