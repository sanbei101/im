package api

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
	"uuid"

	"github.com/phuslu/log"
	"github.com/phuslu/lru"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/proto/pb"
)

const (
	sessionShardCount = 64
)

type sessionShard struct {
	mu       sync.RWMutex
	sessions map[string]*apiConnection
}

type StreamHandler struct {
	store     *store.Store
	nodeID    string
	slots     int
	nodeIndex int
	nodeCount int

	sessionShards [sessionShardCount]sessionShard
	memberCache   *lru.TTLCache[uuid.UUID, []cachedMember]
}

type cachedMember struct {
	userID    uuid.UUID
	userIDStr string
}

type apiConnection struct {
	stream pb.GatewayService_ConnectServer
	// outbox decouples producers from the network: callers enqueue without
	// blocking, and the connection's writer goroutine serializes every Send.
	outbox chan *pb.APIFrame
	// quit is closed when the Connect handler returns; it stops the writer.
	quit chan struct{}

	mu    sync.Mutex // guards users
	users map[string]struct{}
}

// connectionOutboxSize bounds a gateway connection's queued frames; beyond it
// the gateway is not draining fast enough and frames are dropped.
const connectionOutboxSize = 1024

var (
	errConnectionClosed = errors.New("gateway connection closed")
	errOutboxFull       = errors.New("gateway connection outbox is full")
)

func newAPIConnection(stream pb.GatewayService_ConnectServer) *apiConnection {
	return &apiConnection{
		stream: stream,
		users:  make(map[string]struct{}),
		outbox: make(chan *pb.APIFrame, connectionOutboxSize),
		quit:   make(chan struct{}),
	}
}

// enqueue hands a frame to the connection writer and returns immediately, so
// a slow gateway can never stall another gateway's message loop or an HTTP
// handler. Pushes are lossy by design - clients catch up via room_seq pull.
func (c *apiConnection) enqueue(frame *pb.APIFrame) error {
	select {
	case c.outbox <- frame:
		return nil
	case <-c.quit:
		return errConnectionClosed
	default:
		return errOutboxFull
	}
}

// writeOutbox is the single sender of one gateway connection: it drains the
// outbox onto the stream so no other goroutine touches stream.Send.
func (c *apiConnection) writeOutbox(ctx context.Context) {
	for {
		select {
		case <-c.quit:
			return
		case frame := <-c.outbox:
			if err := c.stream.Send(ctx, frame); err != nil {
				return // the stream is broken; Connect's Recv fails on it too
			}
		}
	}
}

func (h *StreamHandler) getSessionShard(userID string) *sessionShard {
	return &h.sessionShards[pkg.StringShard(userID, sessionShardCount)]
}

func NewStreamHandler(data *store.Store, nodeID string, slots, nodeIndex, nodeCount int) *StreamHandler {
	handler := &StreamHandler{
		store:     data,
		nodeID:    nodeID,
		slots:     slots,
		nodeIndex: nodeIndex,
		nodeCount: nodeCount,
		memberCache: lru.NewTTLCache[uuid.UUID, []cachedMember](
			4096,
			lru.WithShards[uuid.UUID, []cachedMember](64),
		),
	}
	for i := range handler.sessionShards {
		handler.sessionShards[i].sessions = make(map[string]*apiConnection)
	}
	return handler
}

func (h *StreamHandler) InvalidateRoomMembers(roomID uuid.UUID) {
	h.memberCache.Delete(roomID)
}

func (h *StreamHandler) getRoomMembers(ctx context.Context, roomID uuid.UUID) ([]cachedMember, error) {
	if cached, ok := h.memberCache.Get(roomID); ok {
		return cached, nil
	}

	rawMembers, err := h.store.Members(ctx, roomID)
	if err != nil {
		return nil, err
	}
	cached := make([]cachedMember, len(rawMembers))
	for i, m := range rawMembers {
		cached[i] = cachedMember{
			userID:    m.UserID,
			userIDStr: m.UserID.String(),
		}
	}
	h.memberCache.Set(roomID, cached, 30*time.Second)
	return cached, nil
}

