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
	// sendBatchSize bounds one coalesced SendBatch frame; messages beyond it
	// stay queued for the next frame.
	sendBatchSize = 64
	// roomUserTTL bounds how long a room's active-user entry lives without a
	// heartbeat; the sweeper evicts stale entries so the maps stay bounded.
	roomUserTTL = 24 * time.Hour
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
	return &g.pendingShards[pkg.StringShard(requestID, pendingShardCount)]
}

func (g *Gateway) getRoomUserShard(roomID string) *roomUserShard {
	return &g.roomUserShards[pkg.StringShard(roomID, roomUserShardCount)]
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
	go g.sweepRoomUsers(ctx)
}

// sweepRoomUsers periodically evicts room-user entries idle past roomUserTTL
// and drops empty sets, keeping the room-user maps bounded.
func (g *Gateway) sweepRoomUsers(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		cutoff := time.Now().Add(-roomUserTTL)
		for i := range g.roomUserShards {
			shard := &g.roomUserShards[i]
			shard.mu.Lock()
			for roomID, set := range shard.m {
				set.mu.Lock()
				for uid, lastSeen := range set.users {
					if lastSeen.Before(cutoff) {
						delete(set.users, uid)
					}
				}
				empty := len(set.users) == 0
				set.mu.Unlock()
				if empty {
					delete(shard.m, roomID)
				}
			}
			shard.mu.Unlock()
		}
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
	client   gatewayservice.Client // created once per stream, reused across reconnects
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
		if err := s.ensureClient(); err != nil {
			log.Error().Err(err).Str("api", s.address).Msg("create api client failed")
			if !s.waitRetry(ctx) {
				return
			}
			continue
		}
		if err := s.connect(ctx); err != nil {
			log.Error().Err(err).Str("api", s.address).Msg("connect api stream failed")
			if !s.waitRetry(ctx) {
				return
			}
			continue
		}
		if err := s.loop(ctx); err != nil {
			log.Error().Err(err).Str("api", s.address).Msg("api stream stopped")
		}
		s.close()
	}
}

// waitRetry sleeps before the next reconnect attempt; false when ctx is done.
func (s *apiStream) waitRetry(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(time.Second):
		return true
	}
}

// ensureClient lazily creates the Kitex client once. The generated client
// interface does not expose Close, but one client per API address lives for
// the process lifetime, so there is nothing to leak across reconnects.
func (s *apiStream) ensureClient() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		return nil
	}
	cli, err := gatewayservice.NewClient("im-api", client.WithHostPorts(s.address))
	if err != nil {
		return err
	}
	s.client = cli
	return nil
}

func (s *apiStream) connect(ctx context.Context) error {
	cli := s.snapshotClient()
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
		s.close()
		return err
	}
	// Replay every local session in one batch so the API node restores its
	// routing table after a reconnect with a single round trip.
	sessions := s.gateway.UserSessionManager.All()
	if len(sessions) > 0 {
		batch := make([]*pb.Session, 0, len(sessions))
		for _, session := range sessions {
			batch = append(batch, &pb.Session{UserId: session.String(), Online: true})
		}
		if err := stream.Send(ctx,
			&pb.GatewayFrame{
				Body: &pb.GatewayFrame_SessionBatch{
					SessionBatch: &pb.SessionBatch{Sessions: batch},
				},
			},
		); err != nil {
			s.close()
			return err
		}
	}
	return nil
}

// loop drives the live stream with one sender and one receiver goroutine. It
// joins both before returning, so no goroutine can outlive this loop and
// touch the next generation's stream.
func (s *apiStream) loop(ctx context.Context) error {
	stream := s.snapshot()
	if stream == nil {
		return errors.New("api stream not connected")
	}
	done := make(chan error, 2)
	stop := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		// batch is owned by this goroutine and reused across frames; Send
		// serializes synchronously before it returns.
		var batch []*pb.SendMessage
		for {
			select {
			case <-ctx.Done():
				done <- ctx.Err()
				return
			case <-stop:
				return
			case message := <-s.queue:
				// Coalesce whatever is already queued into one frame: the
				// non-blocking drain batches only under backpressure, so
				// low-load latency is unchanged.
				batch = append(batch[:0], message)
			drain:
				for len(batch) < sendBatchSize {
					select {
					case next := <-s.queue:
						batch = append(batch, next)
					default:
						break drain
					}
				}
				if err := stream.Send(ctx, &pb.GatewayFrame{Body: &pb.GatewayFrame_SendBatch{
					SendBatch: &pb.SendBatch{BatchId: uuid.NewV7().String(), Messages: batch},
				}}); err != nil {
					done <- err
					return
				}
			case session := <-s.sessions:
				if err := stream.Send(ctx, &pb.GatewayFrame{Body: &pb.GatewayFrame_SessionBatch{
					SessionBatch: &pb.SessionBatch{Sessions: []*pb.Session{session}},
				}}); err != nil {
					done <- err
					return
				}
			}
		}
	}()
	go func() {
		defer wg.Done()
		for {
			frame, err := stream.Recv(ctx)
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
	err := <-done
	close(stop)
	// End the send direction so the API node finishes the RPC and the blocked
	// Recv above returns; teardown happens before Wait so no goroutine is left
	// holding a dying stream. A send already in flight errors out on the broken
	// stream, which is the same condition that triggered this teardown.
	if closeErr := stream.CloseSend(context.Background()); closeErr != nil && err == nil {
		err = closeErr
	}
	wg.Wait()
	return err
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

func (s *apiStream) snapshotClient() gatewayservice.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client
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
