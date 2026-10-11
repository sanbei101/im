package render

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"
)

func BenchmarkRender(b *testing.B) {
	writer := NewFrameWriter()
	push := PushFrame{
		Type: "message", MsgID: "01999999-9999-7999-9999-999999999999",
		ClientMsgID: "01999999-9999-7999-9999-999999999998", SenderID: "01999999-9999-7999-9999-999999999997",
		RoomID: "01999999-9999-7999-9999-999999999996", RoomSeq: 1024, ServerTime: 1728000000000000,
		MsgType: "text", Payload: jsontext.Value(`{"text":"benchmark payload text"}`),
	}
	raw := []byte(
		`{"type":"message","request_id":"req-123","client_msg_id":"c1","room_id":"r1","msg_type":"text","payload":{"text":"benchmark test"}}` + "\n",
	)

	b.Run("encode-push-frame", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			data, err := writer.EncodeFrame(push)
			if err != nil || len(data) == 0 {
				b.Fatalf("encode failed: %v", err)
			}
		}
	})
	b.Run("read-client-frame", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := NewFrameReader(bytes.NewReader(raw)).ReadFrame[ClientFrame](); err != nil {
				b.Fatalf("read frame failed: %v", err)
			}
		}
	})
	b.Run("read-client-frame-reused", func(b *testing.B) {
		reader := NewFrameReader(nil)
		br := bytes.NewReader(raw)
		b.ReportAllocs()
		for b.Loop() {
			br.Reset(raw)
			reader.Reset(br)
			if _, err := reader.ReadFrame[ClientFrame](); err != nil {
				b.Fatalf("read frame failed: %v", err)
			}
		}
	})
}