func (h *StreamHandler) Connect(ctx context.Context, stream pb.GatewayService_ConnectServer) error {
	connection := newAPIConnection(stream)
	defer h.removeConnection(connection)
	defer close(connection.quit)
	go connection.writeOutbox(ctx)

	for {
		frame, err := stream.Recv(ctx)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("receive gateway frame: %w", err)
		}
		switch {
		case frame.GetHello() != nil:
			if err := connection.enqueue(&pb.APIFrame{
				Body: &pb.APIFrame_HelloAck{
					HelloAck: &pb.HelloAck{NodeId: h.nodeID, TopologyVersion: strconv.Itoa(h.slots)},
				},
			}); err != nil {
				return fmt.Errorf("send hello ack: %w", err)
			}
		case frame.GetSessionBatch() != nil:
			h.updateSessions(connection, frame.GetSessionBatch())
		case frame.GetSendBatch() != nil:
			result, pushes, err := h.writeBatch(ctx, frame.GetSendBatch())
			if err != nil {
				return err
			}
			if err := connection.enqueue(&pb.APIFrame{
				Body: &pb.APIFrame_SendResultBatch{SendResultBatch: result},
			}); err != nil {
				return fmt.Errorf("send result batch: %w", err)
			}
			for _, push := range pushes {
				if err := h.Push(ctx, push); err != nil {
					log.Error().Err(err).Msg("push room message failed")
				}
			}
		}
	}
}

func (h *StreamHandler) updateSessions(connection *apiConnection, batch *pb.SessionBatch) {
	for _, session := range batch.GetSessions() {
		if session == nil {
			continue
		}
		userID := session.GetUserId()
		shard := h.getSessionShard(userID)
		shard.mu.Lock()
		if session.GetOnline() {
			shard.sessions[userID] = connection
			connection.mu.Lock()
			connection.users[userID] = struct{}{}
			connection.mu.Unlock()
		} else if current := shard.sessions[userID]; current == connection {
			delete(shard.sessions, userID)
			connection.mu.Lock()
			delete(connection.users, userID)
			connection.mu.Unlock()
		}
		shard.mu.Unlock()
	}
}

func (h *StreamHandler) removeConnection(connection *apiConnection) {
	connection.mu.Lock()
	users := make([]string, 0, len(connection.users))
	for userID := range connection.users {
		users = append(users, userID)
	}
	connection.mu.Unlock()

	for _, userID := range users {
		shard := h.getSessionShard(userID)
		shard.mu.Lock()
		if current := shard.sessions[userID]; current == connection {
			delete(shard.sessions, userID)
		}
		shard.mu.Unlock()
	}
}

func (h *StreamHandler) writeBatch(
	ctx context.Context,
	batch *pb.SendBatch,
) (*pb.SendResultBatch, []*store.Message, error) {
	result := &pb.SendResultBatch{BatchId: batch.GetBatchId()}
	inputs := batch.GetMessages()
	messages := make([]store.Message, 0, len(inputs))
	resultIndexes := make([]int, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		item := &pb.SendResult{
			RequestId: input.GetRequestId(), ClientMsgId: input.GetClientMsgId(), RoomId: input.GetRoomId(),
		}
		result.Results = append(result.Results, item)
		message, err := h.message(input)
		if err == nil {
			err = h.checkRoomOwned(message.RoomID)
		}
		if err != nil {
			item.Code = 1
			item.Error = err.Error()
			continue
		}
		messages = append(messages, message)
		resultIndexes = append(resultIndexes, len(result.Results)-1)
	}
	written := make([]*store.Message, 0, len(messages))
	if len(messages) == 0 {
		return result, written, nil
	}
	writeResults := h.store.WriteMessages(ctx, messages)
	for index := range writeResults {
		writtenResult := &writeResults[index]
		item := result.Results[resultIndexes[index]]
		if writtenResult.Err != nil {
			item.Code = 1
			item.Error = writtenResult.Err.Error()
			continue
		}
		message := writtenResult.Message
		item.MsgId = message.MsgID.String()
		item.RoomSeq = message.RoomSeq
		item.ServerTime = message.ServerTime
		written = append(written, &message)
	}
	return result, written, nil
}

// checkRoomOwned rejects rooms whose shard slot is not served by this node.
func (h *StreamHandler) checkRoomOwned(roomID uuid.UUID) error {
	slot, err := pkg.RoomSlot(roomID, h.slots)
	if err != nil {
		return err
	}
	owner, err := pkg.NodeIndex(slot, h.slots, h.nodeCount)
	if err != nil {
		return err
	}
	if owner != h.nodeIndex {
		return fmt.Errorf("room is owned by api node index %d", owner)
	}
	return nil
}

