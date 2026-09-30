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
)

var (
	ErrNotFound      = errors.New("store: not found")
	ErrAlreadyExists = errors.New("store: already exists")
	ErrConflict      = errors.New("store: conflict")
	ErrClosed        = errors.New("store: closed")
)

const (
	recordVersion      byte = 1
	messageQueueSize        = 4096
	messageBatchSize        = 128
	messageBatchWindow      = time.Millisecond
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

type Store struct {
	db *pebble.DB
	mu sync.RWMutex

	messageQueue chan messageWriteRequest
	closeSignal  chan struct{}
	writerDone   chan struct{}
	closeDone    chan struct{}
	closeOnce    sync.Once
	stateMu      sync.RWMutex
	closed       bool
	closeErr     error
}

type User struct {
	UserID    uuid.UUID
	Username  string
	Password  string
	CreatedAt time.Time
}

type Room struct {
	RoomID         uuid.UUID
	ChatType       string
	Name           string
	AvatarURL      string
	SingleChatHash []byte
	LastSeq        uint64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Member struct {
	RoomID   uuid.UUID
	UserID   uuid.UUID
	Role     string
	IsHidden bool
	IsMuted  bool
}

type MsgType int8

const (
	MsgTypeText   MsgType = 1
	MsgTypeImage  MsgType = 2
	MsgTypeVideo  MsgType = 3
	MsgTypeFile   MsgType = 4
	MsgTypeSystem MsgType = 5
)

// String 返回对外 JSON/wire 使用的名称；未知值原样返回数字字符串。
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
	default:
		return strconv.Itoa(int(t))
	}
}

// Valid 报告是否为已知类型。
func (t MsgType) Valid() bool {
	return t >= MsgTypeText && t <= MsgTypeSystem
}

// ParseMsgType 解析客户端上行的 msg_type：名称或数字字符串均可，
// 未知值报错。名称解析失败时回退到数字，避免 "01" 之类的输入被拒。
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
	} {
		if value == known.name {
			return known.typ, nil
		}
	}
	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid msg_type %q", value)
	}
	typ := MsgType(number)
	if !typ.Valid() {
		return 0, fmt.Errorf("invalid msg_type %q", value)
	}
	return typ, nil
}

type Message struct {
	MsgID        uuid.UUID
	ClientMsgID  uuid.UUID
	SenderID     uuid.UUID
	RoomID       uuid.UUID
	RoomSeq      uint64
	ServerTime   int64
	ReplyToMsgID uuid.UUID
	HasReply     bool
	MsgType      MsgType
	Payload      jsontext.Value
	Ext          jsontext.Value
}

// messageJSON 是历史消息的对外契约：snake_case、msg_type 为名称、payload 内联 JSON。
// 空 Ext / 未设置 ReplyToMsgID 时省略字段，避免输出全零 UUID 或空字符串。
type messageJSON struct {
	MsgID        uuid.UUID      `json:"msg_id"`
	ClientMsgID  uuid.UUID      `json:"client_msg_id"`
	SenderID     uuid.UUID      `json:"sender_id"`
	RoomID       uuid.UUID      `json:"room_id"`
	RoomSeq      uint64         `json:"room_seq"`
	ServerTime   int64          `json:"server_time"`
	MsgType      string         `json:"msg_type"`
	ReplyToMsgID uuid.UUID      `json:"reply_to_msg_id,omitzero"`
	Payload      jsontext.Value `json:"payload,omitzero"`
	Ext          jsontext.Value `json:"ext,omitzero"`
}

