// Package engine is a self-contained message-plane storage engine: it keeps
// room-sequenced messages in Pebble, commits them through a single batching
// writer with room+sender+client idempotency, serves hot tails from memory,
// and freezes cold history into parquet files in a local archive directory.
//
// The engine knows nothing about room metadata or membership: a room exists
// here from its first committed message, and callers decide who may write,
// read, or recall. It also knows nothing about remote storage: the archive
// directory is plain local files, and shipping those files elsewhere is the
// caller's business. That boundary keeps the package reusable on its own — it
// depends on nothing from this repository.
package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
	"github.com/phuslu/log"
	"github.com/phuslu/lru"
)

var (
	ErrNotFound      = errors.New("engine: not found")
	ErrConflict      = errors.New("engine: conflict")
	ErrClosed        = errors.New("engine: closed")
	ErrForbidden     = errors.New("engine: forbidden")
	ErrRecallTimeout = errors.New("engine: recall timeout")
)

// Option tunes Open; every option has a production-ready default and refuses
// values that would degrade one.
type Option func(*Engine)

// RecallWindow bounds how long after ServerTime a non-admin may recall a
// message. Zero or negative keeps the default of two minutes.
func RecallWindow(d time.Duration) Option {
	return func(e *Engine) {
		if d > 0 {
			e.recallWindow = d
		}
	}
}

// ArchiveDir sets the directory the archiver exports parquet files to and
// cold reads serve them from. Without it the engine keeps everything in
// Pebble and refuses to archive.
func ArchiveDir(dir string) Option {
	return func(e *Engine) { e.archiveDir = dir }
}

// Logger receives archiver warnings. Nil keeps phuslu/log's default logger.
func Logger(l *log.Logger) Option {
	return func(e *Engine) {
		if l != nil {
			e.logger = l
		}
	}
}

const (
	messageQueueSize   = 4096
	messageBatchSize   = 512
	messageBatchWindow = time.Millisecond
	recallQueueSize    = 256
	roomShardCount     = 64
	readLockCount      = 256
)

// defaultRecallWindow is the message-recall window when RecallWindow is unset.
const defaultRecallWindow = 2 * time.Minute

// roomCachePerShard bounds the room head cache: 64 shards * 256 rooms.
// Evicted rooms are simply re-read from Pebble on demand.
const roomCachePerShard = 256

// archiveMarkRooms bounds the archive watermark cache. Entries never go
// stale: only this node's archiver moves a room's watermark, and it updates
// the cache in the same breath.
const archiveMarkRooms = 4096

// dedupCacheSize bounds the in-memory idempotency cache, enough to intercept
// seconds of network retries without reading Pebble.
const dedupCacheSize = 32768

type roomShard struct {
	rooms *lru.LRUCache[uuid.UUID, uint64]
}

// Engine is the message-plane storage engine. Every write — message batch or
// recall — is funneled through one writer goroutine that commits at most
// messageBatchSize messages per messageBatchWindow, so random small writes
// become sequential Pebble batches. Methods are safe for concurrent use
// until Close returns.
type Engine struct {
	db *pebble.DB

	roomShards   [roomShardCount]roomShard
	readLocks    [readLockCount]sync.Mutex
	dedupCache   *lru.LRUCache[dedupLookupKey, dedup]
	archiveMarks *lru.LRUCache[uuid.UUID, uint64]
	cache        *messageCache
	// The archive tier: parquet files under archiveDir, with the most
	// recently decoded files kept whole in memory.
	archiveDir   string
	decodedCache *lru.LRUCache[string, *decodedArchive]

	messageQueue chan messageWriteRequest
	recallQueue  chan *recallRequest
	// Encode scratches reused across batches. Only the single writer
	// goroutine touches them, so they need no locking.
	recordScratch []byte
	dedupScratch  [64]byte
	roomScratch   []byte
	digestScratch []byte

	recallWindow time.Duration
	logger       *log.Logger

	closeSignal chan struct{}
	writerDone  chan struct{}
	closeDone   chan struct{}
	closeOnce   sync.Once
	stateMu     sync.RWMutex
	closed      bool
	closeErr    error
}

