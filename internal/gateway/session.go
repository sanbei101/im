package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/phuslu/log"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
)

type UserClient struct {
	gateway *Gateway
	Conn    *websocket.Conn
	Send    chan [][]byte
	UserID  uuid.UUID
}

func (c *UserClient) writePump(ctx context.Context) {
	for frames := range c.Send {
		for _, frame := range frames {
			if err := c.Conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
				return
			}
		}
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
	req := &imv1.SendMessageReq{}
	if err := req.UnmarshalVT(payload); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client unmarshal SendMessageReq failed")
		c.sendError("invalid proto")
		return
	}

	// Ping convention: msg_type == UNSPECIFIED is a keepalive — drop silently.
	if req.GetMsgType() == imv1.MessageType_MESSAGE_TYPE_UNSPECIFIED {
		return
	}

	msgID, err := uuid.NewV7()
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client generate msg_id failed")
		c.sendError("failed to generate msg_id")
		return
	}

	push, err := sendMessageReqToMessagePush(req, c.UserID, msgID, time.Now().UnixMicro())
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client send message invalid")
		c.sendError(err.Error())
		return
	}

	if err := c.gateway.MQ.GatewayPushMessage(ctx, []*imv1.MessagePush{push}); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client push message failed")
		c.sendError("push failed")
	}
}

func (c *UserClient) sendError(errMsg string) {
	code := int32(-1)
	msg := errMsg
	resp := &imv1.SendMessageResp{
		Code:   &code,
		ErrMsg: &msg,
	}
	bin := make([]byte, resp.SizeVT())
	n, err := resp.MarshalToVT(bin)
	if err != nil {
		log.Error().Err(err).Str("err_msg", errMsg).Msg("marshal SendMessageResp failed")
		return
	}
	bin = bin[:n]
	select {
	case c.Send <- [][]byte{bin}:
	default:
		log.Warn().
			Str("error_msg", errMsg).
			Msg("client send error message failed, send channel is full")
	}
}

const shardCount = 256

type sessionShard struct {
	mu sync.RWMutex
	m  map[string]*UserSession
}

type UserSessionManager struct {
	shards [shardCount]*sessionShard
}

func NewSessionManager() *UserSessionManager {
	sm := &UserSessionManager{}
	for i := range shardCount {
		sm.shards[i] = &sessionShard{
			m: make(map[string]*UserSession),
		}
	}
	return sm
}

func (sm *UserSessionManager) getShard(key string) *sessionShard {
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= 16777619
	}
	return sm.shards[hash%shardCount]
}

func (sm *UserSessionManager) LoadOrCreate(key string, createFn func() *UserSession) *UserSession {
	shard := sm.getShard(key)

	shard.mu.RLock()
	if session, ok := shard.m[key]; ok {
		shard.mu.RUnlock()
		return session
	}
	shard.mu.RUnlock()

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if session, ok := shard.m[key]; ok {
		return session
	}

	session := createFn()
	shard.m[key] = session
	return session
}

func (sm *UserSessionManager) Delete(key string) {
	shard := sm.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.m, key)
}

func (sm *UserSessionManager) Load(key string) (*UserSession, bool) {
	shard := sm.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	session, ok := shard.m[key]
	return session, ok
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