package store

import (
	"encoding/json/jsontext"
	"testing"
	"uuid"
)

func BenchmarkMessageCache(b *testing.B) {
	b.Run("append", func(b *testing.B) {
		cache := newMessageCache()
		room := uuid.NewV7()
		message := Message{RoomID: room, RoomSeq: 1, Payload: jsontext.Value(`{"text":"cache benchmark"}`)}

		b.ReportAllocs()
		for i := 0; b.Loop(); i++ {
			message.RoomSeq = uint64(i + 1)
			cache.append(&message)
		}
	})
	b.Run("page", func(b *testing.B) {
		cache := newMessageCache()
		room := uuid.NewV7()
		for i := range 128 {
			cache.append(
				&Message{RoomID: room, RoomSeq: uint64(i + 1), Payload: jsontext.Value(`{"text":"cache benchmark"}`)},
			)
		}

		b.ReportAllocs()
		for b.Loop() {
			messages, _, ok := cache.page(room, 0, 20)
			if !ok || len(messages) != 20 {
				b.Fatalf("page: ok=%v len=%d", ok, len(messages))
			}
		}
	})
}
