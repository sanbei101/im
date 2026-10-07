package gateway

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"sync"
	"time"
	"uuid"

	"github.com/cloudwego/kitex/client"
	"github.com/phuslu/log"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/render"
	"github.com/sanbei101/im/proto/pb"
	"github.com/sanbei101/im/proto/pb/gatewayservice"
)

const (
	pendingShardCount  = 64
	roomUserShardCount = 64
)

type pendingShard struct {
	mu sync.Mutex
	m  map[string]*UserClient
}

type roomUserShard struct {
	mu sync.RWMutex
	m  map[string]*roomUserSet
}

type Gateway struct {
	UserSessionManager *UserSessionManager
	Config             *config.Config
	streams            []*apiStream
	pendingShards      [pendingShardCount]pendingShard
	roomUserShards     [roomUserShardCount]roomUserShard
	frames             *render.FrameWriter
}

type roomUserSet struct {
	mu    sync.RWMutex
	users map[uuid.UUID]time.Time
}

func (g *Gateway) getPendingShard(requestID string) *pendingShard {
	var hVal uint32
	for i := 0; i < len(requestID); i++ {
		hVal = hVal*31 + uint32(requestID[i])
	}
	return &g.pendingShards[hVal%pendingShardCount]
}

func (g *Gateway) getRoomUserShard(roomID string) *roomUserShard {
	var hVal uint32
	for i := 0; i < len(roomID); i++ {
		hVal = hVal*31 + uint32(roomID[i])
	}
	return &g.roomUserShards[hVal%roomUserShardCount]
}

func (g *Gateway) StorePending(requestID string, uc *UserClient) {
	shard := g.getPendingShard(requestID)
	shard.mu.Lock()
	shard.m[requestID] = uc
	shard.mu.Unlock()
}

func (g *Gateway) DeletePending(requestID string) {
	shard := g.getPendingShard(requestID)
	shard.mu.Lock()
	delete(shard.m, requestID)
	shard.mu.Unlock()
}

func (g *Gateway) LoadAndDeletePending(requestID string) *UserClient {
	shard := g.getPendingShard(requestID)
	shard.mu.Lock()
	uc := shard.m[requestID]
	delete(shard.m, requestID)
	shard.mu.Unlock()
	return uc
}

func NewGateway(cfg *config.Config) *Gateway {
	g := &Gateway{
		UserSessionManager: NewSessionManager(),
		Config:             cfg,
		frames:             render.NewFrameWriter(),
	}
	for i := range g.pendingShards {
		g.pendingShards[i].m = make(map[string]*UserClient)
	}
	for i := range g.roomUserShards {
		g.roomUserShards[i].m = make(map[string]*roomUserSet)
	}
	for index, address := range cfg.Gateway.APIAddrs {
		g.streams = append(g.streams, newAPIStream(g, fmt.Sprintf("gateway-%d", index), address))
	}
	return g
}

func (g *Gateway) TouchRoomUser(roomID string, userID uuid.UUID) {
	shard := g.getRoomUserShard(roomID)
	shard.mu.RLock()
	set := shard.m[roomID]
	shard.mu.RUnlock()
	if set == nil {
		shard.mu.Lock()
		set = shard.m[roomID]
		if set == nil {
			set = &roomUserSet{users: make(map[uuid.UUID]time.Time)}
			shard.m[roomID] = set
		}
		shard.mu.Unlock()
	}
	set.mu.Lock()
	set.users[userID] = time.Now()
	set.mu.Unlock()
}

