package store

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

func encodeMessage(message Message) []byte {
	data := []byte{recordVersion}
	var id [16]byte
	putUUID(id[:], message.MsgID)
	data = append(data, id[:]...)
	putUUID(id[:], message.ClientMsgID)
	data = append(data, id[:]...)
	putUUID(id[:], message.SenderID)
	data = append(data, id[:]...)
	putUUID(id[:], message.RoomID)
	data = append(data, id[:]...)
	var number [8]byte
	putU64(number[:], message.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], message.ServerTime)
	data = append(data, number[:]...)
	if message.HasReply {
		data = append(data, 1)
		putUUID(id[:], message.ReplyToMsgID)
		data = append(data, id[:]...)
	} else {
		data = append(data, 0)
	}
	data = appendString(data, message.MsgType)
	data = appendBytes(data, message.Payload)
	return appendBytes(data, message.Ext)
}

func decodeMessage(data []byte) (Message, error) {
	if len(data) < 1 || data[0] != recordVersion {
		return Message{}, errors.New("invalid message record version")
	}
	d := decoder{data: data[1:]}
	message := Message{}
	var err error
	if message.MsgID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.ClientMsgID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.SenderID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.RoomID, err = d.uuid(); err != nil {
		return Message{}, err
	}
	if message.RoomSeq, err = d.u64(); err != nil {
		return Message{}, err
	}
	if message.ServerTime, err = d.i64(); err != nil {
		return Message{}, err
	}
	if d.pos >= len(d.data) {
		return Message{}, errors.New("invalid reply marker")
	}
	message.HasReply = d.data[d.pos] != 0
	d.pos++
	if message.HasReply {
		if message.ReplyToMsgID, err = d.uuid(); err != nil {
			return Message{}, err
		}
	}
	if message.MsgType, err = d.string(); err != nil {
		return Message{}, err
	}
	if message.Payload, err = d.bytes(); err != nil {
		return Message{}, err
	}
	if message.Ext, err = d.bytes(); err != nil || !d.done() {
		return Message{}, errors.New("invalid message record")
	}
	return message, nil
}

func encodeDedup(value Dedup) []byte {
	data := []byte{recordVersion}
	var id [16]byte
	putUUID(id[:], value.MsgID)
	data = append(data, id[:]...)
	var number [8]byte
	putU64(number[:], value.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], value.ServerTime)
	data = append(data, number[:]...)
	return append(data, value.PayloadSum[:]...)
}

func decodeDedup(data []byte) (Dedup, error) {
	if len(data) != 1+16+8+8+32 || data[0] != recordVersion {
		return Dedup{}, errors.New("invalid dedup record")
	}
	d := decoder{data: data[1:]}
	value := Dedup{}
	var err error
	if value.MsgID, err = d.uuid(); err != nil {
		return Dedup{}, err
	}
	if value.RoomSeq, err = d.u64(); err != nil {
		return Dedup{}, err
	}
	if value.ServerTime, err = d.i64(); err != nil {
		return Dedup{}, err
	}
	copy(value.PayloadSum[:], d.data[d.pos:])
	return value, nil
}

func (s *Store) WriteMessage(ctx context.Context, message Message) (Message, error) {
	if err := contextErr(ctx); err != nil {
		return Message{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var room Room
	if err := s.get([]byte(roomKey(message.RoomID)), func(data []byte) error {
		var err error
		room, err = decodeRoom(data)
		return err
	}); err != nil {
		return Message{}, err
	}
	digestInput := append([]byte(message.MsgType), 0)
	digestInput = append(digestInput, message.Payload...)
	digestInput = append(digestInput, 0)
	digestInput = append(digestInput, message.Ext...)
	if message.HasReply {
		digestInput = append(digestInput, message.ReplyToMsgID[:]...)
	}
	digest := sha256.Sum256(digestInput)
	var old Dedup
	if err := s.get([]byte(dedupKey(message.RoomID, message.SenderID, message.ClientMsgID)), func(data []byte) error {
		var err error
		old, err = decodeDedup(data)
		return err
	}); err == nil {
		if old.PayloadSum != digest {
			return Message{}, ErrConflict
		}
		message.MsgID, message.RoomSeq, message.ServerTime = old.MsgID, old.RoomSeq, old.ServerTime
		return message, nil
	} else if !errors.Is(err, ErrNotFound) {
		return Message{}, err
	}
	room.LastSeq++
	message.RoomSeq = room.LastSeq
	message.MsgID = uuid.NewV7()
	message.ServerTime = time.Now().UnixMicro()
	room.UpdatedAt = time.Now()
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.set(batch, roomKey(room.RoomID), encodeRoom(room)); err != nil {
		return Message{}, err
	}
	if err := s.set(batch, messageKey(message.RoomID, message.RoomSeq), encodeMessage(message)); err != nil {
		return Message{}, err
	}
	if err := s.set(
		batch,
		dedupKey(message.RoomID, message.SenderID, message.ClientMsgID),
		encodeDedup(
			Dedup{MsgID: message.MsgID, RoomSeq: message.RoomSeq, ServerTime: message.ServerTime, PayloadSum: digest},
		),
	); err != nil {
		return Message{}, err
	}
	if err := commit(batch); err != nil {
		return Message{}, fmt.Errorf("commit message: %w", err)
	}
	return message, nil
}

func (s *Store) Messages(ctx context.Context, roomID uuid.UUID, before uint64, limit int) (MessagePage, error) {
	if err := contextErr(ctx); err != nil {
		return MessagePage{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	prefix := []byte(messagePrefix(roomID))
	upperBound := messageUpperBound(prefix, before)
	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
	if err != nil {
		return MessagePage{}, err
	}
	defer iter.Close()

	messages := make([]Message, 0, limit)
	for iter.Last(); iter.Valid() && len(messages) <= limit; iter.Prev() {
		value := append([]byte(nil), iter.Value()...)
		message, err := decodeMessage(value)
		if err != nil {
			return MessagePage{}, err
		}
		messages = append(messages, message)
	}
	if err := iter.Error(); err != nil {
		return MessagePage{}, err
	}
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}
	return MessagePage{Messages: messages, HasMore: hasMore}, nil
}

func messageUpperBound(prefix []byte, before uint64) []byte {
	if before == 0 {
		upperBound := append([]byte(nil), prefix...)
		for i := len(upperBound) - 1; i >= 0; i-- {
			if upperBound[i] < 0xff {
				upperBound[i]++
				return upperBound[:i+1]
			}
		}
		return nil
	}
	upperBound := make([]byte, len(prefix)+8)
	copy(upperBound, prefix)
	binary.BigEndian.PutUint64(upperBound[len(prefix):], before)
	return upperBound
}
