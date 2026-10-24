package engine

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"uuid"

	"github.com/cockroachdb/pebble"
)

// ReadSeq returns the user's read cursor for one room.
func (e *Engine) ReadSeq(ctx context.Context, roomID, userID uuid.UUID) (uint64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	key := appendReadKey(nil, roomID, userID)
	return e.readCursor(key)
}

func (e *Engine) readCursor(key []byte) (uint64, error) {
	value, closer, err := e.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("load read cursor: %w", err)
	}
	defer closer.Close()
	if len(value) != 8 {
		return 0, errors.New("invalid read cursor")
	}
	return binary.BigEndian.Uint64(value), nil
}

// MarkRead advances a user's read cursor monotonically. The cursor is
// advisory: it is written without fsync because clients can report it again.
func (e *Engine) MarkRead(ctx context.Context, roomID, userID uuid.UUID, readSeq uint64) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	lock := e.getReadLock(roomID, userID)
	lock.Lock()
	defer lock.Unlock()

	key := appendReadKey(nil, roomID, userID)
	current, err := e.readCursor(key)
	if err == nil && current >= readSeq {
		return nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	batch := e.db.NewBatch()
	defer batch.Close()
	if err := e.setBytes(batch, key, binary.BigEndian.AppendUint64(nil, readSeq)); err != nil {
		return err
	}
	if err := commitReadCursor(batch); err != nil {
		return fmt.Errorf("commit read cursor: %w", err)
	}
	return nil
}

// ReadUsers lists the users whose cursor reached messageSeq. Results follow
// the input order so callers get deterministic output.
func (e *Engine) ReadUsers(
	ctx context.Context,
	roomID uuid.UUID,
	messageSeq uint64,
	userIDs []uuid.UUID,
) ([]uuid.UUID, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	prefix := appendReadPrefix(nil, roomID)
	iter, err := e.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: prefixUpperBound(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("scan read cursors: %w", err)
	}
	defer iter.Close()

	byUser := make(map[uuid.UUID]uint64, len(userIDs))
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) != 1+16+16 {
			continue
		}
		var userID uuid.UUID
		copy(userID[:], key[1+16:])
		value := iter.Value()
		if len(value) != 8 {
			return nil, errors.New("invalid read cursor")
		}
		byUser[userID] = binary.BigEndian.Uint64(value)
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterate read cursors: %w", err)
	}

	result := make([]uuid.UUID, 0, len(userIDs))
	for _, userID := range userIDs {
		if seq, ok := byUser[userID]; ok && seq >= messageSeq {
			result = append(result, userID)
		}
	}
	return result, nil
}

// DeleteRead removes one user's cursor when they leave a room.
func (e *Engine) DeleteRead(ctx context.Context, roomID, userID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	batch := e.db.NewBatch()
	defer batch.Close()
	if err := batch.Delete(appendReadKey(nil, roomID, userID), nil); err != nil {
		return fmt.Errorf("delete read cursor: %w", err)
	}
	return commit(batch)
}

// DeleteRoomReads removes every cursor when a room is dissolved.
func (e *Engine) DeleteRoomReads(ctx context.Context, roomID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	prefix := appendReadPrefix(nil, roomID)
	batch := e.db.NewBatch()
	defer batch.Close()
	if err := batch.DeleteRange(prefix, prefixUpperBound(prefix), nil); err != nil {
		return fmt.Errorf("delete room read cursors: %w", err)
	}
	return commit(batch)
}
