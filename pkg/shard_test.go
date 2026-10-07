package pkg

import (
	"testing"
	"uuid"
)

func BenchmarkRoomSlot(b *testing.B) {
	roomID := uuid.NewV7()
	slots := 1024

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		slot, err := RoomSlot(roomID, slots)
		if err != nil || slot >= 1024 {
			b.Fatalf("room slot: %v", err)
		}
	}
}

func BenchmarkNodeIndex(b *testing.B) {
	slots := 1024
	nodes := 8

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		idx, err := NodeIndex(512, slots, nodes)
		if err != nil || idx < 0 {
			b.Fatalf("node index: %v", err)
		}
	}
}
