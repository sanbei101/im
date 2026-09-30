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
	var number [8]byte
	totalLen := 16*4 + 8 + 8 + 1 + 1 + 4 + len(message.Payload) + 4 + len(message.Ext)
	if message.ReplyToMsgID != uuid.Nil() {
		totalLen += 16
	}
	data := make([]byte, 0, totalLen)
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
	if d.byte() == 1 {
		if message.ReplyToMsgID, err = d.uuid(); err != nil {
			return Message{}, err
		}
	}
	message.MsgType = MsgType(d.byte())
	if message.Payload, err = d.bytes(); err != nil {
		return Message{}, err
	}
	if message.Ext, err = d.bytes(); err != nil || !d.done() {
		return Message{}, errors.New("invalid message record")
	}
	return message, nil
}

func encodeDedup(value Dedup) []byte {
	var number [8]byte
	data := make([]byte, 0, 16+8+8+32)
	data = putUUID(data, value.MsgID)
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
	var msgKeyBuf [25]byte
	var digestBuf [256]byte

	for index := range messages {
		if results[index].Err != nil {
			continue
		}
		message := &messages[index]
		results[index].Message = *message
		room, ok := rooms[message.RoomID]
		if !ok {
			room, ok = s.rooms[message.RoomID]
		}
		if !ok {
			var err error
			if room, err = getRecord(s, appendRoomKey(keyBuf[:0], message.RoomID), decodeRoom); err != nil {
				results[index].Err = err
				continue
			}
		}
		rooms[message.RoomID] = room

		digestInput := digestBuf[:0]
		needLen := len(message.Payload) + len(message.Ext) + 19
		if needLen > len(digestBuf) {
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
			switch {
			case digests[previous] != digests[index]:
				results[index].Err = ErrConflict
			case results[previous].Err != nil:
				results[index].Err = results[previous].Err
			default:
				results[index].Message = results[previous].Message
			}
			continue
		}

		dedupKeyBytes := appendDedupKey(keyBuf[:0], message.RoomID, message.SenderID, message.ClientMsgID)
		pending[lookup] = index
		if old, err := getRecord(s, dedupKeyBytes, decodeDedup); err == nil {
			if old.PayloadSum != digests[index] {
				results[index].Err = ErrConflict
			} else {
				results[index].Message.MsgID, results[index].Message.RoomSeq, results[index].Message.ServerTime = old.MsgID, old.RoomSeq, old.ServerTime
			}
			continue
		} else if !errors.Is(err, ErrNotFound) {
			results[index].Err = err
			continue
		}

		room.LastSeq++
		results[index].Message.RoomSeq = room.LastSeq
		results[index].Message.MsgID = uuid.NewV7()
		results[index].Message.ServerTime = time.Now().UnixMicro()
		room.UpdatedAt = time.Now()
		rooms[message.RoomID] = room

		msgKey := appendMessageKey(msgKeyBuf[:0], message.RoomID, results[index].Message.RoomSeq)
		if err := s.setBytes(batch, msgKey, encodeMessage(results[index].Message)); err != nil {
			results[index].Err = err
			continue
		}
		if err := s.setBytes(batch, dedupKeyBytes, encodeDedup(Dedup{
			MsgID: results[index].Message.MsgID, RoomSeq: results[index].Message.RoomSeq,
			ServerTime: results[index].Message.ServerTime, PayloadSum: digests[index],
		})); err != nil {
			results[index].Err = err
		}
	}
	for roomID := range rooms {
		room := rooms[roomID]
		s.rooms[roomID] = room
		roomKeyBytes := appendRoomKey(keyBuf[:0], roomID)
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