// Open opens the Pebble store at path, starts the writer goroutine, and
// applies opts. The returned engine must be Closed exactly once.
func Open(path string, opts ...Option) (*Engine, error) {
	if path == "" {
		return nil, errors.New("engine path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create engine directory: %w", err)
	}
	pebbleOpts := &pebble.Options{
		Cache:                       pebble.NewCache(64 << 20),
		MemTableSize:                64 << 20,
		MemTableStopWritesThreshold: 4,
		MaxConcurrentCompactions:    func() int { return 4 },
		Levels:                      make([]pebble.LevelOptions, 7),
	}
	for i := range pebbleOpts.Levels {
		pebbleOpts.Levels[i].FilterPolicy = bloom.FilterPolicy(10)
		pebbleOpts.Levels[i].BlockSize = 32 << 10
		pebbleOpts.Levels[i].EnsureDefaults()
	}
	pebbleOpts.EnsureDefaults()
	db, err := pebble.Open(path, pebbleOpts)
	if err != nil {
		return nil, fmt.Errorf("open pebble: %w", err)
	}
	e := &Engine{
		db:           db,
		cache:        newMessageCache(),
		decodedCache: lru.NewLRUCache[string, *decodedArchive](decodedArchiveMaxFiles),
		messageQueue: make(chan messageWriteRequest, messageQueueSize),
		recallQueue:  make(chan *recallRequest, recallQueueSize),
		recallWindow: defaultRecallWindow,
		logger:       &log.DefaultLogger,
		closeSignal:  make(chan struct{}),
		writerDone:   make(chan struct{}),
		closeDone:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(e)
	}
	if e.archiveDir != "" {
		if err := os.MkdirAll(e.archiveDir, 0o755); err != nil {
			db.Close()
			return nil, fmt.Errorf("create archive directory: %w", err)
		}
	}
	for i := range e.roomShards {
		e.roomShards[i].rooms = lru.NewLRUCache[uuid.UUID, uint64](roomCachePerShard)
	}
	e.dedupCache = lru.NewLRUCache[dedupLookupKey, dedup](dedupCacheSize)
	e.archiveMarks = lru.NewLRUCache[uuid.UUID, uint64](
		archiveMarkRooms,
		lru.WithShards[uuid.UUID, uint64](64),
	)
	go e.runMessageWriter()
	return e, nil
}

func (e *Engine) getRoomShard(id uuid.UUID) *roomShard {
	return &e.roomShards[id[15]%roomShardCount]
}

func (e *Engine) getReadLock(roomID, userID uuid.UUID) *sync.Mutex {
	return &e.readLocks[int(roomID[15]^userID[15])%readLockCount]
}

func (e *Engine) Close() error {
	if e == nil {
		return nil
	}
	e.closeOnce.Do(func() {
		e.stateMu.Lock()
		e.closed = true
		close(e.closeSignal)
		e.stateMu.Unlock()
		<-e.writerDone
		e.closeErr = e.db.Close()
		close(e.closeDone)
	})
	<-e.closeDone
	return e.closeErr
}

func (e *Engine) Ping(ctx context.Context) error {
	if e == nil {
		return ErrClosed
	}
	e.stateMu.RLock()
	closed := e.closed
	e.stateMu.RUnlock()
	if closed {
		return ErrClosed
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	_, closer, err := e.db.Get([]byte("healthz:ping"))
	if closer != nil {
		closer.Close()
	}
	if err != nil && !errors.Is(err, pebble.ErrNotFound) {
		return fmt.Errorf("pebble ping: %w", err)
	}
	return nil
}

// Checkpoint writes a consistent on-disk snapshot of the Pebble store to dir,
// replacing any previous checkpoint there.
func (e *Engine) Checkpoint(ctx context.Context, dir string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("create checkpoint parent: %w", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove old checkpoint: %w", err)
	}
	if err := e.db.Checkpoint(dir); err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	return nil
}

// getRecord fetches key and decodes it into a value of T.
func (e *Engine) getRecord[T any](key []byte, decode func([]byte) (T, error)) (T, error) {
	var zero T
	value, closer, err := e.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return zero, ErrNotFound
	}
	if err != nil {
		return zero, err
	}
	defer closer.Close()
	return decode(value)
}

func (e *Engine) setBytes(batch *pebble.Batch, key, value []byte) error {
	if err := batch.Set(key, value, nil); err != nil {
		return fmt.Errorf("set: %w", err)
	}
	return nil
}

// commit persists a batch durably: callers answer their ACK only after it
// returns.
func commit(batch *pebble.Batch) error {
	return batch.Commit(&pebble.WriteOptions{Sync: true})
}

// commitReadCursor persists advisory read state without fsync. Losing a few
// seconds of read cursors on a crash is harmless; clients report them again.
func commitReadCursor(batch *pebble.Batch) error {
	return batch.Commit(&pebble.WriteOptions{Sync: false})
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
