package pkg

import (
	"sync"
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

func BenchmarkSyncMapConcurrent(b *testing.B) {
	var m sync.Map
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := uuid.NewV7()
		for pb.Next() {
			m.Store(id, 1)
			_, _ = m.Load(id)
			m.Delete(id)
		}
	})
}

func BenchmarkShardedMapConcurrent(b *testing.B) {
	const shards = 64
	type shard struct {
		mu sync.Mutex
		m  map[uuid.UUID]int
	}
	var sh [shards]shard
	for i := range sh {
		sh[i].m = make(map[uuid.UUID]int)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := uuid.NewV7()
		idx := id[15] % shards
		for pb.Next() {
			s := &sh[idx]
			s.mu.Lock()
			s.m[id] = 1
			_ = s.m[id]
			delete(s.m, id)
			s.mu.Unlock()
		}
	})
}

func BenchmarkSingleRWMutexMap(b *testing.B) {
	var mu sync.RWMutex
	m := make(map[uuid.UUID]int)

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := uuid.NewV7()
		for pb.Next() {
			mu.Lock()
			m[id] = 1
			mu.Unlock()

			mu.RLock()
			_ = m[id]
			mu.RUnlock()

			mu.Lock()
			delete(m, id)
			mu.Unlock()
		}
	})
}

func BenchmarkShardedRWMutexMap(b *testing.B) {
	const shards = 64
	type shard struct {
		mu sync.RWMutex
		m  map[uuid.UUID]int
	}
	var sh [shards]shard
	for i := range sh {
		sh[i].m = make(map[uuid.UUID]int)
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		id := uuid.NewV7()
		s := &sh[id[15]%shards]
		for pb.Next() {
			s.mu.Lock()
			s.m[id] = 1
			s.mu.Unlock()

			s.mu.RLock()
			_ = s.m[id]
			s.mu.RUnlock()

			s.mu.Lock()
			delete(s.m, id)
			s.mu.Unlock()
		}
	})
}
