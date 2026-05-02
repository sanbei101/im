package gateway

import (
	"context"
	"sync"

	"github.com/coder/websocket"
	"github.com/phuslu/log"
)

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

func (c *Client) writePump(ctx context.Context) {
	for msg := range c.Send {
		if err := c.Conn.Write(ctx, websocket.MessageText, msg); err != nil {
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

func (s *UserSession) Broadcast(payload []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for c := range s.clients {
		select {
		case c.Send <- payload:
		default:
			log.Warn().Msg("gateway client buffer full, dropping message")
		}
	}
}
