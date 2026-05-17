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

type UserClient struct {
	gateway *Gateway
	Conn    *websocket.Conn
	Send    chan [][]byte
	UserID  uuid.UUID
}

var msgBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

func (c *UserClient) writePump(ctx context.Context) {
	for msgs := range c.Send {
		bufPtr := msgBufPool.Get().(*[]byte)
		buf := (*bufPtr)[:0]

		totalLen := 2 + len(msgs) - 1
		for _, msg := range msgs {
			totalLen += len(msg)
		}
		buf = append(buf, '[')
		for i, msg := range msgs {
			if i > 0 {
				buf = append(buf, ',')
			}
			buf = append(buf, msg...)
		}
		buf = append(buf, ']')

		err := c.Conn.Write(ctx, websocket.MessageText, buf)
		if err != nil {
			msgBufPool.Put(bufPtr)
			return
		}
		msgBufPool.Put(bufPtr)
	}
}
func (c *UserClient) readPump(ctx context.Context) {
	for {
		_, payload, err := c.Conn.Read(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client read message failed")
			}
			return
		}
		c.handleUserMessage(ctx, payload)
	}
}

func (c *UserClient) handleUserMessage(ctx context.Context, payload []byte) {
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

	if err := c.gateway.Redis.GatewayPushMessage(ctx, []*db.Message{&message}); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client push message failed")
	}
}

func (c *UserClient) sendError(errMsg string) {
	bin, _ := json.Marshal(map[string]string{"error": errMsg})
	select {
	case c.Send <- [][]byte{bin}:
	default:
		log.Warn().
			Str("error_msg", errMsg).
			Msg("client send error message failed, send channel is full")
	}
}

type UserSessionManager struct {
	UserSessions sync.Map
}

func NewSessionManager() *UserSessionManager {
	return &UserSessionManager{
		UserSessions: sync.Map{},
	}
}

func (sm *UserSessionManager) LoadOrCreate(key string, createFn func() *UserSession) *UserSession {
	if v, ok := sm.UserSessions.Load(key); ok {
		return v.(*UserSession)
	}
	session := createFn()
	actual, _ := sm.UserSessions.LoadOrStore(key, session)
	return actual.(*UserSession)
}

func (sm *UserSessionManager) Delete(key string) {
	sm.UserSessions.Delete(key)
}

func (sm *UserSessionManager) Load(key string) (*UserSession, bool) {
	if v, ok := sm.UserSessions.Load(key); ok {
		return v.(*UserSession), true
	}
	return nil, false
}

type UserSession struct {
	mu      sync.RWMutex
	clients map[*UserClient]struct{}
}

func NewUserSession() *UserSession {
	return &UserSession{
		clients: make(map[*UserClient]struct{}),
	}
}

func (s *UserSession) Add(c *UserClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c] = struct{}{}
}

func (s *UserSession) Remove(c *UserClient) bool {
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
