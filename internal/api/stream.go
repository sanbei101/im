package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"uuid"

	"github.com/sanbei101/im/internal/store"
	imv1 "github.com/sanbei101/im/kitex_gen/im/v1"
	"github.com/sanbei101/im/pkg"
)

type StreamHandler struct {
	store     *store.Store
	nodeID    string
	slots     int
	nodeIndex int
	nodeCount int

	mu       sync.RWMutex
	sessions map[string]*apiConnection
}

type apiConnection struct {
	stream imv1.GatewayService_ConnectServer
	mu     sync.Mutex
	users  map[string]struct{}
}

func NewStreamHandler(data *store.Store, nodeID string, slots, nodeIndex, nodeCount int) *StreamHandler {
	return &StreamHandler{
		store: data, nodeID: nodeID, slots: slots, nodeIndex: nodeIndex, nodeCount: nodeCount,
		sessions: make(map[string]*apiConnection),
	}
}

func (h *StreamHandler) Connect(ctx context.Context, stream imv1.GatewayService_ConnectServer) error {
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
				&imv1.APIFrame{
					Body: &imv1.APIFrame_HelloAck{
						HelloAck: &imv1.HelloAck{NodeId: h.nodeID, TopologyVersion: strconv.Itoa(h.slots)},
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
				&imv1.APIFrame{Body: &imv1.APIFrame_SendResultBatch{SendResultBatch: result}},
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

func (c *apiConnection) send(ctx context.Context, frame *imv1.APIFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stream.Send(ctx, frame)
}

func (h *StreamHandler) updateSessions(connection *apiConnection, batch *imv1.SessionBatch) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, session := range batch.GetSessions() {
		if session == nil {
			continue
		}
		if session.GetOnline() {
			h.sessions[session.GetUserId()] = connection
			connection.users[session.GetUserId()] = struct{}{}
		} else if current := h.sessions[session.GetUserId()]; current == connection {
			delete(h.sessions, session.GetUserId())
			delete(connection.users, session.GetUserId())
		}
	}
}

func (h *StreamHandler) removeConnection(connection *apiConnection) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for userID := range connection.users {
		if current := h.sessions[userID]; current == connection {
			delete(h.sessions, userID)
		}
	}
}

func (h *StreamHandler) writeBatch(
	ctx context.Context,
	batch *imv1.SendBatch,
) (*imv1.SendResultBatch, []*store.Message, error) {
	result := &imv1.SendResultBatch{BatchId: batch.GetBatchId()}
	inputs := batch.GetMessages()
	messages := make([]store.Message, 0, len(inputs))
	resultIndexes := make([]int, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			continue
		}
		item := &imv1.SendResult{
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
	members, err := h.store.Members(ctx, message.RoomID)
	if err != nil {
		return fmt.Errorf("load room members: %w", err)
	}
	// Map connection -> pushes so every gateway stream receives one batch per room fan-out.
	pushes := make(map[*apiConnection][]*imv1.Push)
	h.mu.RLock()
	for _, member := range members {
		if member.UserID == message.SenderID {
			continue
		}
		var replyToMsgID string
		if message.ReplyToMsgID != uuid.Nil() {
			replyToMsgID = message.ReplyToMsgID.String()
		}
		if connection := h.sessions[member.UserID.String()]; connection != nil {
			pushes[connection] = append(pushes[connection], &imv1.Push{
				UserId:       member.UserID.String(),
				RoomId:       message.RoomID.String(),
				RoomSeq:      message.RoomSeq,
				MsgId:        message.MsgID.String(),
				SenderId:     message.SenderID.String(),
				MsgType:      int32(message.MsgType),
				Payload:      message.Payload,
				ServerTime:   message.ServerTime,
				Ext:          message.Ext,
				ClientMsgId:  message.ClientMsgID.String(),
				ReplyToMsgId: replyToMsgID,
			})
		}
	}
	h.mu.RUnlock()
	for connection, items := range pushes {
		if err := connection.send(ctx,
			&imv1.APIFrame{Body: &imv1.APIFrame_PushBatch{PushBatch: &imv1.PushBatch{Pushes: items}}},
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

func (h *StreamHandler) message(input *imv1.SendMessage) (store.Message, error) {
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

var _ imv1.GatewayService = (*StreamHandler)(nil)
