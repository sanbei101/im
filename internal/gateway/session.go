package gateway

import (
	"context"
	"sync"

	"github.com/coder/websocket"
	"github.com/phuslu/log"
)

type Client struct {
	Conn *websocket.Conn
	Send chan [][]byte
}

func (c *Client) writePump(ctx context.Context) {
	for msgs := range c.Send {
		// 将多条消息拼接为 JSON 数组: [msg1, msg2, ...]
		var buf []byte
		buf = append(buf, '[')
		for i, msg := range msgs {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, msg...)
		}
		buf = append(buf, ']')
		if err := c.Conn.Write(ctx, websocket.MessageText, buf); err != nil {
			return
		}
	}
}

type SessionManager struct {
	sessions sync.Map
}

func NewSessionManager() *SessionManager {
	return &SessionManager{}
}

func (sm *SessionManager) LoadOrCreate(key string, createFn func() *UserSession) *UserSession {
	if v, ok := sm.sessions.Load(key); ok {
		return v.(*UserSession)
	}
	session := createFn()
	actual, _ := sm.sessions.LoadOrStore(key, session)
	return actual.(*UserSession)
}

func (sm *SessionManager) Delete(key string) {
	sm.sessions.Delete(key)
}

func (sm *SessionManager) Load(key string) (*UserSession, bool) {
	if v, ok := sm.sessions.Load(key); ok {
		return v.(*UserSession), true
	}
	return nil, false
}

type UserSession struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewUserSession() *UserSession {
	return &UserSession{
		clients: make(map[*Client]struct{}),
	}
}

func (s *UserSession) Add(c *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
}

func (s *UserSession) Remove(c *Client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, c)
	return len(s.clients) == 0
}

func (s *UserSession) Broadcast(payloads [][]byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c := range s.clients {
		select {
		case c.Send <- payloads:
		default:
			log.Warn().Msg("gateway client buffer full, dropping message")
		}
	}
}
