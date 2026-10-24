package engine

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strconv"
	"uuid"
)

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

// Message is one committed room message. Payload and Ext are raw JSON
// carried opaquely; RoomSeq is the room's dense commit order and the only
// pagination cursor.
type Message struct {
	MsgID        uuid.UUID       `json:"msg_id"`
	ClientMsgID  uuid.UUID       `json:"client_msg_id"`
	SenderID     uuid.UUID       `json:"sender_id"`
	RoomID       uuid.UUID       `json:"room_id"`
	RoomSeq      uint64          `json:"room_seq"`
	ServerTime   int64           `json:"server_time"`
	ReplyToMsgID uuid.UUID       `json:"reply_to_msg_id,omitzero"`
	MsgType      MsgType         `json:"msg_type"`
	Payload      jsontext.Value  `json:"payload,omitzero"`
	Ext          jsontext.Value  `json:"ext,omitzero"`
	Reactions    []ReactionGroup `json:"reactions,omitempty"`
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

// ReactionGroup aggregates one emoji's reactions on a message.
type ReactionGroup struct {
	Emoji   string      `json:"emoji"`
	Count   int         `json:"count"`
	UserIDs []uuid.UUID `json:"user_ids"`
}

// MessagePage is one history page: messages in the page's order plus whether
// an older (or newer, for incremental pulls) page exists.
type MessagePage struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
}

// dedup is the idempotency record of room_id+sender_id+client_msg_id: the
// identity a retry replays and the digest a conflicting payload violates.
type dedup struct {
	MsgID      uuid.UUID
	RoomSeq    uint64
	ServerTime int64
	PayloadSum [32]byte
}

// dedupRecordLen is the fixed encoded size of a dedup record: UUID, seq,
// time, and digest.
const dedupRecordLen = 16 + 8 + 8 + 32

func encodeMessage(message Message) []byte {
	return appendMessage(nil, message)
}

// appendMessage encodes into dst so hot batch writes can reuse one buffer;
// Pebble copies the record on batch.Set.
func appendMessage(data []byte, message Message) []byte {
	var number [8]byte
	data = putUUID(data, message.MsgID)
	data = putUUID(data, message.ClientMsgID)
	data = putUUID(data, message.SenderID)
	data = putUUID(data, message.RoomID)
	putU64(number[:], message.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], message.ServerTime)
	data = append(data, number[:]...)
	if message.ReplyToMsgID != uuid.Nil() {
		data = append(data, 1)
		data = putUUID(data, message.ReplyToMsgID)
	} else {
		data = append(data, 0)
	}
	data = append(data, byte(message.MsgType))
	return appendBytes(appendBytes(data, message.Payload), message.Ext)
}

func decodeMessage(data []byte) (Message, error) {
	d := decoder{data: data}
	message := Message{
		MsgID:       d.uuid(),
		ClientMsgID: d.uuid(),
		SenderID:    d.uuid(),
		RoomID:      d.uuid(),
		RoomSeq:     d.u64(),
		ServerTime:  d.i64(),
	}
	if d.byte() == 1 {
		message.ReplyToMsgID = d.uuid()
	}
	message.MsgType = MsgType(d.byte())
	message.Payload = d.bytes()
	message.Ext = d.bytes()
	if err := d.check("message"); err != nil {
		return Message{}, err
	}
	return message, nil
}

// ownedMessage copies Message's variable-length fields out of a pebble-owned
// buffer. decodeMessage aliases the Get/iterator value, which is invalid once
// its closer is closed; callers that keep the Message beyond that (caches,
// responses) must hold an owned copy.
func ownedMessage(m Message) Message {
	m.Payload = jsontext.Value(bytes.Clone(m.Payload))
	m.Ext = jsontext.Value(bytes.Clone(m.Ext))
	return m
}

// appendDedup encodes into dst; the record is fixed-size, so batch writes
// pass a stack buffer and allocate nothing.
func appendDedup(data []byte, value dedup) []byte {
	var number [8]byte
	data = putUUID(data, value.MsgID)
	putU64(number[:], value.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], value.ServerTime)
	data = append(data, number[:]...)
	return append(data, value.PayloadSum[:]...)
}

func decodeDedup(data []byte) (dedup, error) {
	d := decoder{data: data}
	value := dedup{
		MsgID:      d.uuid(),
		RoomSeq:    d.u64(),
		ServerTime: d.i64(),
	}
	copy(value.PayloadSum[:], d.fixed(len(value.PayloadSum)))
	if err := d.check("dedup"); err != nil {
		return dedup{}, err
	}
	return value, nil
}