func (g *Gateway) BroadcastTyping(senderID uuid.UUID, roomID string) {
	shard := g.getRoomUserShard(roomID)
	shard.mu.RLock()
	set := shard.m[roomID]
	shard.mu.RUnlock()
	if set == nil {
		return
	}
	set.mu.RLock()
	var targets []uuid.UUID
	cutoff := time.Now().Add(-24 * time.Hour)
	for uid, lastSeen := range set.users {
		if uid != senderID && lastSeen.After(cutoff) {
			targets = append(targets, uid)
		}
	}
	set.mu.RUnlock()

	if len(targets) == 0 {
		return
	}

	frame := render.TypingFrame{
		Type:   "typing",
		RoomID: roomID,
		UserID: senderID.String(),
	}
	frameBytes, err := g.frames.EncodeFrame(frame)
	if err != nil {
		log.Error().Err(err).Msg("encode typing frame failed")
		return
	}
	for _, targetID := range targets {
		if session, ok := g.UserSessionManager.Load(targetID); ok {
			for _, client := range session.Clients() {
				if err := client.sendFrame(frameBytes); err != nil {
					log.Error().Err(err).Str("user_id", client.UserID.String()).Msg("send typing frame failed")
				}
			}
		}
	}
}

func (g *Gateway) Start(ctx context.Context) {
	for _, stream := range g.streams {
		go stream.run(ctx)
	}
}

func (g *Gateway) streamForRoom(roomID uuid.UUID) *apiStream {
	if len(g.streams) == 0 {
		return nil
	}
	slot, err := pkg.RoomSlot(roomID, g.Config.Shard.Slots)
	if err != nil {
		return nil
	}
	index, err := pkg.NodeIndex(slot, g.Config.Shard.Slots, len(g.streams))
	if err != nil {
		return nil
	}
	return g.streams[index]
}

func (g *Gateway) send(ctx context.Context, roomID uuid.UUID, message *pb.SendMessage) error {
	stream := g.streamForRoom(roomID)
	if stream == nil {
		return errors.New("no api stream available")
	}
	return stream.enqueue(ctx, message)
}

func (g *Gateway) registerUser(ctx context.Context, userID uuid.UUID, online bool) {
	for _, stream := range g.streams {
		if err := stream.register(ctx, userID, online); err != nil {
			log.Error().Err(err).Str("user_id", userID.String()).Msg("register gateway session failed")
		}
	}
}

type apiStream struct {
	gateway  *Gateway
	id       string
	address  string
	stream   gatewayservice.GatewayService_ConnectClient
	queue    chan *pb.SendMessage
	sessions chan *pb.Session
	mu       sync.RWMutex
	frames   *render.FrameWriter
}

func newAPIStream(g *Gateway, id, address string) *apiStream {
	return &apiStream{
		gateway:  g,
		id:       id,
		address:  address,
		queue:    make(chan *pb.SendMessage, 1024),
		sessions: make(chan *pb.Session, 256),
		frames:   render.NewFrameWriter(),
	}
}

func (s *apiStream) run(ctx context.Context) {
	for ctx.Err() == nil {
		if err := s.connect(ctx); err != nil {
			log.Error().Err(err).Str("api", s.address).Msg("connect api stream failed")
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}
		if err := s.loop(ctx); err != nil {
			log.Error().Err(err).Str("api", s.address).Msg("api stream stopped")
		}
		s.close()
	}
}

func (s *apiStream) connect(ctx context.Context) error {
	cli, err := gatewayservice.NewClient("im-api", client.WithHostPorts(s.address))
	if err != nil {
		return err
	}
	stream, err := cli.Connect(ctx)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.stream = stream
	s.mu.Unlock()
	if err := stream.Send(ctx,
		&pb.GatewayFrame{Body: &pb.GatewayFrame_Hello{Hello: &pb.Hello{GatewayId: s.id}}},
	); err != nil {
		return err
	}
	for _, session := range s.gateway.UserSessionManager.All() {
		if err := stream.Send(ctx,
			&pb.GatewayFrame{
				Body: &pb.GatewayFrame_SessionBatch{
					SessionBatch: &pb.SessionBatch{
						Sessions: []*pb.Session{{UserId: session.String(), Online: true}},
					},
				},
			},
		); err != nil {
			return err
		}
	}
	return nil
}

