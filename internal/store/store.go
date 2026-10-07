package store

import (
	"context"
	"encoding/binary"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

var (
	ErrNotFound      = errors.New("store: not found")
	ErrAlreadyExists = errors.New("store: already exists")
	ErrConflict      = errors.New("store: conflict")
	ErrClosed        = errors.New("store: closed")
	ErrRecallTimeout = errors.New("store: recall timeout")
	ErrForbidden     = errors.New("store: forbidden")
)

const (
	messageQueueSize   = 4096
	messageBatchSize   = 128
	messageBatchWindow = time.Millisecond
)

type messageWriteRequest struct {
	ctx      context.Context
	messages []Message
	result   chan []MessageWriteResult
}

type MessageWriteResult struct {
	Message Message
	Err     error
}

const roomShardCount = 64

type roomShard struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]Room
}

type Store struct {
	db *pebble.DB

	roomShards [roomShardCount]roomShard

	messageQueue chan messageWriteRequest
	closeSignal  chan struct{}
	writerDone   chan struct{}
	closeDone    chan struct{}
	closeOnce    sync.Once
	stateMu      sync.RWMutex
	closed       bool
	closeErr     error
}

func (s *Store) getRoomShard(id uuid.UUID) *roomShard {
	return &s.roomShards[id[15]%roomShardCount]
}

