package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

// seqByMsgID resolves a message's seq through the msg-id index. The index
// survives archival precisely so messages stay addressable by id.
func (e *Engine) seqByMsgID(roomID, msgID uuid.UUID) (uint64, error) {
	var indexKeyBuf [msgIDIndexKeyLen]byte
	data, closer, err := e.db.Get(appendMsgIDIndexKey(indexKeyBuf[:0], roomID, msgID))
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	defer closer.Close()
	if len(data) != 8 {
		return 0, errors.New("invalid message index record")
	}
	return binary.BigEndian.Uint64(data), nil
}

// messageBySeq loads a message body by seq, transparently falling back to
// the archive when the local record has been archived away. The result is
// owned: it never aliases a pebble buffer.
func (e *Engine) messageBySeq(ctx context.Context, roomID uuid.UUID, seq uint64) (Message, error) {
	var msgKeyBuf [messageKeyLen]byte
	message, err := e.getRecord(appendMessageKey(msgKeyBuf[:0], roomID, seq), decodeMessage)
	if errors.Is(err, ErrNotFound) {
		archived, archivedErr := e.archivedMessage(ctx, roomID, seq)
		if archivedErr != nil {
			return Message{}, archivedErr
		}
		return archived, nil
	}
	if err != nil {
		return Message{}, err
	}
	return ownedMessage(message), nil
}

// Pins are room-scoped message bookmarks: P:<room>:<msg_id> -> operator and
// timestamp. The pin itself holds no message content; reads resolve through
// the msg-id index so pinned messages stay readable after archival.
func (e *Engine) PinMessage(ctx context.Context, roomID, msgID, operatorID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if _, err := e.seqByMsgID(roomID, msgID); err != nil {
		return err
	}

	batch := e.db.NewBatch()
	defer batch.Close()

	key := appendPinKey(nil, roomID, msgID)
	var val [24]byte
	copy(val[:16], operatorID[:])
	binary.BigEndian.PutUint64(val[16:], uint64(time.Now().Unix()))

	if err := e.setBytes(batch, key, val[:]); err != nil {
		return err
	}
	return commit(batch)
}

func (e *Engine) UnpinMessage(ctx context.Context, roomID, msgID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := e.db.NewBatch()
	defer batch.Close()

	if err := batch.Delete(appendPinKey(nil, roomID, msgID), nil); err != nil {
		return err
	}
	return commit(batch)
}

// PinnedMessages returns the room's pinned messages in pin-key order.
func (e *Engine) PinnedMessages(ctx context.Context, roomID uuid.UUID) ([]Message, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	prefix := appendPinPrefix(nil, roomID)
	upperBound := prefixUpperBound(prefix)
	iter, err := e.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upperBound})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var msgIDs []uuid.UUID
	prefixLen := len(prefix)
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) != prefixLen+16 {
			continue
		}
		var msgID uuid.UUID
		copy(msgID[:], key[prefixLen:])
		msgIDs = append(msgIDs, msgID)
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	messages := make([]Message, 0, len(msgIDs))
	for _, msgID := range msgIDs {
		seq, err := e.seqByMsgID(roomID, msgID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue // stale pin: the message is gone for good
			}
			return nil, err
		}
		message, err := e.messageBySeq(ctx, roomID, seq)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}
