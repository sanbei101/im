package pkg

import (
	"testing"
	"uuid"
)

func BenchmarkShard(b *testing.B) {
	room := uuid.NewV7()
	b.Run("room-slot", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := RoomSlot(room, 1024); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("node-index", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			if _, err := NodeIndex(512, 1024, 8); err != nil {
				b.Fatal(err)
			}
		}
	})
}
