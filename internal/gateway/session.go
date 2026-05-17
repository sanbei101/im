package gateway

import (
	"context"
	"encoding/json/v2"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/phuslu/log"
	"github.com/sanbei101/im/internal/db"
)

type Client struct {
	gateway *Gateway
	Conn    *websocket.Conn
	Send    chan [][]byte
	UserID  uuid.UUID
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
func (c *Client) readPump(ctx context.Context) {
	for {
		_, payload, err := c.Conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client read message failed")
			}
			return
		}
		c.handleIncomingMessage(ctx, payload)
	}
}

func (c *Client) handleIncomingMessage(ctx context.Context, payload []byte) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil && envelope.Type == "ping" {
		return
	}

	var message db.Message
	if err := json.Unmarshal(payload, &message); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client unmarshal message failed")
		c.sendError("invalid message format")
		return
	}

	if message.ClientMsgID == uuid.Nil {
		log.Error().Str("user_id", c.UserID.String()).Msg("client missing client_msg_id")
		c.sendError("missing client_msg_id")
		return
	}

	var err error
	message.MsgID, err = uuid.NewV7()
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client generate msg_id failed")
		c.sendError("failed to generate msg_id")
		return
	}
	message.SenderID = c.UserID
	message.ServerTime = time.Now().UnixMicro()

	// 使用绑定的 gateway 指针去调用 Redis
	if err := c.gateway.redis.GatewayPushMessage(ctx, []*db.Message{&message}); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client push message failed")
	}
}
func (c *Client) sendError(errMsg string) {
	bin, _ := json.Marshal(map[string]string{"error": errMsg})
	select {
	case c.Send <- [][]byte{bin}:
	default:
		log.Warn().
			Str("error_msg", errMsg).
			Msg("client send error message failed, send channel is full")
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
