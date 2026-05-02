package gateway

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/sanbei101/im/internal/gateway"
)

func setupFakeUsers(n int) (*gateway.SessionManager, []*gateway.Client) {
	sm := gateway.NewSessionManager()
	var clients []*gateway.Client

	for i := range n {
		userID := strconv.Itoa(i)
		session := sm.LoadOrCreate(userID, gateway.NewUserSession)
		c := &gateway.Client{
			Send: make(chan []byte, 100),
		}
		session.Add(c)
		clients = append(clients, c)
		go func(ch chan []byte) {
			for range ch {
			}
		}(c.Send)
	}

	return sm, clients
}

func BenchmarkSession_RandomUnicast(b *testing.B) {
	levels := []int{10, 100, 1000, 10000}
	payload := []byte("hello private message")

	for _, n := range levels {
		b.Run(fmt.Sprintf("Users_%d", n), func(b *testing.B) {
			sm, clients := setupFakeUsers(n)

			b.Cleanup(func() {
				for _, c := range clients {
					close(c.Send)
				}
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
