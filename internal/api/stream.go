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

func (h *StreamHandler) Connect(stream imv1.GatewayService_ConnectServer) error {
	ctx := context.Background()
	connection := &apiConnection{stream: stream, users: make(map[string]struct{})}
	defer h.removeConnection(connection)

	for {
		frame, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("receive gateway frame: %w", err)
		}
		switch {
		case frame.GetHello() != nil:
			if err := connection.send(
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

func (c *apiConnection) send(frame *imv1.APIFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stream.Send(frame)
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
	var messages []*store.Message
	for _, input := range batch.GetMessages() {
		if input == nil {
			continue
		}
		item := &imv1.SendResult{
			RequestId:   input.GetRequestId(),
			ClientMsgId: input.GetClientMsgId(),
			RoomId:      input.GetRoomId(),
		}
		message, err := h.message(input)
		if err == nil {
			if slot, slotErr := pkg.RoomSlot(message.RoomID, h.slots); slotErr != nil {
				err = slotErr
			} else if owner, ownerErr := pkg.NodeIndex(slot, h.slots, h.nodeCount); ownerErr != nil {
				err = ownerErr
			} else if owner != h.nodeIndex {
				err = fmt.Errorf("room is owned by api node index %d", owner)
			}
		}
		if err == nil {
			written, writeErr := h.store.WriteMessage(ctx, message)
			err = writeErr
			if err == nil {
				message = written
				messages = append(messages, &message)
			}
		}
		if err != nil {
			item.Code = 1
			item.Error = err.Error()
		} else {
			item.MsgId = message.MsgID.String()
			item.RoomSeq = message.RoomSeq
			item.ServerTime = message.ServerTime
		}
		result.Results = append(result.Results, item)
	}
	return result, messages, nil
}

func (h *StreamHandler) push(ctx context.Context, message *store.Message) error {
	msgType, err := messageTypeNumber(message.MsgType)
	if err != nil {
		return fmt.Errorf("push room message: %w", err)
	}
	members, err := h.store.Members(ctx, message.RoomID)
	if err != nil {
		return fmt.Errorf("load room members: %w", err)
	}
	pushes := make(map[*apiConnection][]*imv1.Push)
	h.mu.RLock()
	for _, member := range members {
		if member.UserID == message.SenderID {
			continue
		}
		connection := h.sessions[member.UserID.String()]
		if connection != nil {
			pushes[connection] = append(
				pushes[connection],
				&imv1.Push{
					UserId:     member.UserID.String(),
					RoomId:     message.RoomID.String(),
					RoomSeq:    message.RoomSeq,
					MsgId:      message.MsgID.String(),
					SenderId:   message.SenderID.String(),
					MsgType:    msgType,
					Payload:    message.Payload,
					ServerTime: message.ServerTime,
					Ext:        message.Ext,
				},
			)
		}
	}
	h.mu.RUnlock()
	for connection, items := range pushes {
		if err := connection.send(
			&imv1.APIFrame{Body: &imv1.APIFrame_PushBatch{PushBatch: &imv1.PushBatch{Pushes: items}}},
		); err != nil {
			return fmt.Errorf("push room message: %w", err)
		}
	}
	return nil
}

func messageTypeNumber(value string) (int32, error) {
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid message type %q: %w", value, err)
	}
	return int32(number), nil
}

func (h *StreamHandler) message(input *imv1.SendMessage) (store.Message, error) {
	sender, err := uuid.Parse(input.GetSenderId())
	if err != nil {
		return store.Message{}, fmt.Errorf("invalid sender_id: %w", err)
	}
	room, err := uuid.Parse(input.GetRoomId())
	if err != nil {
		return store.Message{}, fmt.Errorf("invalid room_id: %w", err)
	}
	client, err := uuid.Parse(input.GetClientMsgId())
	if err != nil {
		return store.Message{}, fmt.Errorf("invalid client_msg_id: %w", err)
	}
	message := store.Message{
		ClientMsgID: client,
		SenderID:    sender,
		RoomID:      room,
		MsgType:     strconv.Itoa(int(input.GetMsgType())),
		Payload:     input.GetPayload(),
		Ext:         input.GetExt(),
	}
	if input.GetReplyToMsgId() != "" {
		message.ReplyToMsgID, err = uuid.Parse(input.GetReplyToMsgId())
		message.HasReply = err == nil
		if err != nil {
			return store.Message{}, fmt.Errorf("invalid reply_to_msg_id: %w", err)
		}
	}
	return message, nil
}

var _ imv1.GatewayService = (*StreamHandler)(nil)