func (h *StreamHandler) push(ctx context.Context, message *store.Message, exclude uuid.UUID) error {
	members, err := h.getRoomMembers(ctx, message.RoomID)
	if err != nil {
		return fmt.Errorf("load room members: %w", err)
	}

	roomIDStr := message.RoomID.String()
	msgIDStr := message.MsgID.String()
	senderIDStr := message.SenderID.String()
	clientMsgIDStr := message.ClientMsgID.String()
	var replyToMsgIDStr string
	if message.ReplyToMsgID != uuid.Nil() {
		replyToMsgIDStr = message.ReplyToMsgID.String()
	}

	// Group online recipients per gateway connection; payload is sent once per gateway.
	targets := make(map[*apiConnection][]string)
	for _, member := range members {
		if exclude != uuid.Nil() && member.userID == exclude {
			continue
		}
		if connection := h.onlineConnection(member.userIDStr); connection != nil {
			targets[connection] = append(targets[connection], member.userIDStr)
		}
	}

	for connection, userIDs := range targets {
		roomPush := &pb.RoomPush{
			RoomId:       roomIDStr,
			RoomSeq:      message.RoomSeq,
			MsgId:        msgIDStr,
			SenderId:     senderIDStr,
			MsgType:      int32(message.MsgType),
			Payload:      message.Payload,
			ServerTime:   message.ServerTime,
			Ext:          message.Ext,
			ClientMsgId:  clientMsgIDStr,
			ReplyToMsgId: replyToMsgIDStr,
			UserIds:      userIDs,
		}
		if err := connection.enqueue(&pb.APIFrame{
			Body: &pb.APIFrame_RoomPushBatch{
				RoomPushBatch: &pb.RoomPushBatch{RoomPushes: []*pb.RoomPush{roomPush}},
			},
		}); err != nil {
			// Drop only this gateway: a stalled peer must not cost the others,
			// and clients recover the message through the room_seq pull path.
			log.Error().Err(err).Str("room_id", roomIDStr).Msg("room push dropped")
			continue
		}
	}
	return nil
}

func parseUUID(field, raw string) (uuid.UUID, error) {
	value, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil(), fmt.Errorf("invalid %s: %w", field, err)
	}
	return value, nil
}

func (h *StreamHandler) message(input *pb.SendMessage) (store.Message, error) {
	sender, err := parseUUID("sender_id", input.GetSenderId())
	if err != nil {
		return store.Message{}, err
	}
	room, err := parseUUID("room_id", input.GetRoomId())
	if err != nil {
		return store.Message{}, err
	}
	client, err := parseUUID("client_msg_id", input.GetClientMsgId())
	if err != nil {
		return store.Message{}, err
	}
	message := store.Message{
		ClientMsgID: client,
		SenderID:    sender,
		RoomID:      room,
		MsgType:     store.MsgType(input.GetMsgType()),
		Payload:     input.GetPayload(),
		Ext:         input.GetExt(),
	}
	if !message.MsgType.Valid() {
		return store.Message{}, fmt.Errorf("invalid msg_type %d", input.GetMsgType())
	}
	if input.GetReplyToMsgId() != "" {
		reply, err := parseUUID("reply_to_msg_id", input.GetReplyToMsgId())
		if err != nil {
			return store.Message{}, err
		}
		message.ReplyToMsgID = reply
	}
	return message, nil
}

// onlineConnection returns the gateway connection carrying the user's
// session, or nil when the user is offline on this node.
func (h *StreamHandler) onlineConnection(userID string) *apiConnection {
	shard := h.getSessionShard(userID)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	return shard.sessions[userID]
}

func (h *StreamHandler) IsOnline(userID string) bool {
	return h.onlineConnection(userID) != nil
}

func (h *StreamHandler) Push(ctx context.Context, message *store.Message) error {
	return h.push(ctx, message, message.SenderID)
}

// PushExcluding pushes a message to every online room member except the given
// operator, who produced the event but is not the original sender (e.g. an
// admin recalling another user's message).
func (h *StreamHandler) PushExcluding(ctx context.Context, message *store.Message, operatorID uuid.UUID) error {
	return h.push(ctx, message, operatorID)
}

// Notification is the typed payload of one push event. Each implementation
// also names its wire event, so an event and its payload shape cannot drift
// apart; the set of notifications is closed and grep-visible via event().
type Notification interface {
	event() string
}

