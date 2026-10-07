package store

import (
	"encoding/json/jsontext"
	"testing"
	"time"
	"uuid"
)

func benchRoomRecord(b *testing.B) []byte {
	b.Helper()
	room := Room{
		RoomID: uuid.NewV7(), ChatType: ChatTypeGroup,
		Name: "一个长度可观的房间名称", AvatarURL: "room://0197a2b3-c4d5-e6f7-a8b9-c0d1e2f3a4b5",
		Notice: "群公告:请勿发送与主题无关的内容", LastSeq: 42,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
		LastMsg: &Message{
			MsgID: uuid.NewV7(), ClientMsgID: uuid.NewV7(), SenderID: uuid.NewV7(),
			RoomID: uuid.UUID{}, RoomSeq: 42, ServerTime: time.Now().UnixMicro(),
			MsgType: MsgTypeText, Payload: jsontext.Value(`{"text":"hello world"}`),
		},
	}
	return encodeRoom(room)
}

func BenchmarkRoomDecode(b *testing.B) {
	data := benchRoomRecord(b)
	b.Run("full", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := decodeRoom(data); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("head", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if _, err := decodeRoomHead(data); err != nil {
				b.Fatal(err)
			}
		}
	})
}
