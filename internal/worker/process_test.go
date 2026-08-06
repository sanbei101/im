package worker

import (
	"testing"

	"github.com/google/uuid"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/db"
	"github.com/sanbei101/im/internal/mq"
)

func TestMessageToDBParamsValidatesAndCopiesServerFields(t *testing.T) {
	msgID, clientMsgID, senderID, roomID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	msgType := imv1.MessageType_MESSAGE_TYPE_TEXT
	msg := &imv1.Message{
		MsgId:       stringPtr(msgID.String()),
		ClientMsgId: stringPtr(clientMsgID.String()),
		SenderId:    stringPtr(senderID.String()),
		RoomId:      stringPtr(roomID.String()),
		MsgType:     &msgType,
		ServerTime:  int64Ptr(123),
		Payload:     []byte("payload"),
	}

	got, err := messageToDBParams(msg)
	if err != nil {
		t.Fatal(err)
	}
	if got.MsgID != msgID || got.ClientMsgID != clientMsgID || got.SenderID != senderID || got.RoomID != roomID || got.MsgType != db.MessageTypeText || got.ServerTime != 123 {
		t.Fatalf("params = %#v", got)
	}
}

func TestMessageToDBParamsRejectsUnknownType(t *testing.T) {
	msg := &imv1.Message{
		MsgId:       stringPtr(uuid.NewString()),
		ClientMsgId: stringPtr(uuid.NewString()),
		SenderId:    stringPtr(uuid.NewString()),
		RoomId:      stringPtr(uuid.NewString()),
	}
	if _, err := messageToDBParams(msg); err == nil {
		t.Fatal("messageToDBParams accepted unspecified message type")
	}
}

func stringPtr(v string) *string { return &v }
func int64Ptr(v int64) *int64    { return &v }

func TestAppendFailureTaskTargetsSender(t *testing.T) {
	senderID, clientMsgID, roomID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	tasks := appendFailureTask(nil, &mq.InboundMsgEnvelope{
		Payload: &imv1.Message{SenderId: &senderID, ClientMsgId: &clientMsgID, RoomId: &roomID},
	}, "room_access_denied", "room access denied")
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(tasks))
	}
	defer mq.ReleaseDeliveryTask(tasks[0])
	if got := tasks[0].Payload.GetTargetUserIds(); len(got) != 1 || got[0] != senderID {
		t.Fatalf("targets = %#v", got)
	}
	if tasks[0].Payload.GetClientMsgId() != clientMsgID || tasks[0].Payload.GetFailed().GetCode() != "room_access_denied" {
		t.Fatalf("task = %#v", tasks[0].Payload)
	}
}
