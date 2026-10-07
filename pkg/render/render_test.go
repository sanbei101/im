package render

import (
	"bytes"
	"encoding/json/jsontext"
	"testing"
)

func BenchmarkFrameWriterPushFrame(b *testing.B) {
	fw := NewFrameWriter()
	frame := PushFrame{
		Type:        "message",
		MsgID:       "01999999-9999-7999-9999-999999999999",
		ClientMsgID: "01999999-9999-7999-9999-999999999998",
		SenderID:    "01999999-9999-7999-9999-999999999997",
		RoomID:      "01999999-9999-7999-9999-999999999996",
		RoomSeq:     1024,
		ServerTime:  1728000000000000,
		MsgType:     "text",
		Payload:     jsontext.Value(`{"text":"benchmark payload text"}`),
	}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		data, err := fw.EncodeFrame(frame)
		if err != nil || len(data) == 0 {
			b.Fatalf("encode failed: %v", err)
		}
	}
}

func BenchmarkFrameReaderClientFrame(b *testing.B) {
	rawJSON := []byte(
		`{"type":"message","request_id":"req-123","client_msg_id":"01999999-9999-7999-9999-999999999998","room_id":"01999999-9999-7999-9999-999999999996","msg_type":"text","payload":{"text":"benchmark test"}}` + "\n",
	)

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		reader := bytes.NewReader(rawJSON)
		fr := NewFrameReader(reader)
		var frame ClientFrame
		if err := fr.ReadFrame(&frame); err != nil {
			b.Fatalf("read frame failed: %v", err)
		}
	}
}