type User struct {
	UserID    uuid.UUID `json:"user_id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	CreatedAt time.Time `json:"created_at"`
}

const (
	ChatTypeSingle = "single"
	ChatTypeGroup  = "group"
)

type Room struct {
	RoomID         uuid.UUID `json:"room_id"`
	ChatType       string    `json:"chat_type"`
	Name           string    `json:"name"`
	AvatarURL      string    `json:"avatar_url"`
	Notice         string    `json:"notice"`
	SingleChatHash []byte    `json:"-"`
	LastSeq        uint64    `json:"last_seq"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

type Member struct {
	RoomID   uuid.UUID `json:"room_id"`
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	IsHidden bool      `json:"is_hidden"`
	IsMuted  bool      `json:"is_muted"`
	IsPinned bool      `json:"is_pinned"`
}

type Friend struct {
	UserID    uuid.UUID `json:"user_id"`
	FriendID  uuid.UUID `json:"friend_id"`
	Remark    string    `json:"remark"`
	CreatedAt time.Time `json:"created_at"`
}

type FriendApplication struct {
	FromUserID uuid.UUID `json:"from_user_id"`
	ToUserID   uuid.UUID `json:"to_user_id"`
	Greeting   string    `json:"greeting"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

type ConversationInfo struct {
	Room        Room     `json:"room"`
	Member      Member   `json:"member"`
	UnreadCount uint64   `json:"unread_count"`
	LastMessage *Message `json:"last_message,omitempty"`
}

type ReactionGroup struct {
	Emoji   string      `json:"emoji"`
	Count   int         `json:"count"`
	UserIDs []uuid.UUID `json:"user_ids"`
}

type DeviceInfo struct {
	Token     string    `json:"token"`
	Platform  string    `json:"platform"`
	UpdatedAt time.Time `json:"updated_at"`
}

type MsgType int8

const (
	MsgTypeText   MsgType = 1
	MsgTypeImage  MsgType = 2
	MsgTypeVideo  MsgType = 3
	MsgTypeFile   MsgType = 4
	MsgTypeSystem MsgType = 5
	MsgTypeRecall MsgType = 6
)

func (t MsgType) String() string {
	switch t {
	case MsgTypeText:
		return "text"
	case MsgTypeImage:
		return "image"
	case MsgTypeVideo:
		return "video"
	case MsgTypeFile:
		return "file"
	case MsgTypeSystem:
		return "system"
	case MsgTypeRecall:
		return "recall"
	default:
		return strconv.Itoa(int(t))
	}
}

func (t MsgType) Valid() bool {
	return t >= MsgTypeText && t <= MsgTypeRecall
}

func (t MsgType) MarshalJSONTo(enc *jsontext.Encoder) error {
	return json.MarshalEncode(enc, t.String())
}

func (t *MsgType) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var s string
	if err := json.UnmarshalDecode(dec, &s); err != nil {
		return err
	}
	parsed, err := ParseMsgType(s)
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

func ParseMsgType(value string) (MsgType, error) {
	for _, known := range []struct {
		name string
		typ  MsgType
	}{
		{"text", MsgTypeText},
		{"image", MsgTypeImage},
		{"video", MsgTypeVideo},
		{"file", MsgTypeFile},
		{"system", MsgTypeSystem},
		{"recall", MsgTypeRecall},
	} {
		if value == known.name {
			return known.typ, nil
		}
	}
	return 0, fmt.Errorf("invalid msg_type %q", value)
}

type Message struct {
	MsgID        uuid.UUID      `json:"msg_id"`
	ClientMsgID  uuid.UUID      `json:"client_msg_id"`
	SenderID     uuid.UUID      `json:"sender_id"`
	RoomID       uuid.UUID      `json:"room_id"`
	RoomSeq      uint64         `json:"room_seq"`
	ServerTime   int64          `json:"server_time"`
	ReplyToMsgID uuid.UUID      `json:"reply_to_msg_id,omitzero"`
	MsgType      MsgType        `json:"msg_type"`
	Payload      jsontext.Value `json:"payload,omitzero"`
	Ext          jsontext.Value `json:"ext,omitzero"`
}

func (m Message) MarshalJSONTo(enc *jsontext.Encoder) error {
	if len(m.Payload) == 0 {
		m.Payload = nil
	}
	if len(m.Ext) == 0 {
		m.Ext = nil
	}
	type plain Message
	return json.MarshalEncode(enc, plain(m))
}

type Dedup struct {
	MsgID      uuid.UUID
	RoomSeq    uint64
	ServerTime int64
	PayloadSum [32]byte
}

type RoomInfo struct {
	Room   Room
	Member Member
}

type MessagePage struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"hasMore"`
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	opts := &pebble.Options{
		Cache:                       pebble.NewCache(64 << 20),
		MemTableSize:                64 << 20,
		MemTableStopWritesThreshold: 4,
		MaxConcurrentCompactions:    func() int { return 4 },
		Levels:                      make([]pebble.LevelOptions, 7),
	}
	for i := range opts.Levels {
		opts.Levels[i].FilterPolicy = bloom.FilterPolicy(10)
		opts.Levels[i].BlockSize = 32 << 10
		opts.Levels[i].EnsureDefaults()
	}
	opts.EnsureDefaults()
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, fmt.Errorf("open pebble: %w", err)
	}
	store := &Store{
		db:           db,
		messageQueue: make(chan messageWriteRequest, messageQueueSize),
		closeSignal:  make(chan struct{}),
		writerDone:   make(chan struct{}),
		closeDone:    make(chan struct{}),
	}
	for i := range store.roomShards {
		store.roomShards[i].rooms = make(map[uuid.UUID]Room)
	}
	go store.runMessageWriter()
	return store, nil
}

func (s *Store) Close() error {
	if s == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.stateMu.Lock()
		s.closed = true
		close(s.closeSignal)
		s.stateMu.Unlock()
		<-s.writerDone
		s.closeErr = s.db.Close()
		close(s.closeDone)
	})
	<-s.closeDone
	return s.closeErr
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil {
		return ErrClosed
	}
	s.stateMu.RLock()
	closed := s.closed
	s.stateMu.RUnlock()
	if closed {
		return ErrClosed
	}
	if err := contextErr(ctx); err != nil {
		return err
	}
	_, closer, err := s.db.Get([]byte("healthz:ping"))
	if closer != nil {
		closer.Close()
	}
	if err != nil && !errors.Is(err, pebble.ErrNotFound) {
		return fmt.Errorf("pebble ping: %w", err)
	}
	return nil
}

