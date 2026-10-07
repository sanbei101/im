package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"uuid"

	"github.com/coder/websocket"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
)

type UserClient struct {
	gateway *Gateway
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  uuid.UUID
	frames  *render.FrameWriter

	closed atomic.Bool
}

func (c *UserClient) writePump(ctx context.Context) {
	defer func() {
		c.closed.Store(true)
		_ = c.Conn.Close(websocket.StatusNormalClosure, "")
	}()
	for frame := range c.Send {
		if err := c.Conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
			return
		}
	}
}

func (c *UserClient) readPump(ctx context.Context) {
	for {
		_, r, err := c.Conn.Reader(ctx)
		if err != nil {
			if websocket.CloseStatus(err) == -1 {
				log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("client read message failed")
			}
			return
		}
		// 硬性不变量：无论 decode 成败，reader 必须被读尽，
		// 否则 coder/websocket 会把剩余字节当成下一帧头，直接判协议错误断开连接。
		// drain 必须在本次迭代内同步完成，不能用 defer（循环里的 defer 只在函数返回时才执行）。
		if err := c.handleFrame(ctx, r); err != nil {
			c.sendError(err.Error())
		}
		if _, err := io.Copy(io.Discard, r); err != nil && ctx.Err() == nil {
			log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("drain client frame failed")
		}
	}
}

func (c *UserClient) handleFrame(ctx context.Context, r io.Reader) error {
	var input render.ClientFrame
	if err := render.NewFrameReader(r).ReadFrame(&input); err != nil {
		return fmt.Errorf("decode websocket message: %w", err)
	}
	if input.Type == "ping" {
		return c.encodeFrame(render.PongFrame{Type: "pong"})
	}
	if input.Type == "typing" {
		if input.RoomID == "" {
			return errors.New("room_id is required for typing")
		}
		c.gateway.TouchRoomUser(input.RoomID, c.UserID)
		c.gateway.BroadcastTyping(c.UserID, input.RoomID)
		return nil
	}
	if input.Type == "join_room" {
		if input.RoomID == "" {
			return errors.New("room_id is required")
		}
		c.gateway.TouchRoomUser(input.RoomID, c.UserID)
		return nil
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
	msgType, err := store.ParseMsgType(input.MsgType)
	if err != nil {
		return err
	}
	message := &pb.SendMessage{
		RequestId: requestID, ClientMsgId: input.ClientMsgID, SenderId: c.UserID.String(),
		RoomId: input.RoomID, MsgType: int32(msgType), Payload: input.Payload, Ext: input.Ext,
		ReplyToMsgId: input.ReplyToMsgID,
	}
	c.gateway.TouchRoomUser(input.RoomID, c.UserID)
	c.gateway.pending.Store(requestID, c)
	if err := c.gateway.send(ctx, roomID, message); err != nil {
		c.gateway.pending.Delete(requestID)
		return err
	}
	return nil
}

// sendFrame 非阻塞投递已编码帧；客户端已拆除或缓冲已满返回错误。
func (c *UserClient) sendFrame(frame []byte) error {
	if c.closed.Load() {
		return errors.New("client is closed")
	}
	select {
	case c.Send <- frame:
		return nil
	default:
		return errors.New("client send buffer is full")
	}
}

// encodeFrame 编码一帧并投递到 Send；编码失败或缓冲已满时返回错误。
func (c *UserClient) encodeFrame[T any](v T) error {
	frame, err := c.frames.EncodeFrame(v)
	if err != nil {
		return err
	}
	return c.sendFrame(frame)
}

func (c *UserClient) sendError(message string) {
	if err := c.encodeFrame(render.ErrorFrame{Type: "error", Error: message}); err != nil {
		log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("send error frame failed")
	}
}

const sessionShardCount = 256

type sessionShard struct {
	mu sync.RWMutex
	m  map[uuid.UUID]*UserSession
}

type UserSessionManager struct {
	shards [sessionShardCount]sessionShard
}

func NewSessionManager() *UserSessionManager {
	manager := &UserSessionManager{}
	for i := range manager.shards {
		manager.shards[i].m = make(map[uuid.UUID]*UserSession)
	}
	return manager
}

func (manager *UserSessionManager) shard(id uuid.UUID) *sessionShard {
	return &manager.shards[id[15]]
}

func (manager *UserSessionManager) LoadOrCreate(id uuid.UUID, create func() *UserSession) *UserSession {
	shard := manager.shard(id)
	shard.mu.RLock()
	session := shard.m[id]
	shard.mu.RUnlock()
	if session != nil {
		return session
	}
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if session = shard.m[id]; session == nil {
		session = create()
		shard.m[id] = session
	}
	return session
}

func (manager *UserSessionManager) Delete(id uuid.UUID) {
	shard := manager.shard(id)
	shard.mu.Lock()
	delete(shard.m, id)
	shard.mu.Unlock()
}

func (manager *UserSessionManager) Load(id uuid.UUID) (*UserSession, bool) {
	shard := manager.shard(id)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	value, ok := shard.m[id]
	return value, ok
}

func (manager *UserSessionManager) All() []uuid.UUID {
	var result []uuid.UUID
	for i := range manager.shards {
		shard := &manager.shards[i]
		shard.mu.RLock()
		for id := range shard.m {
			result = append(result, id)
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

// Clients 返回会话内全部客户端的快照（调用方遍历发送）。
func (s *UserSession) Clients() []*UserClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	clients := make([]*UserClient, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	return clients
}
