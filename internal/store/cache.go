package store

import (
	"sync"
	"uuid"
)

// Tail cache bounds. Content bytes are accounted per stored message; the
// pre-allocated ring slots add a fixed ~18KB per room that is not counted,
// bounded by the room limit (cacheShardCount * cacheShardRooms).
const (
	tailCapacity    = 128
	tailMaxBytes    = 256 << 10
	cacheShardCount = 64
	cacheShardRooms = 64
	cacheShardBytes = 2 << 20
)

type roomTail struct {
	roomID uuid.UUID
	prev   *roomTail // intrusive LRU ring around the shard sentinel head
	next   *roomTail

	mu       sync.RWMutex
	messages []Message // ring buffer of tailCapacity slots, ascending by RoomSeq
	head     int       // ring slot holding the oldest message
	count    int
	startSeq uint64 // oldest cached seq, 0 when empty
	endSeq   uint64 // newest cached seq, 0 when empty
	bytes    int
}

type cacheShard struct {
	mu    sync.Mutex
	rooms map[uuid.UUID]*roomTail
	head  roomTail // sentinel: head.next is the most recently used tail
	bytes int
}

// messageCache keeps the newest messages of the most active rooms in memory.
// Only messages committed to Pebble enter the cache, so it never holds data
// that a successful ACK has not already confirmed as durable.
type messageCache struct {
	shards [cacheShardCount]cacheShard
}

func newMessageCache() *messageCache {
	cache := &messageCache{}
	for i := range cache.shards {
		shard := &cache.shards[i]
		shard.rooms = make(map[uuid.UUID]*roomTail)
		shard.head.next = &shard.head
		shard.head.prev = &shard.head
	}
	return cache
}

func (c *messageCache) shardFor(room uuid.UUID) *cacheShard {
	return &c.shards[room[15]%cacheShardCount]
}

func messageSize(m *Message) int {
	return len(m.Payload) + len(m.Ext) + 90
}

// page serves a history page from the cached tail, mirroring the Pebble scan
// semantics: up to limit messages with seq < before (the newest page when
// before is 0) in descending order, plus hasMore. ok is false when the tail
// does not fully cover the window, in which case the caller falls back to Pebble.
func (c *messageCache) page(room uuid.UUID, before uint64, limit int) ([]Message, bool, bool) {
	shard := c.shardFor(room)
	shard.mu.Lock()
	tail, ok := shard.rooms[room]
	if ok {
		shard.touch(tail)
	}
	shard.mu.Unlock()
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

// put backfills the tail from a Pebble read (ascending messages). Pages older
// than the cached tail are ignored; a page reaching past the tail replaces it.
func (c *messageCache) put(room uuid.UUID, messages []Message) {
	if len(messages) == 0 {
		return
	}
	shard := c.shardFor(room)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	tail := c.tail(shard, room)
	page := messages
	if len(page) > tailCapacity {
		page = page[len(page)-tailCapacity:]
	}

	tail.mu.Lock()
	defer tail.mu.Unlock()
	if tail.endSeq >= page[len(page)-1].RoomSeq {
		return
	}
	if tail.messages == nil {
		tail.messages = make([]Message, tailCapacity)
	}
	shard.bytes -= tail.bytes
	tail.head, tail.count = 0, len(page)
	tail.startSeq, tail.endSeq = page[0].RoomSeq, page[len(page)-1].RoomSeq
	tail.bytes = 0
	copy(tail.messages, page)
	for i := range page {
		tail.bytes += messageSize(&page[i])
	}
	for tail.bytes > tailMaxBytes && tail.count > 1 {
		tail.trimOldest(&shard.bytes)
	}
	tail.startSeq = tail.messages[tail.head].RoomSeq
	shard.bytes += tail.bytes
	c.evictUntilFits(shard)
}

// append records a committed message; called from the single writer goroutine
// after a successful batch commit. Dedup replays of older messages are ignored.
func (c *messageCache) append(message *Message) {
	shard := c.shardFor(message.RoomID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	tail := c.tail(shard, message.RoomID)

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
	size := messageSize(message)
	tail.bytes += size
	shard.bytes += size
	for tail.bytes > tailMaxBytes && tail.count > 1 {
		tail.trimOldest(&shard.bytes)
	}
	tail.startSeq = tail.messages[tail.head].RoomSeq
	for shard.bytes > cacheShardBytes && len(shard.rooms) > 1 {
		shard.evictOne()
	}
}

// replace updates a recalled message in place while its tail is still cached.
func (c *messageCache) replace(room uuid.UUID, message *Message) {
	shard := c.shardFor(room)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	tail, ok := shard.rooms[room]
	if !ok {
		return
	}
	shard.touch(tail)

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
	shard.bytes += delta
}

// tail returns the cached tail for room, creating and LRU-inserting an empty
// one when absent. Caller must hold shard.mu.
func (c *messageCache) tail(shard *cacheShard, room uuid.UUID) *roomTail {
	if tail, ok := shard.rooms[room]; ok {
		shard.touch(tail)
		return tail
	}
	for len(shard.rooms) >= cacheShardRooms && shard.head.prev != &shard.head {
		shard.evictOne()
	}
	tail := &roomTail{roomID: room}
	tail.linkAfter(&shard.head)
	shard.rooms[room] = tail
	return tail
}

func (c *messageCache) evictUntilFits(shard *cacheShard) {
	for (len(shard.rooms) > cacheShardRooms || shard.bytes > cacheShardBytes) && shard.head.prev != &shard.head {
		shard.evictOne()
	}
}

// touch moves a tail to the front of the shard LRU. Caller holds shard.mu.
func (s *cacheShard) touch(tail *roomTail) {
	tail.unlink()
	tail.linkAfter(&s.head)
}

// evictOne drops the least recently used tail. Caller holds shard.mu.
func (s *cacheShard) evictOne() bool {
	victim := s.head.prev
	if victim == &s.head {
		return false
	}
	victim.unlink()
	delete(s.rooms, victim.roomID)
	s.bytes -= victim.bytes
	return true
}

func (t *roomTail) unlink() {
	t.prev.next = t.next
	t.next.prev = t.prev
	t.prev, t.next = nil, nil
}

func (t *roomTail) linkAfter(head *roomTail) {
	t.prev = head
	t.next = head.next
	head.next.prev = t
	head.next = t
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
// reference. Caller must hold t.mu; delta is the shard byte counter.
func (t *roomTail) trimOldest(delta *int) {
	oldest := &t.messages[t.head]
	size := messageSize(oldest)
	*oldest = Message{}
	t.bytes -= size
	*delta -= size
	t.head = (t.head + 1) % tailCapacity
	t.count--
}
