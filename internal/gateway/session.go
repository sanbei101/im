package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"uuid"

	"github.com/coder/websocket"
	"github.com/phuslu/log"

	imv1 "github.com/sanbei101/im/kitex_gen/im/v1"
)

type UserClient struct {
	gateway *Gateway
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  uuid.UUID
}

func (c *UserClient) writePump(ctx context.Context) {
	for frame := range c.Send {
		if err := c.Conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
			return
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
		if err := c.handleFrame(ctx, payload); err != nil {
			c.sendError(err.Error())
		}
	}
}

func (c *UserClient) handleFrame(ctx context.Context, payload []byte) error {
	var input struct {
		Type         string          `json:"type"`
		RequestID    string          `json:"request_id"`
		ClientMsgID  string          `json:"client_msg_id"`
		RoomID       string          `json:"room_id"`
		MsgType      string          `json:"msg_type"`
		Payload      json.RawMessage `json:"payload"`
		ReplyToMsgID string          `json:"reply_to_msg_id"`
		Ext          json.RawMessage `json:"ext"`
	}
	if err := json.Unmarshal(payload, &input); err != nil {
		return fmt.Errorf("decode websocket message: %w", err)
	}
	if input.Type == "ping" {
		return c.sendJSON(map[string]any{"type": "pong"})
	}
	if input.RoomID == "" || input.ClientMsgID == "" {
		return errors.New("room_id and client_msg_id are required")
	}
	roomID, err := uuid.Parse(input.RoomID)
	if err != nil {
		return fmt.Errorf("invalid room_id: %w", err)
	}
	requestID := input.RequestID
	if requestID == "" {
		requestID = uuid.NewV7().String()
	}
	msgType, err := messageType(input.MsgType)
	if err != nil {
		return err
	}
	message := &imv1.SendMessage{
		RequestId: requestID, ClientMsgId: input.ClientMsgID, SenderId: c.UserID.String(),
		RoomId: input.RoomID, MsgType: msgType, Payload: input.Payload, Ext: input.Ext,
		ReplyToMsgId: input.ReplyToMsgID,
	}
	c.gateway.pending.Store(requestID, c)
	if err := c.gateway.send(ctx, roomID, message); err != nil {
		c.gateway.pending.Delete(requestID)
		return err
	}
	return nil
}

func messageType(value string) (int32, error) {
	switch value {
	case "text":
		return 1, nil
	case "image":
		return 2, nil
	case "video":
		return 3, nil
	case "file":
		return 4, nil
	case "system":
		return 5, nil
	default:
		number, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid msg_type %q: %w", value, err)
		}
		return int32(number), nil
	}
}

func (c *UserClient) sendJSON(value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	select {
	case c.Send <- data:
		return nil
	default:
		return errors.New("client send buffer is full")
	}
}

func (c *UserClient) sendError(message string) {
	if err := c.sendJSON(map[string]any{"type": "error", "error": message}); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("send error frame failed")
	}
}

const sessionShardCount = 256

type sessionShard struct {
	mu sync.RWMutex
	m  map[string]*UserSession
}

type UserSessionManager struct {
	shards [sessionShardCount]*sessionShard
}

func NewSessionManager() *UserSessionManager {
	manager := &UserSessionManager{}
	for i := range manager.shards {
		manager.shards[i] = &sessionShard{m: make(map[string]*UserSession)}
	}
	return manager
}

func (manager *UserSessionManager) shard(key string) *sessionShard {
	var hash uint32 = 2166136261
	for i := range key {
		hash = (hash ^ uint32(key[i])) * 16777619
	}
	return manager.shards[hash%sessionShardCount]
}

func (manager *UserSessionManager) LoadOrCreate(key string, create func() *UserSession) *UserSession {
	shard := manager.shard(key)
	shard.mu.RLock()
	session := shard.m[key]
	shard.mu.RUnlock()
	if session != nil {
		return session
	}
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if session = shard.m[key]; session == nil {
		session = create()
		shard.m[key] = session
	}
	return session
}

func (manager *UserSessionManager) Delete(key string) {
	shard := manager.shard(key)
	shard.mu.Lock()
	delete(shard.m, key)
	shard.mu.Unlock()
}

func (manager *UserSessionManager) Load(key string) (*UserSession, bool) {
	shard := manager.shard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	value, ok := shard.m[key]
	return value, ok
}

func (manager *UserSessionManager) All() []uuid.UUID {
	var result []uuid.UUID
	for i := range manager.shards {
		shard := manager.shards[i]
		shard.mu.RLock()
		for key := range shard.m {
			if id, err := uuid.Parse(key); err == nil {
				result = append(result, id)
			}
		}
		shard.mu.RUnlock()
	}
	return result
}

type UserSession struct {
	mu      sync.RWMutex
	clients map[*UserClient]struct{}
}

func NewUserSession() *UserSession { return &UserSession{clients: make(map[*UserClient]struct{})} }
func (s *UserSession) Add(client *UserClient) {
	s.mu.Lock()
	s.clients[client] = struct{}{}
	s.mu.Unlock()
}

func (s *UserSession) Remove(client *UserClient) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, client)
	return len(s.clients) == 0
}

func (s *UserSession) Broadcast(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		select {
		case client.Send <- data:
		default:
			log.Warn().Str("user_id", client.UserID.String()).Msg("client send buffer is full")
		}
	}
}
