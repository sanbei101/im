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
	imv1 "github.com/sanbei101/im/kitex_gen/im/v1"
	"github.com/sanbei101/im/kitex_gen/im/v1/gatewayservice"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/pkg/config"
	"github.com/sanbei101/im/pkg/render"
)

type Gateway struct {
	UserSessionManager *UserSessionManager
	Config             *config.Config
	streams            []*apiStream
	pending            sync.Map
}

func NewGateway(cfg *config.Config) *Gateway {
	g := &Gateway{UserSessionManager: NewSessionManager(), Config: cfg}
	for index, address := range cfg.Gateway.APIAddrs {
		g.streams = append(g.streams, newAPIStream(g, fmt.Sprintf("gateway-%d", index), address))
	}
	return g
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

func (g *Gateway) send(ctx context.Context, roomID uuid.UUID, message *imv1.SendMessage) error {
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
	queue    chan *imv1.SendMessage
	sessions chan *imv1.Session
	mu       sync.RWMutex
}

func newAPIStream(g *Gateway, id, address string) *apiStream {
	return &apiStream{
		gateway:  g,
		id:       id,
		address:  address,
		queue:    make(chan *imv1.SendMessage, 1024),
		sessions: make(chan *imv1.Session, 256),
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
		&imv1.GatewayFrame{Body: &imv1.GatewayFrame_Hello{Hello: &imv1.Hello{GatewayId: s.id}}},
	); err != nil {
		return err
	}
	for _, session := range s.gateway.UserSessionManager.All() {
		if err := stream.Send(ctx,
			&imv1.GatewayFrame{
				Body: &imv1.GatewayFrame_SessionBatch{
					SessionBatch: &imv1.SessionBatch{
						Sessions: []*imv1.Session{{UserId: session.String(), Online: true}},
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
				if err := s.sendFrame(ctx, &imv1.GatewayFrame{Body: &imv1.GatewayFrame_SendBatch{
					SendBatch: &imv1.SendBatch{BatchId: uuid.NewV7().String(), Messages: []*imv1.SendMessage{message}},
				}}); err != nil {
					done <- err
					return
				}
			case session := <-s.sessions:
				if err := s.sendFrame(ctx, &imv1.GatewayFrame{Body: &imv1.GatewayFrame_SessionBatch{
					SessionBatch: &imv1.SessionBatch{Sessions: []*imv1.Session{session}},
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
func (s *apiStream) sendFrame(ctx context.Context, frame *imv1.GatewayFrame) error {
	return s.snapshot().Send(ctx, frame)
}

func (s *apiStream) register(ctx context.Context, userID uuid.UUID, online bool) error {
	select {
	case s.sessions <- &imv1.Session{UserId: userID.String(), Online: online}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("session queue is full")
	}
}

func (s *apiStream) handleResults(batch *imv1.SendResultBatch) {
	for _, result := range batch.GetResults() {
		value, ok := s.gateway.pending.LoadAndDelete(result.GetRequestId())
		if !ok {
			continue
		}
		userClient, ok := value.(*UserClient)
		if !ok {
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

func (s *apiStream) handlePush(batch *imv1.PushBatch) {
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
		frame := render.PushFrame{
			Type:        "message",
			MsgID:       push.GetMsgId(),
			ClientMsgID: push.GetClientMsgId(),
			SenderID:    push.GetSenderId(),
			RoomID:      push.GetRoomId(),
			RoomSeq:     push.GetRoomSeq(),
			ServerTime:  push.GetServerTime(),
			MsgType:     store.MsgType(push.GetMsgType()).String(),
			Payload:     jsontext.Value(push.GetPayload()),
			Ext:         jsontext.Value(push.GetExt()),
		}
		for _, client := range session.Clients() {
			if err := client.encodeFrame(frame); err != nil {
				log.Error().Err(err).Str("user_id", client.UserID.String()).Msg("send push frame to websocket failed")
			}
		}
	}
}

func (s *apiStream) enqueue(ctx context.Context, message *imv1.SendMessage) error {
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
