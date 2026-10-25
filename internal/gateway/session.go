package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
	"uuid"

	"github.com/coder/websocket"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/pkg/engine"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
)

const (
	heartbeatInterval = 30 * time.Second
	heartbeatTimeout  = 5 * time.Second
)

type UserClient struct {
	gateway *Gateway
	Conn    *websocket.Conn
	Send    chan []byte
	UserID  uuid.UUID
	frames  *render.FrameWriter
	reader  *render.FrameReader

	sendMu sync.Mutex
	closed bool
}

func (c *UserClient) writePump(ctx context.Context) {
	defer func() {
		c.Conn.Close(websocket.StatusNormalClosure, "")
	}()
	for frame := range c.Send {
		if err := c.Conn.Write(ctx, websocket.MessageBinary, frame); err != nil {
			return
		}
	}
}

// pingPump proves the peer is alive with WebSocket control frames. The client
// stack answers pong automatically; a missed pong closes the connection so the
// normal offline path runs.
func (c *UserClient) pingPump(ctx context.Context, interval, timeout time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, timeout)
			err := c.Conn.Ping(pingCtx)
			cancel()
			if err == nil {
				continue
			}
			if ctx.Err() != nil {
				return
			}
			log.Warn().Err(err).Str("user_id", c.UserID.String()).Msg("websocket ping failed")
			if closeErr := c.Conn.CloseNow(); closeErr != nil {
				log.Debug().
					Err(closeErr).
					Str("user_id", c.UserID.String()).
					Msg("close websocket after ping failure")
			}
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
		// Hard invariant: the reader must be drained to EOF whether or not
		// decode succeeded, otherwise coder/websocket treats the leftover
		// bytes as the next frame header and drops the connection with a
		// protocol error. The drain must complete synchronously in this
		// iteration; a defer would only run when readPump returns.
		if err := c.handleFrame(ctx, r); err != nil {
			c.sendError(err.Error())
		}
		if _, err := io.Copy(io.Discard, r); err != nil && ctx.Err() == nil {
			log.Error().Err(err).Str("user_id", c.UserID.String()).Msg("drain client frame failed")
		}
	}
}

func (c *UserClient) handleFrame(ctx context.Context, r io.Reader) error {
	if c.reader == nil {
		c.reader = render.NewFrameReader(r)
	} else {
		c.reader.Reset(r)
	}
	input, err := c.reader.ReadFrame[render.ClientFrame]()
	if err != nil {
		return fmt.Errorf("decode websocket message: %w", err)
	}
	if input.Type == "typing" || input.Type == "join_room" {
		roomID, err := uuid.Parse(input.RoomID)
		if err != nil {
			return errors.New("room_id must be a valid uuid")
		}
		// The canonical string is the map key: two spellings of one UUID must
		// not grow two presence entries.
		roomIDStr := roomID.String()
		c.gateway.TouchRoomUser(roomIDStr, c.UserID)
		if input.Type == "typing" {
			c.gateway.BroadcastTyping(c.UserID, roomIDStr)
		}
		return nil
	}
	if input.Type != "message" {
		return fmt.Errorf("unsupported frame type %q", input.Type)
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
	msgType, err := engine.ParseMsgType(input.MsgType)
	if err != nil {
		return err
	}
	message := &pb.SendMessage{
		RequestId: requestID, ClientMsgId: input.ClientMsgID, SenderId: c.UserID.String(),
		RoomId: input.RoomID, MsgType: int32(msgType), Payload: input.Payload, Ext: input.Ext,
		ReplyToSeq: input.ReplyToSeq,
	}
	c.gateway.TouchRoomUser(roomID.String(), c.UserID)
	c.gateway.StorePending(requestID, c)
	if err := c.gateway.send(ctx, roomID, message); err != nil {
		c.gateway.DeletePending(requestID)
		return err
	}
	return nil
}

// sendFrame delivers an already-encoded frame without blocking; it fails when
// the client is torn down or its send buffer is full.
func (c *UserClient) sendFrame(frame []byte) error {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if c.closed {
		return errors.New("client is closed")
	}
	select {
	case c.Send <- frame:
		return nil
	default:
		return errors.New("client send buffer is full")
	}
}

func (c *UserClient) closeSend() {
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.Send)
	}
}

// encodeFrame encodes one frame and hands it to Send; it fails when encoding
// fails or the buffer is full.
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

// CloseAll closes every live websocket across all sessions, used at shutdown:
// each read pump fails immediately and its handler unwinds through the
// normal cleanup path (offline registration, session removal).
func (manager *UserSessionManager) CloseAll() {
	for i := range manager.shards {
		shard := &manager.shards[i]
		shard.mu.RLock()
		for _, session := range shard.m {
			session.ForEachClient(func(client *UserClient) {
				client.Conn.Close(websocket.StatusGoingAway, "server shutting down")
			})
		}
		shard.mu.RUnlock()
	}
}

func (manager *UserSessionManager) RemoveClient(id uuid.UUID, client *UserClient) bool {
	shard := manager.shard(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	session, ok := shard.m[id]
	if !ok {
		return false
	}
	session.mu.Lock()
	delete(session.clients, client)
	empty := len(session.clients) == 0
	session.mu.Unlock()
	if empty {
		delete(shard.m, id)
	}
	return empty
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

// ForEachClient traverses all clients in the session without allocating a slice.
func (s *UserSession) ForEachClient(fn func(*UserClient)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for client := range s.clients {
		fn(client)
	}
}