// MarshalJSONTo 把存储记录编码为对外 JSON。HasReply 只是 ReplyToMsgID 的冗余标记，
// wire 上不输出。
func (m Message) MarshalJSONTo(enc *jsontext.Encoder) error {
	// jsontext.Value 是 []byte：nil 与空切片在 omitzero 下行为不同，
	// 空但非 nil 会被当成\"有值\"并产出空字符串 -> 非法 JSON。这里统一归一成 nil。
	var payload, ext jsontext.Value
	if len(m.Payload) > 0 {
		payload = m.Payload
	}
	if len(m.Ext) > 0 {
		ext = m.Ext
	}
	return json.MarshalEncode(enc, messageJSON{
		MsgID:        m.MsgID,
		ClientMsgID:  m.ClientMsgID,
		SenderID:     m.SenderID,
		RoomID:       m.RoomID,
		RoomSeq:      m.RoomSeq,
		ServerTime:   m.ServerTime,
		MsgType:      m.MsgType.String(),
		ReplyToMsgID: m.ReplyToMsgID,
		Payload:      payload,
		Ext:          ext,
	})
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
	Messages []Message
	HasMore  bool
}

func Open(path string) (*Store, error) {
	if path == "" {
		return nil, errors.New("store path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	db, err := pebble.Open(path, &pebble.Options{})
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

func (s *Store) WriteMessages(ctx context.Context, messages []Message) []MessageWriteResult {
	if len(messages) == 0 {
		return nil
	}
	queued := append([]Message(nil), messages...)
	result := make(chan []MessageWriteResult, 1)
	request := messageWriteRequest{ctx: ctx, messages: queued, result: result}
	s.stateMu.RLock()
	if s.closed {
		s.stateMu.RUnlock()
		results := make([]MessageWriteResult, len(messages))
		for i := range results {
			results[i].Err = ErrClosed
		}
		return results
	}
	select {
	case s.messageQueue <- request:
		s.stateMu.RUnlock()
	case <-ctx.Done():
		s.stateMu.RUnlock()
		results := make([]MessageWriteResult, len(messages))
		for i := range results {
			results[i].Err = ctx.Err()
		}
		return results
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
	timer := time.NewTimer(messageBatchWindow)
	defer timer.Stop()
	for messageCount < messageBatchSize {
		select {
		case request := <-s.messageQueue:
			requests = append(requests, request)
			messageCount += len(request.messages)
		case <-timer.C:
			return requests
		case <-s.closeSignal:
			for messageCount < messageBatchSize {
				select {
				case request := <-s.messageQueue:
					requests = append(requests, request)
					messageCount += len(request.messages)
				default:
					return requests
				}
			}
		}
	}
	return requests
}

func (s *Store) commitMessageRequests(requests []messageWriteRequest) {
	total := 0
	for _, request := range requests {
		total += len(request.messages)
	}
	results := make([]MessageWriteResult, total)
	messages := make([]Message, 0, total)
	resultIndex := 0
	for _, request := range requests {
		for _, message := range request.messages {
			result := &results[resultIndex]
			result.Message = message
			if err := contextErr(request.ctx); err != nil {
				result.Err = err
			} else {
				messages = append(messages, message)
			}
			resultIndex++
		}
	}
	if len(messages) > 0 {
		committed, errs := s.writeMessageBatch(messages)
		committedIndex := 0
		for i := range results {
			if results[i].Err != nil {
				continue
			}
			results[i] = MessageWriteResult{Message: committed[committedIndex], Err: errs[committedIndex]}
			committedIndex++
		}
	}
	resultIndex = 0
	for _, request := range requests {
		request.result <- results[resultIndex : resultIndex+len(request.messages)]
		resultIndex += len(request.messages)
	}
}

func (s *Store) Checkpoint(ctx context.Context, dir string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return fmt.Errorf("create checkpoint parent: %w", err)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove old checkpoint: %w", err)
	}
	if err := s.db.Checkpoint(dir); err != nil {
		return fmt.Errorf("create checkpoint: %w", err)
	}
	return nil
}

func (s *Store) get(key []byte, decode func([]byte) error) error {
	value, closer, err := s.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	defer closer.Close()
	return decode(value)
}

func (s *Store) set(batch *pebble.Batch, key string, value []byte) error {
	if err := batch.Set([]byte(key), value, nil); err != nil {
		return fmt.Errorf("set %q: %w", key, err)
	}
	return nil
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

func putUUID(dst []byte, value uuid.UUID) { copy(dst, value[:]) }

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
	if size < 0 || d.pos+size > len(d.data) {
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
