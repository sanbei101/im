package render

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"testing"
)

func TestRender(t *testing.T) {
	t.Run("frame writer is reusable across frames", func(t *testing.T) {
		writer := NewFrameWriter()
		frames := []PushFrame{
			{Type: "message", MsgID: "m1", RoomSeq: 1, MsgType: "text", Payload: jsontext.Value(`{"text":"one"}`)},
			{Type: "message", MsgID: "m2", RoomSeq: 2, MsgType: "text", Payload: jsontext.Value(`{"text":"two"}`)},
			{Type: "pong"},
		}
		for i, frame := range frames {
			data, err := writer.EncodeFrame(frame)
			if err != nil {
				t.Fatalf("frame %d: %v", i, err)
			}
			if bytes.ContainsAny(data, "\n") {
				t.Fatalf("frame %d must be newline-free: %s", i, data)
			}
			var back PushFrame
			if err := json.Unmarshal(data, &back); err != nil {
				t.Fatalf("frame %d is not valid JSON: %s: %v", i, data, err)
			}
			if back.MsgID != frame.MsgID || back.RoomSeq != frame.RoomSeq {
				t.Fatalf("frame %d corrupted by encoder reuse: want %+v got %+v", i, frame, back)
			}
		}
	})

	t.Run("encoder recovers after a failed encode", func(t *testing.T) {
		writer := NewFrameWriter()
		// A channel has no JSON representation: the encode fails and the
		// shared buffer must be reset before the next frame.
		if _, err := writer.EncodeFrame(map[string]chan int{"bad": nil}); err == nil {
			t.Fatal("expected error for unsupported value")
		}
		data, err := writer.EncodeFrame(PongFrame{Type: "pong"})
		if err != nil {
			t.Fatalf("encode after failure: %v", err)
		}
		if !bytes.Contains(data, []byte(`"pong"`)) {
			t.Fatalf("frame after failure corrupted: %s", data)
		}
	})

	t.Run("frame reader streams consecutive frames", func(t *testing.T) {
		writer := NewFrameWriter()
		stream := new(bytes.Buffer)
		for _, ack := range []AckFrame{
			{Type: "ack", RequestID: "r1", ClientMsgID: "c1", RoomSeq: 1, Code: 0},
			{Type: "ack", RequestID: "r2", ClientMsgID: "c2", RoomSeq: 2, Code: 500, Error: "conflict"},
		} {
			data, err := writer.EncodeFrame(ack)
			if err != nil {
				t.Fatal(err)
			}
			stream.Write(data)
			stream.WriteByte('\n')
		}
		reader := NewFrameReader(stream)
		for i, want := range []AckFrame{
			{Type: "ack", RequestID: "r1", ClientMsgID: "c1", RoomSeq: 1, Code: 0},
			{Type: "ack", RequestID: "r2", ClientMsgID: "c2", RoomSeq: 2, Code: 500, Error: "conflict"},
		} {
			var got AckFrame
			if err := reader.ReadFrame(&got); err != nil {
				t.Fatalf("frame %d: %v", i, err)
			}
			if got != want {
				t.Fatalf("frame %d mismatch: want %+v got %+v", i, want, got)
			}
		}
		if err := reader.ReadFrame(&AckFrame{}); err == nil {
			t.Fatal("expected error at end of stream")
		}
	})

	t.Run("client frame decodes payload verbatim", func(t *testing.T) {
		raw := `{"type":"message","request_id":"req-1","client_msg_id":"c-1","room_id":"r-1","msg_type":"text","payload":{"text":"你好"},"ext":{"k":1}}`
		var frame ClientFrame
		if err := NewFrameReader(bytes.NewBufferString(raw)).ReadFrame(&frame); err != nil {
			t.Fatal(err)
		}
		if frame.Type != "message" || frame.RequestID != "req-1" || frame.MsgType != "text" {
			t.Fatalf("scalar fields wrong: %+v", frame)
		}
		if string(frame.Payload) != `{"text":"你好"}` || string(frame.Ext) != `{"k":1}` {
			t.Fatalf("raw JSON fields must survive untouched: payload=%s ext=%s", frame.Payload, frame.Ext)
		}
	})
}

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
			var frame ClientFrame
			if err := NewFrameReader(bytes.NewReader(raw)).ReadFrame(&frame); err != nil {
				b.Fatalf("read frame failed: %v", err)
			}
		}
	})
}
