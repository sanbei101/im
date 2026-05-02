package gateway

import (
	"context"
	"sync"

	"github.com/coder/websocket"
	"github.com/phuslu/log"
)

type client struct {
	conn *websocket.Conn
	send chan []byte
}

type UserSession struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

func NewUserSession() *UserSession {
	return &UserSession{
		clients: make(map[*client]struct{}),
	}
}

func (s *UserSession) Add(c *client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
}

func (s *UserSession) Remove(c *client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, c)
	return len(s.clients) == 0
}

func (s *UserSession) Broadcast(payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c := range s.clients {
		select {
		case c.send <- payload:
		default:
			log.Warn().Msg("gateway client buffer full, dropping message")
		}
	}
}

func (c *client) writePump(ctx context.Context) {
	for msg := range c.send {
		if err := c.conn.Write(ctx, websocket.MessageText, msg); err != nil {
			return
		}
	}
}
