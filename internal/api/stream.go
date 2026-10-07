package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	"github.com/sanbei101/im/pkg"
	"github.com/sanbei101/im/proto/pb"
)

const (
	sessionShardCount     = 64
	memberCacheShardCount = 64
)

type sessionShard struct {
	mu       sync.RWMutex
	sessions map[string]*apiConnection
}

type memberCacheShard struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]*roomMemberCacheEntry
}

type StreamHandler struct {
	store     *store.Store
	nodeID    string
	slots     int
	nodeIndex int
	nodeCount int

	sessionShards [sessionShardCount]sessionShard
	memberShards  [memberCacheShardCount]memberCacheShard
}

type roomMemberCacheEntry struct {
	members []cachedMember
	expires time.Time
}

type cachedMember struct {
	userID    uuid.UUID
	userIDStr string
}

type apiConnection struct {
	stream pb.GatewayService_ConnectServer
	mu     sync.Mutex
	users  map[string]struct{}
}

func (h *StreamHandler) getSessionShard(userID string) *sessionShard {
	var hVal uint32
	for i := 0; i < len(userID); i++ {
		hVal = hVal*31 + uint32(userID[i])
	}
	return &h.sessionShards[hVal%sessionShardCount]
}

func (h *StreamHandler) getMemberShard(roomID uuid.UUID) *memberCacheShard {
	return &h.memberShards[roomID[15]%memberCacheShardCount]
}

func NewStreamHandler(data *store.Store, nodeID string, slots, nodeIndex, nodeCount int) *StreamHandler {
	handler := &StreamHandler{
		store: data, nodeID: nodeID, slots: slots, nodeIndex: nodeIndex, nodeCount: nodeCount,
	}
	for i := range handler.sessionShards {
		handler.sessionShards[i].sessions = make(map[string]*apiConnection)
	}
	for i := range handler.memberShards {
		handler.memberShards[i].entries = make(map[uuid.UUID]*roomMemberCacheEntry)
	}
	return handler
}

func (h *StreamHandler) InvalidateRoomMembers(roomID uuid.UUID) {
	shard := h.getMemberShard(roomID)
	shard.mu.Lock()
	delete(shard.entries, roomID)
	shard.mu.Unlock()
}

func (h *StreamHandler) getRoomMembers(ctx context.Context, roomID uuid.UUID) ([]cachedMember, error) {
	now := time.Now()
	shard := h.getMemberShard(roomID)
	shard.mu.RLock()
	entry, ok := shard.entries[roomID]
	if ok && now.Before(entry.expires) {
		members := entry.members
		shard.mu.RUnlock()
		return members, nil
	}
	shard.mu.RUnlock()

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
	shard.mu.Lock()
	shard.entries[roomID] = &roomMemberCacheEntry{
		members: cached,
		expires: now.Add(30 * time.Second),
	}
	shard.mu.Unlock()
	return cached, nil
}

func (h *StreamHandler) Connect(ctx context.Context, stream pb.GatewayService_ConnectServer) error {
	connection := &apiConnection{stream: stream, users: make(map[string]struct{})}
	defer h.removeConnection(connection)

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
			if err := connection.send(
				ctx,
				&pb.APIFrame{
					Body: &pb.APIFrame_HelloAck{
						HelloAck: &pb.HelloAck{NodeId: h.nodeID, TopologyVersion: strconv.Itoa(h.slots)},
					},
				},
			); err != nil {
				return fmt.Errorf("send hello ack: %w", err)
			}
		case frame.GetSessionBatch() != nil:
			h.updateSessions(connection, frame.GetSessionBatch())
		case frame.GetSendBatch() != nil:
			result, pushes, err := h.writeBatch(ctx, frame.GetSendBatch())
			if err != nil {
				return err
			}
			if err := connection.send(
				ctx,
				&pb.APIFrame{Body: &pb.APIFrame_SendResultBatch{SendResultBatch: result}},
			); err != nil {
				return fmt.Errorf("send result batch: %w", err)
			}
			for _, push := range pushes {
				if err := h.push(ctx, push); err != nil {
					return err
				}
			}
		}
	}
}

func (c *apiConnection) send(ctx context.Context, frame *pb.APIFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stream.Send(ctx, frame)
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

func (h *StreamHandler) push(ctx context.Context, message *store.Message) error {
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

	// Map connection -> pushes so every gateway stream receives one batch per room fan-out.
	pushes := make(map[*apiConnection][]*pb.Push)
	for _, member := range members {
		if member.userID == message.SenderID {
			continue
		}
		shard := h.getSessionShard(member.userIDStr)
		shard.mu.RLock()
		connection := shard.sessions[member.userIDStr]
		shard.mu.RUnlock()
		if connection != nil {
			pushes[connection] = append(pushes[connection], &pb.Push{
				UserId:       member.userIDStr,
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
			})
		}
	}
	for connection, items := range pushes {
		if err := connection.send(ctx,
			&pb.APIFrame{Body: &pb.APIFrame_PushBatch{PushBatch: &pb.PushBatch{Pushes: items}}},
		); err != nil {
			return fmt.Errorf("push room message: %w", err)
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

func (h *StreamHandler) IsOnline(userID string) bool {
	shard := h.getSessionShard(userID)
	shard.mu.RLock()
	_, ok := shard.sessions[userID]
	shard.mu.RUnlock()
	return ok
}

func (h *StreamHandler) Push(ctx context.Context, message *store.Message) error {
	return h.push(ctx, message)
}

var _ pb.GatewayService = (*StreamHandler)(nil)