// freeResultChans recycles per-request reply channels. Channels are put back
// only after their result has been received, so every handed-out channel is
// empty; a bounded free list avoids both the allocation and the type assertion.
var freeResultChans = make(chan chan []MessageWriteResult, 64)

func getResultChan() chan []MessageWriteResult {
	select {
	case channel := <-freeResultChans:
		return channel
	default:
		return make(chan []MessageWriteResult, 1)
	}
}

func putResultChan(channel chan []MessageWriteResult) {
	select {
	case freeResultChans <- channel:
	default: // free list full: let the collector reclaim the channel
	}
}

func (s *Store) WriteMessages(ctx context.Context, messages []Message) []MessageWriteResult {
	if len(messages) == 0 {
		return nil
	}
	failed := func(err error) []MessageWriteResult {
		results := make([]MessageWriteResult, len(messages))
		for i := range results {
			results[i] = MessageWriteResult{Message: messages[i], Err: err}
		}
		return results
	}
	result := getResultChan()
	defer putResultChan(result)
	request := messageWriteRequest{ctx: ctx, messages: messages, result: result}
	s.stateMu.RLock()
	if s.closed {
		s.stateMu.RUnlock()
		return failed(ErrClosed)
	}
	select {
	case s.messageQueue <- request:
		s.stateMu.RUnlock()
	case <-ctx.Done():
		s.stateMu.RUnlock()
		return failed(ctx.Err())
	}
	return <-result
}

func (s *Store) runMessageWriter() {
	defer close(s.writerDone)
	for {
		select {
		case request := <-s.messageQueue:
			s.commitMessageRequests(s.collectMessageRequests(request))
		case <-s.closeSignal:
			for {
				select {
				case request := <-s.messageQueue:
					s.commitMessageRequests(s.collectMessageRequests(request))
				default:
					return
				}
			}
		}
	}
}

func (s *Store) collectMessageRequests(first messageWriteRequest) []messageWriteRequest {
	requests := []messageWriteRequest{first}
	messageCount := len(first.messages)
	for messageCount < messageBatchSize {
		select {
		case request := <-s.messageQueue:
			requests = append(requests, request)
			messageCount += len(request.messages)
		default:
			return requests
		}
	}
	return requests
}

func (s *Store) commitMessageRequests(requests []messageWriteRequest) {
	total := 0
	for _, request := range requests {
		total += len(request.messages)
	}
	if total == 0 {
		return
	}
	// Single request: reuse the caller's slice, no copy.
	if len(requests) == 1 {
		request := &requests[0]
		results := make([]MessageWriteResult, len(request.messages))
		for index := range request.messages {
			results[index].Message = request.messages[index]
			if err := contextErr(request.ctx); err != nil {
				results[index].Err = err
			}
		}
		s.writeMessageBatch(request.messages, results)
		request.result <- results
		return
	}
	results := make([]MessageWriteResult, total)
	messages := make([]Message, total)
	offset := 0
	for i := range requests {
		request := &requests[i]
		for j := range request.messages {
			// Contexts that already died fail locally and leave a zero Message slot;
			// writeMessageBatch skips pre-errored entries, keeping the two slices aligned.
			if err := contextErr(request.ctx); err != nil {
				results[offset+j] = MessageWriteResult{Message: request.messages[j], Err: err}
				continue
			}
			messages[offset+j] = request.messages[j]
		}
		offset += len(request.messages)
	}
	s.writeMessageBatch(messages, results)
	offset = 0
	for i := range requests {
		requests[i].result <- results[offset : offset+len(requests[i].messages)]
		offset += len(requests[i].messages)
	}
}

func (s *Store) setBytes(batch *pebble.Batch, key, value []byte) error {
	if err := batch.Set(key, value, nil); err != nil {
		return fmt.Errorf("set: %w", err)
	}
	return nil
}

