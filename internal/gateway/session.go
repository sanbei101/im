package gateway

import (
	"context"
	"errors"
	"io"
	"net"
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
	Send    chan []byte
	UserID  uuid.UUID
}

func (c *UserClient) writePump(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case firstMsg, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.Conn.Write(ctx, websocket.MessageBinary, firstMsg); err != nil {
				return
			}
		BatchLoop:
			for len(c.Send) > 0 {
				select {
				case nextMsg := <-c.Send:
					if err := c.Conn.Write(ctx, websocket.MessageBinary, nextMsg); err != nil {
						return
					}
				default:
					break BatchLoop
				}
			}
		}
	}
}

func (c *UserClient) readPump(ctx context.Context) {
	for {
		_, payload, err := c.Conn.Read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				log.Info().Str("user_id", c.UserID.String()).Msg("client disconnected")
				return
			}
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client read message failed")
			}
			return
		}
		c.handleUserMessage(ctx, payload)
	}
}

func (c *UserClient) handleUserMessage(ctx context.Context, payload []byte) {
	frame := &imv1.ClientFrame{}
	if err := frame.UnmarshalVT(payload); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client unmarshal ClientFrame failed")
		c.sendError("", "invalid_frame", "invalid client frame")
		return
	}

	clientMsgID := frame.GetClientMsgId()
	if _, err := uuid.Parse(clientMsgID); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client frame has invalid client_msg_id")
		c.sendError(clientMsgID, "invalid_client_msg_id", "client_msg_id must be a UUID")
		return
	}

	if frame.GetPing() != nil {
		c.sendFrame(&imv1.ServerFrame{
			ClientMsgId: &clientMsgID,
			Payload:     &imv1.ServerFrame_Pong{Pong: &imv1.Pong{}},
		})
		return
	}

	req := frame.GetSendMessage()
	if req == nil {
		c.sendError(clientMsgID, "unsupported_frame", "frame type is not supported")
		return
	}
	if c.gateway.RoomAccess == nil {
		c.sendError(clientMsgID, "unavailable", "room access check unavailable")
		return
	}
	allowed, err := c.gateway.RoomAccess.CanSend(ctx, req.GetRoomId(), c.UserID.String())
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", c.UserID.String()).
			Str("room_id", req.GetRoomId()).
			Str("client_msg_id", clientMsgID).
			Msg("check room access failed")
		c.sendError(clientMsgID, "unavailable", "room access check unavailable")
		return
	}
	if !allowed {
		c.sendError(clientMsgID, "room_access_denied", "room access denied")
		return
	}

	msgID, err := uuid.NewV7()
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client generate msg_id failed")
		c.sendError(clientMsgID, "internal", "failed to accept message")
		return
	}

	serverTime := time.Now().UnixMicro()
	msg, err := sendMessageReqToMessage(req, clientMsgID, c.UserID, msgID, serverTime)
	if err != nil {
		log.Error().
			Err(err).
			Str("user_id", c.UserID.String()).
			Str("client_msg_id", clientMsgID).
			Msg("client send message invalid")
		c.sendError(clientMsgID, "invalid_message", "invalid message")
		return
	}

	if err := c.gateway.MQ.GatewayEnqueueMessage(ctx, []*imv1.Message{msg}); err != nil {
		log.Error().
			Err(err).
			Str("user_id", c.UserID.String()).
			Str("client_msg_id", clientMsgID).
			Msg("client enqueue message failed")
		c.sendError(clientMsgID, "unavailable", "message queue unavailable")
		return
	}

	c.sendAck(clientMsgID, msgID.String(), serverTime)
}

func (c *UserClient) sendReady(sessionID string) {
	serverTime := time.Now().UnixMicro()
	c.sendFrame(&imv1.ServerFrame{
		Payload: &imv1.ServerFrame_Ready{Ready: &imv1.Ready{
			SessionId:  &sessionID,
			ServerTime: &serverTime,
		}},
	})
}

func (c *UserClient) sendAck(clientMsgID, msgID string, serverTime int64) {
	c.sendFrame(&imv1.ServerFrame{
		ClientMsgId: &clientMsgID,
		Payload: &imv1.ServerFrame_Ack{Ack: &imv1.SendMessageAck{
			MsgId:      &msgID,
			ServerTime: &serverTime,
		}},
	})
}

func (c *UserClient) sendError(clientMsgID, code, message string) {
	frame := &imv1.ServerFrame{
		Payload: &imv1.ServerFrame_Error{Error: &imv1.Error{
			Code:    &code,
			Message: &message,
		}},
	}
	if clientMsgID != "" {
		frame.ClientMsgId = &clientMsgID
	}
	c.sendFrame(frame)
}

func (c *UserClient) sendFrame(frame *imv1.ServerFrame) {
	bin, err := frame.MarshalVT()
	if err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("marshal ServerFrame failed")
		return
	}
	select {
	case c.Send <- bin:
	default:
		log.Warn().Str("user_id", c.UserID.String()).Msg("gateway client buffer full, dropping server frame")
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

func (s *UserSession) Broadcast(payloads []byte) {
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
