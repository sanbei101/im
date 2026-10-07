package store

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/cockroachdb/pebble"
)

const (
	FriendStatusPending  = "pending"
	FriendStatusAccepted = "accepted"
	FriendStatusRejected = "rejected"
)

func encodeFriend(f Friend) []byte {
	var timestamp [8]byte
	putI64(timestamp[:], f.CreatedAt.UnixMicro())
	data := make([]byte, 0, 16*2+8+4+len(f.Remark))
	data = putUUID(data, f.UserID)
	data = putUUID(data, f.FriendID)
	data = append(data, timestamp[:]...)
	return appendString(data, f.Remark)
}

func decodeFriend(data []byte) (Friend, error) {
	d := decoder{data: data}
	userID, err := d.uuid()
	if err != nil {
		return Friend{}, err
	}
	friendID, err := d.uuid()
	if err != nil {
		return Friend{}, err
	}
	micros, err := d.i64()
	if err != nil {
		return Friend{}, err
	}
	remark, err := d.string()
	if err != nil || !d.done() {
		return Friend{}, errors.New("invalid friend record")
	}
	return Friend{
		UserID:    userID,
		FriendID:  friendID,
		Remark:    remark,
		CreatedAt: time.UnixMicro(micros),
	}, nil
}

func encodeApplication(a FriendApplication) []byte {
	var timestamp [8]byte
	putI64(timestamp[:], a.CreatedAt.UnixMicro())
	data := make([]byte, 0, 16*2+8+4+len(a.Greeting)+4+len(a.Status))
	data = putUUID(data, a.FromUserID)
	data = putUUID(data, a.ToUserID)
	data = append(data, timestamp[:]...)
	data = appendString(data, a.Greeting)
	return appendString(data, a.Status)
}

func decodeApplication(data []byte) (FriendApplication, error) {
	d := decoder{data: data}
	from, err := d.uuid()
	if err != nil {
		return FriendApplication{}, err
	}
	to, err := d.uuid()
	if err != nil {
		return FriendApplication{}, err
	}
	micros, err := d.i64()
	if err != nil {
		return FriendApplication{}, err
	}
	greeting, err := d.string()
	if err != nil {
		return FriendApplication{}, err
	}
	status, err := d.string()
	if err != nil || !d.done() {
		return FriendApplication{}, errors.New("invalid application record")
	}
	return FriendApplication{
		FromUserID: from,
		ToUserID:   to,
		Greeting:   greeting,
		Status:     status,
		CreatedAt:  time.UnixMicro(micros),
	}, nil
}

func (s *Store) ApplyFriend(ctx context.Context, fromID, toID uuid.UUID, greeting string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if fromID == toID {
		return errors.New("cannot apply to add yourself as friend")
	}

	var buf [64]byte
	friendKey := appendFriendKey(buf[:0], fromID, toID)
	exists, err := s.exists(friendKey)
	if err != nil {
		return fmt.Errorf("check friendship: %w", err)
	}
	if exists {
		return ErrAlreadyExists
	}

	appKey := appendApplicationKey(buf[:0], toID, fromID)
	app := FriendApplication{
		FromUserID: fromID,
		ToUserID:   toID,
		Greeting:   greeting,
		Status:     FriendStatusPending,
		CreatedAt:  time.Now(),
	}

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, appKey, encodeApplication(app)); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) AuditFriend(ctx context.Context, toID, fromID uuid.UUID, accept bool) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var buf [64]byte
	appKey := appendApplicationKey(buf[:0], toID, fromID)
	app, err := s.getRecord(appKey, decodeApplication)
	if err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	now := time.Now()
	if accept {
		app.Status = FriendStatusAccepted
		f1 := Friend{UserID: toID, FriendID: fromID, CreatedAt: now}
		f2 := Friend{UserID: fromID, FriendID: toID, CreatedAt: now}

		var fBuf1 [64]byte
		var fBuf2 [64]byte
		if err := s.setBytes(batch, appendFriendKey(fBuf1[:0], toID, fromID), encodeFriend(f1)); err != nil {
			return err
		}
		if err := s.setBytes(batch, appendFriendKey(fBuf2[:0], fromID, toID), encodeFriend(f2)); err != nil {
			return err
		}
	} else {
		app.Status = FriendStatusRejected
	}

	if err := s.setBytes(batch, appKey, encodeApplication(app)); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) Applications(ctx context.Context, toID uuid.UUID) ([]FriendApplication, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	var buf [32]byte
	return s.scanPrefix(appendApplicationPrefix(buf[:0], toID), decodeApplication)
}

func (s *Store) Friends(ctx context.Context, userID uuid.UUID) ([]Friend, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	var buf [32]byte
	return s.scanPrefix(appendFriendPrefix(buf[:0], userID), decodeFriend)
}

func (s *Store) DeleteFriend(ctx context.Context, userID, friendID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	batch := s.db.NewBatch()
	defer batch.Close()

	var buf1 [64]byte
	var buf2 [64]byte
	if err := batch.Delete(appendFriendKey(buf1[:0], userID, friendID), nil); err != nil {
		return err
	}
	if err := batch.Delete(appendFriendKey(buf2[:0], friendID, userID), nil); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) UpdateFriendRemark(ctx context.Context, userID, friendID uuid.UUID, remark string) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var buf [64]byte
	key := appendFriendKey(buf[:0], userID, friendID)
	friend, err := s.getRecord(key, decodeFriend)
	if err != nil {
		return err
	}

	friend.Remark = remark
	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, key, encodeFriend(friend)); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) AddBlacklist(ctx context.Context, userID, targetID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	if userID == targetID {
		return errors.New("cannot blacklist yourself")
	}

	var timestamp [8]byte
	putI64(timestamp[:], time.Now().UnixMicro())

	var buf [64]byte
	key := appendBlacklistKey(buf[:0], userID, targetID)

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := s.setBytes(batch, key, timestamp[:]); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) RemoveBlacklist(ctx context.Context, userID, targetID uuid.UUID) error {
	if err := contextErr(ctx); err != nil {
		return err
	}

	var buf [64]byte
	key := appendBlacklistKey(buf[:0], userID, targetID)

	batch := s.db.NewBatch()
	defer batch.Close()
	if err := batch.Delete(key, nil); err != nil {
		return err
	}
	return commit(batch)
}

func (s *Store) Blacklist(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}

	var buf [32]byte
	prefix := appendBlacklistPrefix(buf[:0], userID)
	upper := prefixUpperBound(prefix)

	iter, err := s.db.NewIter(&pebble.IterOptions{LowerBound: prefix, UpperBound: upper})
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	var targets []uuid.UUID
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()
		// key: 'b' (1) + userID (16) + targetID (16)
		if len(key) >= 33 {
			id, err := getUUID(key[17:33])
			if err == nil {
				targets = append(targets, id)
			}
		}
	}
	return targets, iter.Error()
}

func (s *Store) IsBlacklisted(ctx context.Context, userID, targetID uuid.UUID) (bool, error) {
	if err := contextErr(ctx); err != nil {
		return false, err
	}

	var buf [64]byte
	return s.exists(appendBlacklistKey(buf[:0], userID, targetID))
}