func (s *Store) Checkpoint(ctx context.Context, dir string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("create checkpoint parent: %w", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove old checkpoint: %w", err)
	}
	if err := s.db.Checkpoint(dir); err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	return nil
}

// getRecord fetches key and decodes it into a value of T.
func (s *Store) getRecord[T any](key []byte, decode func([]byte) (T, error)) (T, error) {
	var zero T
	value, closer, err := s.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return zero, ErrNotFound
	}
	if err != nil {
		return zero, err
	}
	defer closer.Close()
	return decode(value)
}

// scanPrefix scans all keys with the given prefix and decodes values into []T.
func (s *Store) scanPrefix[T any](prefix []byte, decode func([]byte) (T, error)) ([]T, error) {
	upper := prefixUpperBound(prefix)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var result []T
	for iter.First(); iter.Valid(); iter.Next() {
		item, err := decode(iter.Value())
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, iter.Error()
}

// exists reports whether key exists in the database.
func (s *Store) exists(key []byte) (bool, error) {
	_, closer, err := s.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	closer.Close()
	return true, nil
}

func commit(batch *pebble.Batch) error {
	return batch.Commit(&pebble.WriteOptions{Sync: true})
}

func contextErr(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func putUUID(dst []byte, value uuid.UUID) []byte { return append(dst, value[:]...) }

func getUUID(src []byte) (uuid.UUID, error) {
	if len(src) < 16 {
		return uuid.Nil(), errors.New("invalid uuid")
	}
	var value uuid.UUID
	copy(value[:], src[:16])
	return value, nil
}

func putU64(dst []byte, value uint64) { binary.BigEndian.PutUint64(dst, value) }

func getU64(src []byte) (uint64, error) {
	if len(src) < 8 {
		return 0, errors.New("invalid uint64")
	}
	return binary.BigEndian.Uint64(src[:8]), nil
}

func putI64(dst []byte, value int64) { binary.BigEndian.PutUint64(dst, uint64(value)) }

func appendBytes(dst, value []byte) []byte {
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	dst = append(dst, size[:]...)
	return append(dst, value...)
}

func appendString(dst []byte, value string) []byte { return appendBytes(dst, []byte(value)) }

func boolByte(v bool) byte {
	if v {
		return 1
	}
	return 0
}

type decoder struct {
	data []byte
	pos  int
}

func (d *decoder) bytes() ([]byte, error) {
	if d.pos+4 > len(d.data) {
		return nil, errors.New("invalid length")
	}
	size := int(binary.BigEndian.Uint32(d.data[d.pos:]))
	d.pos += 4
	if d.pos+size > len(d.data) {
		return nil, errors.New("invalid value length")
	}
	value := d.data[d.pos : d.pos+size]
	d.pos += size
	return value, nil
}

func (d *decoder) string() (string, error) {
	value, err := d.bytes()
	return string(value), err
}

func (d *decoder) uuid() (uuid.UUID, error) {
	if d.pos+16 > len(d.data) {
		return uuid.Nil(), errors.New("invalid uuid")
	}
	value, err := getUUID(d.data[d.pos : d.pos+16])
	d.pos += 16
	return value, err
}

func (d *decoder) u64() (uint64, error) {
	if d.pos+8 > len(d.data) {
		return 0, errors.New("invalid uint64")
	}
	value, err := getU64(d.data[d.pos : d.pos+8])
	d.pos += 8
	return value, err
}

func (d *decoder) i64() (int64, error) {
	value, err := d.u64()
	return int64(value), err
}

func (d *decoder) done() bool { return d.pos == len(d.data) }

// byte consumes one trailing byte, returning 0 when the record is exhausted.
func (d *decoder) byte() byte {
	if d.pos >= len(d.data) {
		return 0
	}
	value := d.data[d.pos]
	d.pos++
	return value
}
