package store

import (
	"encoding/json/jsontext"
	"testing"
	"uuid"
)

// tailOf appends messages 1..count into a fresh cache via the commit path.
func tailOf(count int, payload string) *messageCache {
	cache := newMessageCache()
	for i := range count {
		cache.append(&Message{
			RoomID:  uuid.UUID{1},
			RoomSeq: uint64(i + 1),
			Payload: jsontext.Value(payload),
		})
	}
	return cache
}

func TestMessageCache(t *testing.T) {
	room := uuid.UUID{1}

	t.Run("newest page served from memory", func(t *testing.T) {
		cache := newMessageCache()
		for i := range 10 {
			cache.append(&Message{RoomID: room, RoomSeq: uint64(i + 1), Payload: jsontext.Value("m")})
		}
		messages, hasMore, ok := cache.page(room, 0, 4)
		if !ok || !hasMore || len(messages) != 4 {
			t.Fatalf("page: ok=%v hasMore=%v len=%d", ok, hasMore, len(messages))
		}
		if messages[0].RoomSeq != 10 || messages[3].RoomSeq != 7 {
			t.Fatalf("newest page must descend from the top: %+v", messages)
		}
	})

	t.Run("uncached window falls back", func(t *testing.T) {
		cache := newMessageCache()
		// startSeq == 2: a window reaching below the tail start must fall back
		// to Pebble because seq 1 is not cached.
		for i := range 5 {
			cache.append(&Message{RoomID: room, RoomSeq: uint64(i + 2), Payload: jsontext.Value("m")})
		}
		if _, _, ok := cache.page(room, 0, 10); ok {
			t.Fatal("window below tail start must not be served")
		}
		if _, _, ok := cache.page(room, 0, 4); !ok {
			t.Fatal("window inside the tail must be served")
		}
		if _, _, ok := cache.page(uuid.UUID{2}, 0, 4); ok {
			t.Fatal("unknown room must fall back")
		}
	})

	t.Run("dedup replay of older seq ignored", func(t *testing.T) {
		cache := newMessageCache()
		for seq := uint64(1); seq <= 3; seq++ {
			cache.append(&Message{RoomID: room, RoomSeq: seq, Payload: jsontext.Value("m")})
		}
		// A replayed commit of seq 2 must not re-enter the ring: the tail stays
		// a dense ascending run and the stale payload never shows up.
		cache.append(&Message{RoomID: room, RoomSeq: 2, Payload: jsontext.Value("stale")})
		messages, _, ok := cache.page(room, 0, 10)
		if !ok || len(messages) != 3 {
			t.Fatalf("stale append broke the tail: ok=%v messages=%+v", ok, messages)
		}
		for _, message := range messages {
			if string(message.Payload) != "m" {
				t.Fatalf("stale payload entered the tail: %+v", message)
			}
		}
	})

	t.Run("byte bound trims oldest", func(t *testing.T) {
		// 40 payloads of 4KB exceed tailMaxBytes (64KB): the tail must keep
		// trimming from the front while staying a dense ascending run.
		big := make([]byte, 4096)
		cache := newMessageCache()
		for i := range 40 {
			cache.append(&Message{RoomID: room, RoomSeq: uint64(i + 1), Payload: jsontext.Value(big)})
		}
		tail, ok := cache.cache.Get(room)
		if !ok {
			t.Fatal("room tail missing from cache")
		}
		if tail.bytes > tailMaxBytes {
			t.Fatalf("tail holds %d bytes, bound is %d", tail.bytes, tailMaxBytes)
		}
		// The trimmed tail no longer reaches seq 1, so only a window inside
		// [startSeq, endSeq] is servable: limit must stay below the tail count.
		messages, hasMore, served := cache.page(room, 0, tail.count-1)
		if !served || !hasMore || len(messages) != tail.count-1 {
			t.Fatalf("trimmed tail page: ok=%v hasMore=%v len=%d count=%d", served, hasMore, len(messages), tail.count)
		}
		if messages[0].RoomSeq != 40 {
			t.Fatalf("trimmed tail must end at the newest seq: top=%d", messages[0].RoomSeq)
		}
	})

	t.Run("ring wraps after capacity", func(t *testing.T) {
		const count = tailCapacity + 30
		cache := tailOf(count, "wrap")
		tail, cached := cache.cache.Get(room)
		if !cached {
			t.Fatal("room tail missing from cache")
		}
		if tail.count != tailCapacity {
			t.Fatalf("tail holds %d messages, capacity is %d", tail.count, tailCapacity)
		}
		messages, hasMore, ok := cache.page(room, 0, tail.count-1)
		if !ok || !hasMore || len(messages) != tailCapacity-1 {
			t.Fatalf("wrapped tail page: ok=%v hasMore=%v len=%d", ok, hasMore, len(messages))
		}
		if messages[0].RoomSeq != count {
			t.Fatalf("wrapped tail must end at the newest seq: top=%d", messages[0].RoomSeq)
		}
	})

	t.Run("replace updates recalled message in place", func(t *testing.T) {
		cache := newMessageCache()
		cache.append(&Message{RoomID: room, RoomSeq: 1, MsgType: MsgTypeText, Payload: jsontext.Value("orig")})
		cache.replace(room, &Message{RoomID: room, RoomSeq: 1, MsgType: MsgTypeRecall, Payload: jsontext.Value("gone")})
		messages, _, ok := cache.page(room, 0, 10)
		if !ok || messages[0].MsgType != MsgTypeRecall {
			t.Fatalf("replace not visible: ok=%v messages=%+v", ok, messages)
		}
		cache.replace(room, &Message{RoomID: room, RoomSeq: 99, MsgType: MsgTypeRecall})
		messages, _, _ = cache.page(room, 0, 10)
		if len(messages) != 1 {
			t.Fatalf("replace of unknown seq must be a no-op, tail=%+v", messages)
		}
	})

	t.Run("backfill only from a page reaching the newest seq", func(t *testing.T) {
		cache := newMessageCache()
		lastSeq := uint64(6)
		old := []Message{
			{RoomID: room, RoomSeq: 1, Payload: jsontext.Value("m")},
			{RoomID: room, RoomSeq: 2, Payload: jsontext.Value("m")},
		}
		cache.put(room, old, lastSeq)
		if _, _, ok := cache.page(room, 0, 4); ok {
			t.Fatal("page not reaching lastSeq must not install a tail")
		}

		full := []Message{
			{RoomID: room, RoomSeq: 3, Payload: jsontext.Value("m")},
			{RoomID: room, RoomSeq: 4, Payload: jsontext.Value("m")},
			{RoomID: room, RoomSeq: 5, Payload: jsontext.Value("m")},
			{RoomID: room, RoomSeq: 6, Payload: jsontext.Value("m")},
		}
		cache.put(room, full, lastSeq)
		messages, hasMore, ok := cache.page(room, 0, 2)
		if !ok || !hasMore || len(messages) != 2 || messages[0].RoomSeq != 6 {
			t.Fatalf("backfilled tail page: ok=%v hasMore=%v messages=%+v", ok, hasMore, messages)
		}

		// A read of seq 2 arrives after the tail reached 6: installing it would
		// masquerade as the newest page, so it must be ignored.
		cache.put(room, old, 2)
		messages, _, _ = cache.page(room, 0, 2)
		if messages[0].RoomSeq != 6 {
			t.Fatalf("stale backfill replaced the tail: top=%d", messages[0].RoomSeq)
		}
	})
}