func (s *apiStream) loop(ctx context.Context) error {
	done := make(chan error, 1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return
			case message := <-s.queue:
				if err := s.sendFrame(ctx, &pb.GatewayFrame{Body: &pb.GatewayFrame_SendBatch{
					SendBatch: &pb.SendBatch{BatchId: uuid.NewV7().String(), Messages: []*pb.SendMessage{message}},
				}}); err != nil {
					done <- err
					return
				}
			case session := <-s.sessions:
				if err := s.sendFrame(ctx, &pb.GatewayFrame{Body: &pb.GatewayFrame_SessionBatch{
					SessionBatch: &pb.SessionBatch{Sessions: []*pb.Session{session}},
				}}); err != nil {
					done <- err
					return
				}
			}
		}
	}()
	go func() {
		for {
			frame, err := s.stream.Recv(ctx)
			if err != nil {
				done <- err
				return
			}
			switch {
			case frame.GetSendResultBatch() != nil:
				s.handleResults(frame.GetSendResultBatch())
			case frame.GetPushBatch() != nil:
				s.handlePush(frame.GetPushBatch())
			}
		}
	}()
	return <-done
}

// sendFrame sends one frame on the live stream.
func (s *apiStream) sendFrame(ctx context.Context, frame *pb.GatewayFrame) error {
	return s.snapshot().Send(ctx, frame)
}

func (s *apiStream) register(ctx context.Context, userID uuid.UUID, online bool) error {
	select {
	case s.sessions <- &pb.Session{UserId: userID.String(), Online: online}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("session queue is full")
	}
}

func (s *apiStream) handleResults(batch *pb.SendResultBatch) {
	for _, result := range batch.GetResults() {
		userClient := s.gateway.LoadAndDeletePending(result.GetRequestId())
		if userClient == nil {
			continue
		}
		if err := userClient.encodeFrame(render.AckFrame{
			Type:        "ack",
			RequestID:   result.GetRequestId(),
			ClientMsgID: result.GetClientMsgId(),
			MsgID:       result.GetMsgId(),
			RoomID:      result.GetRoomId(),
			RoomSeq:     result.GetRoomSeq(),
			ServerTime:  result.GetServerTime(),
			Code:        result.GetCode(),
			Error:       result.GetError(),
		}); err != nil {
			log.Error().Err(err).Msg("send message result to websocket failed")
		}
	}
}

func (s *apiStream) handlePush(batch *pb.PushBatch) {
	for _, push := range batch.GetPushes() {
		if push == nil {
			continue
		}
		userID, err := uuid.Parse(push.GetUserId())
		if err != nil {
			continue
		}
		session, ok := s.gateway.UserSessionManager.Load(userID)
		if !ok {
			continue
		}
		clients := session.Clients()
		if len(clients) == 0 {
			continue
		}
		frame := render.PushFrame{
			Type:         "message",
			MsgID:        push.GetMsgId(),
			ClientMsgID:  push.GetClientMsgId(),
			SenderID:     push.GetSenderId(),
			RoomID:       push.GetRoomId(),
			RoomSeq:      push.GetRoomSeq(),
			ServerTime:   push.GetServerTime(),
			MsgType:      store.MsgType(push.GetMsgType()).String(),
			Payload:      jsontext.Value(push.GetPayload()),
			ReplyToMsgID: push.GetReplyToMsgId(),
			Ext:          jsontext.Value(push.GetExt()),
		}
		frameBytes, err := s.frames.EncodeFrame(frame)
		if err != nil {
			log.Error().Err(err).Msg("encode push frame failed")
			continue
		}
		for _, client := range clients {
			if err := client.sendFrame(frameBytes); err != nil {
				log.Error().Err(err).Str("user_id", client.UserID.String()).Msg("send push frame to websocket failed")
			}
		}
	}
}

func (s *apiStream) enqueue(ctx context.Context, message *pb.SendMessage) error {
	select {
	case s.queue <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *apiStream) close() {
	s.mu.Lock()
	s.stream = nil
	s.mu.Unlock()
}

// snapshot returns the live stream; nil means the stream was torn down by close().
func (s *apiStream) snapshot() gatewayservice.GatewayService_ConnectClient {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stream
}
