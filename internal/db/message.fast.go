package db

import (
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"sync"
	"unsafe"

	"github.com/google/uuid"
)

var MessagePool = sync.Pool{
	New: func() any {
		return &Message{}
	},
}

func AcquireMessage() *Message {
	return MessagePool.Get().(*Message)
}

func ReleaseMessage(m *Message) {
	m.Reset()
	MessagePool.Put(m)
}

func (m *Message) Reset() {
	m.MsgID = uuid.UUID{}
	m.ClientMsgID = uuid.UUID{}
	m.SenderID = uuid.UUID{}
	m.RoomID = uuid.UUID{}
	m.ServerTime = 0
	m.ReplyToMsgID = nil
	m.MsgType = ""
	if m.Payload != nil {
		m.Payload = m.Payload[:0]
	}
	if m.Ext != nil {
		m.Ext = m.Ext[:0]
	}
}

func (m *Message) Marshal() ([]byte, error) {
	size := 16*4 + 8 + 1

	if m.ReplyToMsgID != nil {
		size += 16
	}
	size += 2 + len(m.MsgType)
	size += 4 + len(m.Payload)
	size += 4 + len(m.Ext)

	buf := make([]byte, size)
	offset := 0
	copy(buf[offset:], m.MsgID[:])
	offset += 16
	copy(buf[offset:], m.ClientMsgID[:])
	offset += 16
	copy(buf[offset:], m.SenderID[:])
	offset += 16
	copy(buf[offset:], m.RoomID[:])
	offset += 16

	binary.BigEndian.PutUint64(buf[offset:], uint64(m.ServerTime))
	offset += 8

	if m.ReplyToMsgID != nil {
		buf[offset] = 1
		offset += 1
		copy(buf[offset:], (*m.ReplyToMsgID)[:])
		offset += 16
	} else {
		buf[offset] = 0
		offset += 1
	}
	binary.BigEndian.PutUint16(buf[offset:], uint16(len(m.MsgType)))
	offset += 2
	copy(buf[offset:], m.MsgType)
	offset += len(m.MsgType)

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(m.Payload)))
	offset += 4
	copy(buf[offset:], m.Payload)
	offset += len(m.Payload)

	binary.BigEndian.PutUint32(buf[offset:], uint32(len(m.Ext)))
	offset += 4
	copy(buf[offset:], m.Ext)
	offset += len(m.Ext)

	return buf, nil
}

func (m *Message) Unmarshal(data []byte) error {
	if len(data) < 83 {
		return errors.New("data too short")
	}
	offset := 0
	copy(m.MsgID[:], data[offset:offset+16])
	offset += 16
	copy(m.ClientMsgID[:], data[offset:offset+16])
	offset += 16
	copy(m.SenderID[:], data[offset:offset+16])
	offset += 16
	copy(m.RoomID[:], data[offset:offset+16])
	offset += 16

	m.ServerTime = int64(binary.BigEndian.Uint64(data[offset : offset+8]))
	offset += 8

	if data[offset] == 1 {
		offset += 1
		if m.ReplyToMsgID == nil {
			m.ReplyToMsgID = new(uuid.UUID)
		}
		copy((*m.ReplyToMsgID)[:], data[offset:offset+16])
		offset += 16
	} else {
		offset += 1
		m.ReplyToMsgID = nil
	}

	msgTypeLen := int(binary.BigEndian.Uint16(data[offset : offset+2]))
	offset += 2
	m.MsgType = MessageType(unsafe.String(unsafe.SliceData(data[offset:offset+msgTypeLen]), msgTypeLen))
	offset += msgTypeLen

	payloadLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	offset += 4
	if cap(m.Payload) >= payloadLen {
		m.Payload = m.Payload[:payloadLen]
	} else {
		m.Payload = make(jsontext.Value, payloadLen)
	}
	copy(m.Payload, data[offset:offset+payloadLen])
	offset += payloadLen

	extLen := int(binary.BigEndian.Uint32(data[offset : offset+4]))
	offset += 4
	if cap(m.Ext) >= extLen {
		m.Ext = m.Ext[:extLen]
	} else {
		m.Ext = make(jsontext.Value, extLen)
	}
	copy(m.Ext, data[offset:offset+extLen])
	offset += extLen

	return nil
}
