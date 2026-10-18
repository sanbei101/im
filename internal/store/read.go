package store

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"uuid"

	"github.com/cockroachdb/pebble"
)

// ReadSeq returns the user's read cursor for one room.
func (s *Store) ReadSeq(ctx context.Context, roomID, userID uuid.UUID) (uint64, error) {
	if err := contextErr(ctx); err != nil {
		return 0, err
	}
	key := appendReadKey(nil, roomID, userID)
	return s.readCursor(key)
}

func (s *Store) readCursor(key []byte) (uint64, error) {
	value, closer, err := s.db.Get(key)
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
func (s *Store) MarkRead(ctx context.Context, roomID, userID uuid.UUID, readSeq uint64) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	lock := s.getReadLock(roomID, userID)
	lock.Lock()
	defer lock.Unlock()

	key := appendReadKey(nil, roomID, userID)
	current, err := s.readCursor(key)
	if err == nil && current >= readSeq {
		return nil
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, key, binary.BigEndian.AppendUint64(nil, readSeq)); err != nil {
		return err
	}
	if err := commitReadCursor(batch); err != nil {
		return fmt.Errorf("commit read cursor: %w", err)
	}
	return nil
}

// ReadUsers lists the members whose cursor reached messageSeq. Results follow
// the member order so callers get deterministic output.
func (s *Store) ReadUsers(
	ctx context.Context,
	roomID uuid.UUID,
	messageSeq uint64,
	members []Member,
) ([]uuid.UUID, error) {
	if len(members) == 0 {
		return nil, nil
	}
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	prefix := appendReadPrefix(nil, roomID)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: prefix,
		UpperBound: prefixUpperBound(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("scan read cursors: %w", err)
	}
	defer iter.Close()

	byUser := make(map[uuid.UUID]uint64, len(members))
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

	result := make([]uuid.UUID, 0, len(members))
	for _, member := range members {
		if seq, ok := byUser[member.UserID]; ok && seq >= messageSeq {
			result = append(result, member.UserID)
		}
	}
	return result, nil
}

// DeleteRead removes one user's cursor when they leave a room.
func (s *Store) DeleteRead(ctx context.Context, roomID, userID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := batch.Delete(appendReadKey(nil, roomID, userID), nil); err != nil {
		return fmt.Errorf("delete read cursor: %w", err)
	}
	return commit(batch)
}

// DeleteRoomReads removes every cursor when a room is dissolved.
func (s *Store) DeleteRoomReads(ctx context.Context, roomID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	prefix := appendReadPrefix(nil, roomID)
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := batch.DeleteRange(prefix, prefixUpperBound(prefix), nil); err != nil {
		return fmt.Errorf("delete room read cursors: %w", err)
	}
	return commit(batch)
}