// RoomUpdated reports room lifecycle changes; Action discriminates "created",
// "invited", "info_updated" and "dissolved", and only the fields an action
// carries are set.
type RoomUpdated struct {
	Action    string    `json:"action"`
	RoomID    uuid.UUID `json:"room_id"`
	ChatType  string    `json:"chat_type,omitzero"`
	Name      string    `json:"name,omitzero"`
	AvatarURL string    `json:"avatar_url,omitzero"`
	Notice    string    `json:"notice,omitzero"`
	InviterID uuid.UUID `json:"inviter_id,omitzero"`
}

func (RoomUpdated) event() string { return "room_update" }

// RoomMembersChanged tells the room's online members that membership changed.
type RoomMembersChanged struct {
	RoomID uuid.UUID `json:"room_id"`
	Action string    `json:"action"`
}

func (RoomMembersChanged) event() string { return "room_members_changed" }

// ReactionUpdated carries the current reaction groups of one message.
type ReactionUpdated struct {
	RoomID    uuid.UUID             `json:"room_id"`
	MsgID     uuid.UUID             `json:"msg_id"`
	Reactions []store.ReactionGroup `json:"reactions"`
}

func (ReactionUpdated) event() string { return "reaction" }

// FriendApplication tells the target about an incoming friend request, with
// the sender's profile filled in when it resolves.
type FriendApplication struct {
	FromUserID    uuid.UUID `json:"from_user_id"`
	Greeting      string    `json:"greeting"`
	FromUsername  string    `json:"from_username,omitzero"`
	FromNickname  string    `json:"from_nickname,omitzero"`
	FromAvatarURL string    `json:"from_avatar_url,omitzero"`
}

func (FriendApplication) event() string { return "friend_application" }

// FriendAccepted tells the applicant that the request was accepted, with the
// accepter's profile filled in when it resolves.
type FriendAccepted struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username,omitzero"`
	Nickname  string    `json:"nickname,omitzero"`
	AvatarURL string    `json:"avatar_url,omitzero"`
}

func (FriendAccepted) event() string { return "friend_accepted" }

// NotifyUser delivers one notification to a user, if the user is online on
// this node. Fire-and-forget: the frame lands in the connection outbox and a
// drop is logged.
func (h *StreamHandler) NotifyUser(userID uuid.UUID, n Notification) {
	h.NotifyUsers([]uuid.UUID{userID}, n)
}

// NotifyUsers delivers one notification to every online user in the list,
// grouped per gateway connection so the payload crosses each gateway once.
func (h *StreamHandler) NotifyUsers(userIDs []uuid.UUID, n Notification) {
	payload, err := json.Marshal(n)
	if err != nil {
		log.Error().Err(err).Str("event", n.event()).Msg("marshal notification failed")
		return
	}
	targets := make(map[*apiConnection][]string)
	for _, uid := range userIDs {
		uidStr := uid.String()
		if conn := h.onlineConnection(uidStr); conn != nil {
			targets[conn] = append(targets[conn], uidStr)
		}
	}
	h.fanOut(targets, n.event(), payload)
}

// NotifyRoom delivers one notification to every online room member except
// excludeUserID.
func (h *StreamHandler) NotifyRoom(ctx context.Context, roomID, excludeUserID uuid.UUID, n Notification) {
	members, err := h.getRoomMembers(ctx, roomID)
	if err != nil {
		log.Error().Err(err).Str("room_id", roomID.String()).Msg("load room members for notification failed")
		return
	}
	userIDs := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		if excludeUserID != uuid.Nil() && member.userID == excludeUserID {
			continue
		}
		userIDs = append(userIDs, member.userID)
	}
	h.NotifyUsers(userIDs, n)
}

// fanOut enqueues one notification frame per gateway connection; a stalled
// gateway drops only its frame.
func (h *StreamHandler) fanOut(targets map[*apiConnection][]string, event string, payload []byte) {
	for conn, userIDs := range targets {
		notifications := make([]*pb.Notification, len(userIDs))
		for i, userIDStr := range userIDs {
			notifications[i] = &pb.Notification{UserId: userIDStr, Event: event, Data: payload}
		}
		if err := conn.enqueue(&pb.APIFrame{
			Body: &pb.APIFrame_NotificationBatch{
				NotificationBatch: &pb.NotificationBatch{Notifications: notifications},
			},
		}); err != nil {
			log.Error().Err(err).Str("event", event).Int("dropped", len(userIDs)).Msg("notification dropped")
		}
	}
}

var _ pb.GatewayService = (*StreamHandler)(nil)
