package store

import (
	"context"
	"sync"
	"uuid"

	"github.com/phuslu/lru"
)

// Tail cache bounds: at most cacheMaxRooms rooms, each holding the newest
// tailCapacity messages and at most tailMaxBytes of payload/ext content.
// Worst-case content is cacheMaxRooms*tailMaxBytes, plus the fixed ring slots
// (~18KB per cached room).
const (
	tailCapacity  = 128
	tailMaxBytes  = 64 << 10
	cacheMaxRooms = 4096
	cacheShards   = 64
)

type roomTail struct {
	mu       sync.RWMutex
	messages []Message // ring buffer of tailCapacity slots, ascending by RoomSeq
	head     int       // ring slot holding the oldest message
	count    int
	startSeq uint64 // oldest cached seq, 0 when empty
	endSeq   uint64 // newest cached seq, 0 when empty
	bytes    int
}

// messageCache keeps the newest messages of the most active rooms in memory.
// Only messages committed to Pebble enter the cache, so it never holds data
// that a successful ACK has not already confirmed as durable. A tail always
// ends at the room's newest seq, which is what makes the before=0 page
// servable from memory alone.
type messageCache struct {
	cache *lru.LRUCache[uuid.UUID, *roomTail]
}

func newMessageCache() *messageCache {
	return &messageCache{
		cache: lru.NewLRUCache(
			cacheMaxRooms,
			lru.WithShards[uuid.UUID, *roomTail](cacheShards),
		),
	}
}

func messageSize(m *Message) int {
	return len(m.Payload) + len(m.Ext) + 90
}

// load returns the room's tail, creating an empty one when absent. The loader
// cannot fail; the fallback only guards against a nil deref.
func (c *messageCache) load(room uuid.UUID) *roomTail {
	tail, err, _ := c.cache.GetOrLoad(context.Background(), room,
		func(context.Context, uuid.UUID) (*roomTail, error) { return &roomTail{}, nil })
	if err != nil {
		return &roomTail{}
	}
	return tail
}

// page serves a history page from the cached tail, mirroring the Pebble scan
// semantics: up to limit messages with seq < before (the newest page when
// before is 0) in descending order, plus hasMore. ok is false when the tail
// does not fully cover the window, in which case the caller falls back to Pebble.
func (c *messageCache) page(room uuid.UUID, before uint64, limit int) ([]Message, bool, bool) {
	tail, ok := c.cache.Get(room)
	if !ok {
		return nil, false, false
	}

	tail.mu.RLock()
	defer tail.mu.RUnlock()
	if tail.startSeq == 0 {
		return nil, false, false
	}
	hi := before
	if before == 0 {
		hi = tail.endSeq + 1
	}
	lo := uint64(1)
	if hi > uint64(limit)+1 {
		lo = hi - uint64(limit+1)
	}
	if hi-1 > tail.endSeq || (lo < tail.startSeq && tail.startSeq != 1) {
		return nil, false, false
	}

	messages := make([]Message, 0, hi-lo)
	for i := tail.find(lo); i < tail.count; i++ {
		message := &tail.messages[(tail.head+i)%tailCapacity]
		if message.RoomSeq >= hi {
			break
		}
		messages = append(messages, *message)
	}
	if uint64(len(messages)) != hi-lo {
		// Seq gap from a failed batch write: let Pebble answer.
		return nil, false, false
	}
	hasMore := uint64(len(messages)) > uint64(limit)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, true
}

// put backfills the tail from a Pebble read (ascending messages). Only a page
// that reaches the room's newest seq may install or replace a tail: anything
// older would masquerade as the newest page in before=0 reads. Pages older
// than an existing tail are ignored.
func (c *messageCache) put(room uuid.UUID, messages []Message, lastSeq uint64) {
	if len(messages) == 0 || messages[len(messages)-1].RoomSeq != lastSeq {
		return
	}
	page := messages
	if len(page) > tailCapacity {
		page = page[len(page)-tailCapacity:]
	}

	tail := c.load(room)
	tail.mu.Lock()
	defer tail.mu.Unlock()
	if tail.endSeq >= page[len(page)-1].RoomSeq {
		return
	}
	if tail.messages == nil {
		tail.messages = make([]Message, tailCapacity)
	}
	tail.head, tail.count = 0, len(page)
	tail.startSeq, tail.endSeq = page[0].RoomSeq, page[len(page)-1].RoomSeq
	tail.bytes = 0
	copy(tail.messages, page)
	for i := range page {
		tail.bytes += messageSize(&page[i])
	}
	for tail.bytes > tailMaxBytes && tail.count > 1 {
		tail.trimOldest()
	}
	tail.startSeq = tail.messages[tail.head].RoomSeq
}

// append records a committed message; called from the single writer goroutine
// after a successful batch commit. Dedup replays of older messages are ignored.
func (c *messageCache) append(message *Message) {
	tail := c.load(message.RoomID)

	tail.mu.Lock()
	defer tail.mu.Unlock()
	if message.RoomSeq <= tail.endSeq {
		return
	}
	if tail.messages == nil {
		tail.messages = make([]Message, tailCapacity)
	}
	if tail.count == tailCapacity {
		tail.bytes -= messageSize(&tail.messages[tail.head])
		tail.messages[tail.head] = *message
		tail.head = (tail.head + 1) % tailCapacity
	} else {
		tail.messages[(tail.head+tail.count)%tailCapacity] = *message
		tail.count++
	}
	tail.startSeq = tail.messages[tail.head].RoomSeq
	tail.endSeq = message.RoomSeq
	tail.bytes += messageSize(message)
	for tail.bytes > tailMaxBytes && tail.count > 1 {
		tail.trimOldest()
	}
	tail.startSeq = tail.messages[tail.head].RoomSeq
}

// replace updates a recalled message in place while its tail is still cached.
func (c *messageCache) replace(room uuid.UUID, message *Message) {
	tail, ok := c.cache.Get(room)
	if !ok {
		return
	}

	tail.mu.Lock()
	defer tail.mu.Unlock()
	index := tail.find(message.RoomSeq)
	if index == tail.count || tail.messages[(tail.head+index)%tailCapacity].RoomSeq != message.RoomSeq {
		return
	}
	slot := &tail.messages[(tail.head+index)%tailCapacity]
	delta := messageSize(message) - messageSize(slot)
	*slot = *message
	tail.bytes += delta
}

// find returns the logical index of the first message with seq >= target,
// or t.count when every cached seq is smaller. Caller must hold t.mu.
func (t *roomTail) find(target uint64) int {
	lo, hi := 0, t.count
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if t.messages[(t.head+mid)%tailCapacity].RoomSeq < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}

// trimOldest drops the oldest message from the ring, releasing its payload
// reference. Caller must hold t.mu.
func (t *roomTail) trimOldest() {
	oldest := &t.messages[t.head]
	t.bytes -= messageSize(oldest)
	*oldest = Message{}
	t.head = (t.head + 1) % tailCapacity
	t.count--
}
