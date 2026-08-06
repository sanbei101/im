package gateway

import (
	"context"
	"testing"

	"github.com/google/uuid"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/mq"
)

type testMQ struct {
	messages []*imv1.Message
	acked    []string
}

func (m *testMQ) InitStreamGroups(context.Context) error { return nil }
func (m *testMQ) WorkerPullMessage(context.Context, int64) ([]*mq.InboundMsgEnvelope, error) {
	return nil, nil
}
func (m *testMQ) WorkerEnqueueDeliveryTask(context.Context, []*mq.DeliverTaskEnvelope) error {
	return nil
}
func (m *testMQ) WorkerAckMessage(context.Context, ...string) error { return nil }
func (m *testMQ) WorkerEnqueueDeadLetter(context.Context, *mq.InboundMsgEnvelope, string) error {
	return nil
}
func (m *testMQ) GatewayPullDeliveryTask(context.Context, int64) ([]*mq.DeliverTaskEnvelope, error) {
	return nil, nil
}
func (m *testMQ) GatewayEnqueueMessage(_ context.Context, messages []*imv1.Message) error {
	m.messages = append(m.messages, messages...)
	return nil
}
func (m *testMQ) GatewayAckMessage(_ context.Context, ids ...string) error {
	m.acked = append(m.acked, ids...)
	return nil
}

func TestUserClientSendMessageUsesFrames(t *testing.T) {
	clientMsgID := uuid.NewString()
	roomID := uuid.NewString()
	msgType := imv1.MessageType_MESSAGE_TYPE_TEXT
	frame := &imv1.ClientFrame{
		ClientMsgId: &clientMsgID,
		Payload: &imv1.ClientFrame_SendMessage{SendMessage: &imv1.SendMessageReq{
			RoomId:  &roomID,
			MsgType: &msgType,
			Payload: []byte("hello"),
		}},
	}
	payload, err := frame.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}

	broker := &testMQ{}
	client := &UserClient{
		gateway: &Gateway{MQ: broker},
		Send:    make(chan []byte, 1),
		UserID:  uuid.MustParse(uuid.NewString()),
	}
	client.handleUserMessage(context.Background(), payload)

	if len(broker.messages) != 1 {
		t.Fatalf("enqueued %d messages, want 1", len(broker.messages))
	}
	if got := broker.messages[0].GetClientMsgId(); got != clientMsgID {
		t.Fatalf("enqueued client_msg_id = %q, want %q", got, clientMsgID)
	}

	response := &imv1.ServerFrame{}
	if err := response.UnmarshalVT(<-client.Send); err != nil {
		t.Fatal(err)
	}
	if got := response.GetClientMsgId(); got != clientMsgID {
		t.Fatalf("response client_msg_id = %q, want %q", got, clientMsgID)
	}
	if response.GetAck() == nil || response.GetAck().GetMsgId() == "" || response.GetAck().GetServerTime() == 0 {
		t.Fatalf("response = %#v, want populated ACK", response)
	}
}

func TestUserClientPingUsesPongFrame(t *testing.T) {
	clientMsgID := uuid.NewString()
	frame := &imv1.ClientFrame{
		ClientMsgId: &clientMsgID,
		Payload:     &imv1.ClientFrame_Ping{Ping: &imv1.Ping{}},
	}
	payload, err := frame.MarshalVT()
	if err != nil {
		t.Fatal(err)
	}

	client := &UserClient{Send: make(chan []byte, 1), UserID: uuid.MustParse(uuid.NewString())}
	client.handleUserMessage(context.Background(), payload)

	response := &imv1.ServerFrame{}
	if err := response.UnmarshalVT(<-client.Send); err != nil {
		t.Fatal(err)
	}
	if response.GetPong() == nil || response.GetClientMsgId() != clientMsgID {
		t.Fatalf("response = %#v, want correlated Pong", response)
	}
}

func TestGatewayDeliversWorkerFailureFrame(t *testing.T) {
	broker := &testMQ{}
	gateway := &Gateway{MQ: broker, UserSessionManager: NewSessionManager()}
	userID := uuid.NewString()
	client := &UserClient{Send: make(chan []byte, 1)}
	session := gateway.UserSessionManager.LoadOrCreate(userID, NewUserSession)
	session.Add(client)

	clientMsgID, streamID := uuid.NewString(), "1-0"
	code, message := "room_access_denied", "room access denied"
	task := mq.AcquireDeliveryTask()
	task.StreamID = streamID
	task.Payload.TargetUserIds = []string{userID}
	task.Payload.ClientMsgId = &clientMsgID
	task.Payload.Failed = &imv1.MessageFailed{Code: &code, Message: &message}
	gateway.processTasks(context.Background(), []*mq.DeliverTaskEnvelope{task})

	frame := &imv1.ServerFrame{}
	if err := frame.UnmarshalVT(<-client.Send); err != nil {
		t.Fatal(err)
	}
	if frame.GetClientMsgId() != clientMsgID || frame.GetFailed() == nil || frame.GetFailed().GetCode() != code {
		t.Fatalf("frame = %#v", frame)
	}
	if len(broker.acked) != 1 || broker.acked[0] != streamID {
		t.Fatalf("acked = %#v", broker.acked)
	}
}
