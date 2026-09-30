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
	var id [16]byte
	totalLen := 16*4 + 8 + 8 + 1 + 1 + 4 + len(message.Payload) + 4 + len(message.Ext)
	if message.ReplyToMsgID != uuid.Nil() {
		totalLen += 16
	}
	data := make([]byte, 0, totalLen)
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
	if message.ReplyToMsgID != uuid.Nil() {
		data = append(data, 1)
		putUUID(id[:], message.ReplyToMsgID)
		data = append(data, id[:]...)
	} else {
		data = append(data, 0)
	}
	data = append(data, byte(message.MsgType))
	return appendBytes(appendBytes(data, message.Payload), message.Ext)
}

func decodeMessage(data []byte) (Message, error) {
	d := decoder{data: data}
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
	hasReply := d.data[d.pos] != 0
	d.pos++
	if hasReply {
		if message.ReplyToMsgID, err = d.uuid(); err != nil {
			return Message{}, err
		}
	}
	if d.pos >= len(d.data) {
		return Message{}, errors.New("invalid msg_type")
	}
	message.MsgType = MsgType(d.data[d.pos])
	d.pos++
	if message.Payload, err = d.bytes(); err != nil {
		return Message{}, err
	}
	if message.Ext, err = d.bytes(); err != nil || !d.done() {
		return Message{}, errors.New("invalid message record")
	}
	return message, nil
}

func encodeDedup(value Dedup) []byte {
	var id [16]byte
	putUUID(id[:], value.MsgID)
	data := make([]byte, 0, 16+8+8+32)
	data = append(data, id[:]...)
	var number [8]byte
	putU64(number[:], value.RoomSeq)
	data = append(data, number[:]...)
	putI64(number[:], value.ServerTime)
	data = append(data, number[:]...)
	return append(data, value.PayloadSum[:]...)
}

func decodeDedup(data []byte) (Dedup, error) {
	if len(data) != 16+8+8+32 {
		return Dedup{}, errors.New("invalid dedup record")
	}
	d := decoder{data: data}
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
	results := s.WriteMessages(ctx, []Message{message})
	if len(results) == 0 {
		return Message{}, ErrClosed
	}
	return results[0].Message, results[0].Err
}

type dedupLookupKey struct {
	room   uuid.UUID
	sender uuid.UUID
	client uuid.UUID
}

func (s *Store) writeMessageBatch(messages []Message, results []MessageWriteResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	batch := s.db.NewBatch()
	defer batch.Close()
	rooms := make(map[uuid.UUID]Room)
	digests := make([][32]byte, len(messages))
	pending := make(map[dedupLookupKey]int, len(messages))

	var keyBuf [64]byte
	var digestBuf [256]byte

	for index, message := range messages {
		results[index].Message = message
		room, ok := rooms[message.RoomID]
		if !ok {
			room, ok = s.rooms[message.RoomID]
		}
		if !ok {
			if err := s.get([]byte(roomKey(message.RoomID)), func(data []byte) error {
				var err error
				room, err = decodeRoom(data)
				return err
			}); err != nil {
				results[index].Err = err
				continue
			}
		}
		rooms[message.RoomID] = room

		var digestInput []byte
		needLen := len(message.Payload) + len(message.Ext) + 19
		if needLen <= len(digestBuf) {
			digestInput = digestBuf[:0]
		} else {
			digestInput = make([]byte, 0, needLen)
		}
		digestInput = append(digestInput, byte(message.MsgType))
		digestInput = append(digestInput, message.Payload...)
		digestInput = append(digestInput, 0)
		digestInput = append(digestInput, message.Ext...)
		if message.ReplyToMsgID != uuid.Nil() {
			digestInput = append(digestInput, message.ReplyToMsgID[:]...)
		}
		digest := sha256.Sum256(digestInput)
		digests[index] = digest

		lookup := dedupLookupKey{room: message.RoomID, sender: message.SenderID, client: message.ClientMsgID}
		if previous, ok := pending[lookup]; ok {
			if digests[previous] != digest {
				results[index].Err = ErrConflict
			} else if results[previous].Err != nil {
				results[index].Err = results[previous].Err
			} else {
				results[index].Message = results[previous].Message
			}
			continue
		}

		dedupKeyBytes := appendDedupKey(keyBuf[:0], message.RoomID, message.SenderID, message.ClientMsgID)
		var old Dedup
		if err := s.get(dedupKeyBytes, func(data []byte) error {
			var err error
			old, err = decodeDedup(data)
			return err
		}); err == nil {
			if old.PayloadSum != digest {
				results[index].Err = ErrConflict
			} else {
				results[index].Message.MsgID, results[index].Message.RoomSeq, results[index].Message.ServerTime = old.MsgID, old.RoomSeq, old.ServerTime
			}
			pending[lookup] = index
			continue
		} else if !errors.Is(err, ErrNotFound) {
			results[index].Err = err
			pending[lookup] = index
			continue
		}

		room.LastSeq++
		results[index].Message.RoomSeq = room.LastSeq
		results[index].Message.MsgID = uuid.NewV7()
		results[index].Message.ServerTime = time.Now().UnixMicro()
		room.UpdatedAt = time.Now()
		rooms[message.RoomID] = room
		pending[lookup] = index

		msgKeyBytes := appendMessageKey(keyBuf[:0], message.RoomID, results[index].Message.RoomSeq)
		if err := s.setBytes(batch, msgKeyBytes, encodeMessage(results[index].Message)); err != nil {
			results[index].Err = err
			continue
		}
		dedupKeyBytes = appendDedupKey(keyBuf[:0], message.RoomID, message.SenderID, message.ClientMsgID)
		if err := s.setBytes(batch, dedupKeyBytes, encodeDedup(Dedup{
			MsgID: results[index].Message.MsgID, RoomSeq: results[index].Message.RoomSeq,
			ServerTime: results[index].Message.ServerTime, PayloadSum: digest,
		})); err != nil {
			results[index].Err = err
		}
	}
	for roomID, room := range rooms {
		s.rooms[roomID] = room
		roomKeyBytes := []byte(roomKey(roomID))
		if err := s.setBytes(batch, roomKeyBytes, encodeRoom(room)); err != nil {
			for index := range results {
				if results[index].Message.RoomID == roomID && results[index].Err == nil {
					results[index].Err = err
				}
			}
		}
	}
	if err := commit(batch); err != nil {
		for index := range results {
			if results[index].Err == nil {
				results[index].Err = fmt.Errorf("commit messages: %w", err)
			}
		}
	}
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

	var prefixBuf [32]byte
	prefix := appendMessagePrefix(prefixBuf[:0], roomID)
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
