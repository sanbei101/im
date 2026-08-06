package gateway

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	imv1 "github.com/sanbei101/im/gen/go/proto/im/v1"
	"github.com/sanbei101/im/internal/db"
)

// protoMsgTypeToDB maps the proto MessageType enum to the sqlc string enum
// used by the DB layer.
func protoMsgTypeToDB(t imv1.MessageType) db.MessageType {
	switch t {
	case imv1.MessageType_MESSAGE_TYPE_TEXT:
		return db.MessageTypeText
	case imv1.MessageType_MESSAGE_TYPE_IMAGE:
		return db.MessageTypeImage
	case imv1.MessageType_MESSAGE_TYPE_VIDEO:
		return db.MessageTypeVideo
	case imv1.MessageType_MESSAGE_TYPE_FILE:
		return db.MessageTypeFile
	case imv1.MessageType_MESSAGE_TYPE_SYSTEM:
		return db.MessageTypeSystem
	}
	return ""
}

// dbMsgTypeToProto maps the sqlc string enum back to the proto MessageType.
func dbMsgTypeToProto(t db.MessageType) imv1.MessageType {
	switch t {
	case db.MessageTypeText:
		return imv1.MessageType_MESSAGE_TYPE_TEXT
	case db.MessageTypeImage:
		return imv1.MessageType_MESSAGE_TYPE_IMAGE
	case db.MessageTypeVideo:
		return imv1.MessageType_MESSAGE_TYPE_VIDEO
	case db.MessageTypeFile:
		return imv1.MessageType_MESSAGE_TYPE_FILE
	case db.MessageTypeSystem:
		return imv1.MessageType_MESSAGE_TYPE_SYSTEM
	}
	return imv1.MessageType_MESSAGE_TYPE_UNSPECIFIED
}

// sendMessageReqToMessage converts an inbound client request into a
// fully populated Message ready for the MQ inbound stream. Server-side
// fields (msg_id, sender_id, server_time) are filled here.
func sendMessageReqToMessage(
	req *imv1.SendMessageReq,
	clientMsgID string,
	senderID, msgID uuid.UUID,
	serverTime int64,
) (*imv1.Message, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	clientMsgUUID, err := uuid.Parse(clientMsgID)
	if err != nil {
		return nil, fmt.Errorf("invalid client_msg_id: %w", err)
	}
	roomID, err := uuid.Parse(req.GetRoomId())
	if err != nil {
		return nil, fmt.Errorf("invalid room_id: %w", err)
	}
	msgType := protoMsgTypeToDB(req.GetMsgType())
	if msgType == "" {
		return nil, fmt.Errorf("invalid msg_type: %v", req.GetMsgType())
	}

	// protobuf-go-lite represents proto3 scalar fields as *T; bind to local
	// variables so the proto takes ownership of the values.
	msgIDStr := msgID.String()
	clientMsgIDStr := clientMsgUUID.String()
	senderIDStr := senderID.String()
	roomIDStr := roomID.String()
	protoType := dbMsgTypeToProto(msgType)
	serverTimeVal := serverTime

	msg := &imv1.Message{
		MsgId:       &msgIDStr,
		ClientMsgId: &clientMsgIDStr,
		SenderId:    &senderIDStr,
		RoomId:      &roomIDStr,
		ServerTime:  &serverTimeVal,
		MsgType:     &protoType,
		Payload:     req.GetPayload(),
		Ext:         req.GetExt(),
	}
	if s := req.GetReplyToMsgId(); s != "" {
		if _, err := uuid.Parse(s); err != nil {
			return nil, fmt.Errorf("invalid reply_to_msg_id: %w", err)
		}
		reply := s
		msg.ReplyToMsgId = &reply
	}
	return msg, nil
}
