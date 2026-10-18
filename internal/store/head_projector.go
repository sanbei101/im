package store

import (
	"context"
	"sync"
	"time"
	"uuid"

	"github.com/phuslu/log"
)

const (
	headProjectorMaxRooms = 4096
	headProjectorMaxBytes = 16 << 20
	headProjectorBatch    = 256
	headProjectorTimeout  = 2 * time.Second
	headProjectorRetry    = 250 * time.Millisecond
)

// roomHeadProjector decouples the PostgreSQL conversation projection from the
// message ACK path. Updates are coalesced per room and bounded by both room
// count and payload bytes, so a slow database can never grow the queue without
// bound or block Pebble commits.
type roomHeadProjector struct {
	db *DB

	mu           sync.Mutex
	pending      map[uuid.UUID]RoomHeadUpdate
	pendingBytes int
	lastWarn     time.Time

	signal    chan struct{}
	stop      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
}

func newRoomHeadProjector(db *DB) *roomHeadProjector {
	return &roomHeadProjector{
		db:      db,
		pending: make(map[uuid.UUID]RoomHeadUpdate),
		signal:  make(chan struct{}, 1),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (p *roomHeadProjector) start() {
	go p.run()
}

func (p *roomHeadProjector) enqueue(update RoomHeadUpdate) {
	select {
	case <-p.stop:
		return
	default:
	}

	p.mu.Lock()
	old, exists := p.pending[update.RoomID]
	if exists && old.Seq > update.Seq {
		p.mu.Unlock()
		return
	}
	if !exists && len(p.pending) >= headProjectorMaxRooms {
		p.warnDropLocked(update.RoomID, "room head projection queue full")
		p.mu.Unlock()
		return
	}
	size := roomHeadUpdateBytes(update)
	if exists {
		p.pendingBytes -= roomHeadUpdateBytes(old)
	}
	if p.pendingBytes+size > headProjectorMaxBytes {
		if exists {
			p.pendingBytes += roomHeadUpdateBytes(old)
		}
		p.warnDropLocked(update.RoomID, "room head projection byte budget full")
		p.mu.Unlock()
		return
	}
	p.pending[update.RoomID] = update
	p.pendingBytes += size
	p.mu.Unlock()
	p.wake()
}

func (p *roomHeadProjector) warnDropLocked(roomID uuid.UUID, message string) {
	if time.Since(p.lastWarn) < time.Second {
		return
	}
	p.lastWarn = time.Now()
	log.Warn().Str("room_id", roomID.String()).Msg(message)
}

func roomHeadUpdateBytes(update RoomHeadUpdate) int {
	size := 128
	if update.Message != nil {
		size += len(update.Message.Payload) + len(update.Message.Ext)
	}
	return size
}

func (p *roomHeadProjector) wake() {
	select {
	case p.signal <- struct{}{}:
	default:
	}
}

func (p *roomHeadProjector) takeBatch() []RoomHeadUpdate {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.pending) == 0 {
		return nil
	}
	updates := make([]RoomHeadUpdate, 0, min(len(p.pending), headProjectorBatch))
	for roomID, update := range p.pending {
		updates = append(updates, update)
		delete(p.pending, roomID)
		p.pendingBytes -= roomHeadUpdateBytes(update)
		if len(updates) == headProjectorBatch {
			break
		}
	}
	return updates
}

func (p *roomHeadProjector) run() {
	defer close(p.done)
	for {
		select {
		case <-p.signal:
			p.flush()
		case <-p.stop:
			p.flush()
			return
		}
	}
}

func (p *roomHeadProjector) flush() {
	for {
		updates := p.takeBatch()
		if len(updates) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), headProjectorTimeout)
		err := p.db.UpdateRoomHeads(ctx, updates)
		cancel()
		if err == nil {
			continue
		}
		log.Error().Err(err).Int("rooms", len(updates)).Msg("project room heads failed")
		for _, update := range updates {
			p.enqueue(update)
		}
		timer := time.NewTimer(headProjectorRetry)
		select {
		case <-p.stop:
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (p *roomHeadProjector) close() {
	p.closeOnce.Do(func() { close(p.stop) })
	<-p.done
}
