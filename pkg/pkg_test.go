package pkg

import (
	"strconv"
	"testing"
	"uuid"
)

func TestShard(t *testing.T) {
	t.Run("slot is deterministic and in range", func(t *testing.T) {
		room := uuid.NewV7()
		first, err := RoomSlot(room, 1024)
		if err != nil {
			t.Fatal(err)
		}
		for range 100 {
			again, err := RoomSlot(room, 1024)
			if err != nil || again != first {
				t.Fatalf("slot not deterministic: first=%d again=%d err=%v", first, again, err)
			}
		}
	})

	t.Run("slots spread rooms across the ring", func(t *testing.T) {
		const slots = 1024
		seen := make(map[uint16]bool)
		for range 5000 {
			slot, err := RoomSlot(uuid.NewV7(), slots)
			if err != nil || slot >= slots {
				t.Fatalf("slot out of range: slot=%d err=%v", slot, err)
			}
			seen[slot] = true
		}
		if len(seen) < slots*9/10 {
			t.Fatalf("rooms covered only %d of %d slots", len(seen), slots)
		}
	})

	t.Run("invalid slot counts rejected", func(t *testing.T) {
		for _, slots := range []int{0, -1, 1<<16 + 1} {
			if _, err := RoomSlot(uuid.NewV7(), slots); err == nil {
				t.Fatalf("expected error for slots=%d", slots)
			}
		}
	})

	t.Run("node index maps slot to node", func(t *testing.T) {
		const slots, nodes = 1024, 8
		last := -1
		for slot := range slots {
			index, err := NodeIndex(uint16(slot), slots, nodes)
			if err != nil || index < 0 || index >= nodes {
				t.Fatalf("node index: slot=%d index=%d err=%v", slot, index, err)
			}
			if index < last {
				t.Fatalf("node index must not decrease: slot=%d index=%d last=%d", slot, index, last)
			}
			last = index
		}
		if last != nodes-1 {
			t.Fatalf("highest slot maps to node %d, want %d", last, nodes-1)
		}
	})

	t.Run("invalid topology rejected", func(t *testing.T) {
		for _, c := range []struct{ slot, slots, nodes int }{
			{0, 0, 1}, {0, 4, 0}, {4, 4, 2},
		} {
			if _, err := NodeIndex(uint16(c.slot), c.slots, c.nodes); err == nil {
				t.Fatalf("expected error for slot=%d slots=%d nodes=%d", c.slot, c.slots, c.nodes)
			}
		}
	})

	t.Run("string shard is deterministic and in range", func(t *testing.T) {
		const shards = 64
		if got := StringShard("request-1", shards); got != StringShard("request-1", shards) {
			t.Fatal("StringShard not deterministic")
		}
		seen := make(map[int]bool)
		for i := range 5000 {
			shard := StringShard("request-"+strconv.Itoa(i), shards)
			if shard < 0 || shard >= shards {
				t.Fatalf("shard out of range: %d", shard)
			}
			seen[shard] = true
		}
		if len(seen) < shards*9/10 {
			t.Fatalf("keys covered only %d of %d shards", len(seen), shards)
		}
	})
}

func TestSingleRoomID(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	first := SingleRoomID(a, b)

	t.Run("deterministic and order independent", func(t *testing.T) {
		if again := SingleRoomID(a, b); again != first {
			t.Fatalf("single room id is not deterministic: %s != %s", first, again)
		}
		if reversed := SingleRoomID(b, a); reversed != first {
			t.Fatalf("single room id depends on user order: %s != %s", reversed, first)
		}
		if first == uuid.Nil() {
			t.Fatal("single room id is nil")
		}
		if first == SingleRoomID(a, uuid.NewV7()) {
			t.Fatal("different user pairs produced the same room id")
		}
	})

	t.Run("matches chat hash", func(t *testing.T) {
		hash := SingleChatHash(a, b)
		if hash != SingleChatHash(b, a) {
			t.Fatal("single chat hash depends on user order")
		}
		var derived uuid.UUID
		copy(derived[:], hash[:16])
		derived[6] = (derived[6] & 0x0f) | 0x80
		derived[8] = (derived[8] & 0x3f) | 0x80
		if derived != first {
			t.Fatalf("room id does not match the chat hash: %s != %s", first, derived)
		}
	})
}

var (
	benchSlot uint16
	benchNode int
)

func BenchmarkShard(b *testing.B) {
	room := uuid.NewV7()
	b.Run("room-slot", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			slot, err := RoomSlot(room, 1024)
			if err != nil {
				b.Fatal(err)
			}
			benchSlot = slot
		}
	})
	b.Run("node-index", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()
		for b.Loop() {
			index, err := NodeIndex(512, 1024, 8)
			if err != nil {
				b.Fatal(err)
			}
			benchNode = index
		}
	})
}
